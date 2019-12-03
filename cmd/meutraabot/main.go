package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	. "github.com/volatiletech/sqlboiler/queries/qm"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/irc"
	"gitlab.com/meutraa/meutraabot/pkg/models"
)

func main() {
	// Read our username from the environment, end if failure
	username := os.Getenv("TWITCH_USERNAME")
	oauth := os.Getenv("TWITCH_OAUTH_TOKEN")
	connString := os.Getenv("POSTGRES_CONNECTION_STRING")
	activeInterval, _ := strconv.ParseInt(os.Getenv("ACTIVE_INTERVAL"), 10, 64)

	if "" == username || "" == connString || 0 == activeInterval || "" == oauth {
		log.Println("Missing environment variable")
		return
	}

	// Create a connection to our database, end if failure
	db, err := data.Connection(connString, activeInterval)
	if nil != err {
		cleanup(db, nil, err)
	}

	// Create a channel for the OS to notify us of interrupts/signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Create a websocket connection to the IRC server, end if failure
	client, err := irc.NewClient()
	if nil != err {
		cleanup(db, client, err)
	}

	// Authenticate with our IRC connection, end if failure
	if err := client.Authenticate(username, oauth); nil != err {
		cleanup(db, client, err)
	}

	// Register a function for handling IRC private messages
	messages := make(chan *irc.PrivateMessage, 16)
	done := make(chan bool, 1)
	go client.SetMessageChannel(messages, done)

	// Join our own channel, if this fails, end program
	if err := client.JoinChannel("#" + username); nil != err {
		cleanup(db, client, err)
	}

	// Get a list of all our channel
	channels, err := models.Channels().All(db.Context, db.DB)
	if nil != err {
		log.Println(err)
	}

	// Try to join all channels in our channel table
	for _, channel := range channels {
		if err := client.JoinChannel(channel.ChannelName); nil != err {
			log.Println(err)
		}
	}

	for {
		select {
		case msg := <-messages:
			go handleMessage(client, db, msg, username)
		case <-done:
			cleanup(db, client, errors.New("Program is done"))
			return
		case <-interrupt:
			cleanup(db, client, errors.New("Interupt received"))
			return
		}
	}
}

func handleCommand(db *data.Database, client *irc.Client, botname, channel, sender, text string) string {
	switch strings.SplitN(text, " ", 2)[0] {
	case "!metrics", "!metric":
		return watchTimeHandler(db, channel, sender)
	case "!code":
		return fmt.Sprintf("%v lines of code", 460)
	case "!restart":
		return restartHandler(db, client, botname, sender)
	case "!leave":
		return leaveHandler(db, client, botname, channel, sender)
	case "!join":
		return joinHandler(db, client, botname, channel, sender)
	case "!version":
		return "1.5.0"
	case "!emoji":
		return emojiHandler(db, channel, sender, text)
	case "hey", "hello", "howdy", "hi",
		"hey!", "hello!", "howdy!", "hi!",
		"hey.", "hello.", "howdy.", "hi.":
		return randomGreeting() + " " + sender + "!"
	}
	if sender == botname {
		return ""
	}
	if msg, valid := sleepHandler(sender, text); valid {
		return msg
	}
	if msg, valid := backHandler(sender, text); valid {
		return msg
	}
	if msg, valid := vulpseHandler(db, channel, sender, text); valid {
		return msg
	}
	return ""
}

func vulpseHandler(db *data.Database, channel, sender, text string) (string, bool) {
	if channel != "#vulpesmusketeer" {
		return "", false
	}

	if text == "h" {
		channel, err := models.FindChannel(db.Context, db.DB, channel)
		if nil != err {
			log.Println("Unable to find channel", err)
			return "", true
		}

		channel.HiccupCount += 1

		err = channel.Update(db.Context, db.DB, boil.Whitelist(models.ChannelColumns.HiccupCount))
		if nil != err {
			log.Println("Unable to update hiccup_count", err)
		}
		return "", true
	}

	// Only commands for these guys
	if sender == "casweets" ||
		sender == "meutraa" ||
		sender == "vulpesmusketeer" ||
		sender == "biological" ||
		sender == "tristantwist_" {
		if len(text) >= 2 && text[:2] == "h " {
			countStr := text[2:]
			count, err := strconv.ParseInt(countStr, 10, 64)
			if nil != err {
				return "Unable to parse count!: " + err.Error(), true
			}

			channel, err := models.FindChannel(db.Context, db.DB, channel)
			if nil != err {
				log.Println("Unable to find channel", err)
				return "", true
			}

			channel.HiccupCount += count

			err = channel.Update(db.Context, db.DB, boil.Whitelist(models.ChannelColumns.HiccupCount))
			if nil != err {
				log.Println("Unable to update hiccup_count", err)
			}

			return "", true
		}
	}

	if strings.HasPrefix(text, "!hiccups") {
		channel, err := models.FindChannel(db.Context, db.DB, channel)
		if nil != err {
			log.Println("Unable to find channel", err)
			return "", true
		}
		return fmt.Sprintf("Casweets has hiccuped %v times!", channel.HiccupCount), true
	}
	return "", false
}

func emojiHandler(db *data.Database, channel, sender, text string) string {
	parts := strings.Split(text, " ")
	if len(parts) != 2 {
		return ""
	}

	// Check emoji is a valid character
	rune, _ := utf8.DecodeRuneInString(parts[1])
	if rune == utf8.RuneError {
		log.Println("Unable to parse rune from emoji")
		return ""
	}

	topUsers, err := models.Users(
		models.UserWhere.ChannelName.EQ(channel),
		OrderBy(models.UserColumns.WatchTime+" DESC"),
		Limit(3),
	).All(db.Context, db.DB)
	if nil != err {
		log.Println("Unable to get top 3 users", err)
		return ""
	}

	// Check user is in top 3
	var user *models.User
	for _, u := range topUsers {
		if u.Sender == sender {
			user = u
			break
		}
	}
	if nil == user {
		return "Must be in top 3 of leaderboard to use this command"
	}

	user.Emoji = null.String{String: string(rune), Valid: true}

	err = user.Update(db.Context, db.DB, boil.Whitelist(models.UserColumns.Emoji))
	if nil != err {
		log.Println("Unable to set Emoji for user", sender, ":", err)
	}
	return ""
}

func sleepHandler(sender, text string) (string, bool) {
	return "No sleep.", (strings.HasPrefix(text, "😴") ||
		strings.Contains(text, "sleep")) &&
		!strings.Contains(text, "no sleep") &&
		!strings.Contains(text, "not sleep")
}

func joinHandler(db *data.Database, client *irc.Client, botname, channel, sender string) string {
	if "#"+botname != channel {
		return ""
	}

	ch := models.Channel{ChannelName: "#" + sender}
	if err := ch.Upsert(db.Context, db.DB, false, nil, boil.Whitelist(), boil.Infer()); nil != err {
		log.Println("Unable to insert channel:", err)
		return ""
	}

	if err := client.JoinChannel(ch.ChannelName); nil != err {
		return "Failed to join channel"
	}

	if err := client.SendMessage(ch.ChannelName, "Hi 🙋"); nil != err {
		log.Println("Failed to send welcome message", err)
	}
	return "Bye bye 👋"
}

func leaveHandler(db *data.Database, client *irc.Client, botname, channel, sender string) string {
	if "#"+sender != channel {
		return ""
	}

	ch := models.Channel{ChannelName: channel}
	if err := ch.Delete(db.Context, db.DB); nil != err {
		return "Failed to leave channel"
	}

	go func() {
		time.Sleep(5 * time.Second)
		if err := client.PartChannel(channel); nil != err {
			log.Println("Failed to leave channel", channel, ":", err)
		}
	}()
	return "Bye bye 👋"
}

func watchTimeHandler(db *data.Database, channel, sender string) string {
	user, err := models.Users(
		Where(
			models.UserColumns.ChannelName+" = ? AND "+models.UserColumns.Sender+" = ?",
			channel, sender),
	).One(db.Context, db.DB)

	if nil != err {
		log.Println("Unable to lookup user for watch_time", err)
		return ""
	}

	return fmt.Sprintf("%v active time, %v messages, %v words",
		time.Duration(user.WatchTime*1000000000),
		user.MessageCount,
		user.WordCount)
}

func backHandler(sender, text string) (string, bool) {
	return "Hi back, I thought your name was " + sender + " 🤔.",
		strings.HasPrefix(text, "i'm back") ||
			strings.HasPrefix(text, "i am back") ||
			text == "back" ||
			strings.HasPrefix(text, "im back")
}

func restartHandler(db *data.Database, client *irc.Client, botname, sender string) string {
	if sender != botname {
		return ""
	}

	go func() {
		time.Sleep(5 * time.Second)
		cleanup(db, client, nil)
	}()
	return "Restarting in 5 seconds"
}

func handleMessage(client *irc.Client, db *data.Database, msg *irc.PrivateMessage, botname string) {
	// Insert a user if they do not exist
	u := models.User{
		ChannelName: msg.Channel,
		Sender:      msg.Sender,
	}
	if err := u.Upsert(db.Context, db.DB, false, nil, boil.Whitelist(), boil.Infer()); nil != err {
		log.Println("Unable to upsert user:", err)
	}

	// Save message
	message := models.Message{
		ChannelName: msg.Channel,
		Sender:      msg.Sender,
		Message:     msg.OriginalMessage,
	}
	if err := message.Insert(db.Context, db.DB, boil.Infer()); nil != err {
		log.Println(msg, err)
	}

	// Get the user to update stats
	user, err := models.Users(
		Where(
			models.UserColumns.ChannelName+" = ? AND "+models.UserColumns.Sender+" = ?",
			msg.Channel, msg.Sender),
	).One(db.Context, db.DB)

	if nil != err {
		log.Println("Unable to find user to update metrics", err)
	} else {
		// Update user metrics
		user.MessageCount += 1
		user.WordCount += int64(len(strings.Split(msg.Message, " ")))

		now := time.Now()
		if user.UpdatedAt.Valid {
			diff := now.Sub(user.UpdatedAt.Time)
			if diff.Seconds() < float64(db.ActiveInterval) {
				user.WatchTime += int64(diff.Seconds())
			}
		}

		if err := user.Update(db.Context, db.DB, boil.Whitelist(
			models.UserColumns.WatchTime,
			models.UserColumns.MessageCount,
			models.UserColumns.UpdatedAt,
			models.UserColumns.WordCount,
		)); nil != err {
			log.Println("Unable to update user metrics", err)
		}
	}

	res := handleCommand(db, client, botname, msg.Channel, msg.Sender, msg.Message)
	diff := time.Now().Sub(msg.ReceivedTime)
	log.Printf("[%v ms] %v:%v:%v < %v\n", diff.Milliseconds(), msg.Channel, msg.Sender, msg.OriginalMessage, res)
	if "" != res {
		client.SendMessage(msg.Channel, res)
	}
}

func randomGreeting() string {
	return [...]string{
		"Hello", "Howdy", "Salutations", "Greetings", "Hi", "Welcome", "Good day", "Hey",
	}[rand.Intn(7)]
}

// Function to cleanup database and irc client before closing
func cleanup(db *data.Database, client *irc.Client, err error) {
	if nil != err {
		log.Println(err)
	}
	if nil != db {
		if err := db.Close(); nil != err {
			log.Println("Unable to close database connection:", err)
		}
	}
	if nil != client {
		if err := client.Depart(); nil != err {
			log.Println("write close:", err)
		}
	}
	os.Exit(0)
}

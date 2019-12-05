package main

import (
	"database/sql"
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

	_ "github.com/lib/pq"
	ircevent "github.com/thoj/go-ircevent"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	. "github.com/volatiletech/sqlboiler/queries/qm"
	"gitlab.com/meutraa/meutraabot/pkg/models"
)

var irc *ircevent.Connection
var username string
var greetings = [...]string{
	"Hello", "Howdy", "Salutations", "Greetings",
	"Hi", "Welcome", "Good day", "Hey",
}
var colors = [...]string{
	"#ff9aa2", "#ffb7b2", "#ffdac1", "#e2f0cb", "#b5ead7", "#c7ceea",
}

const activeInterval = 600

func complain(err error) {
	if nil != err {
		log.Println(err)
	}
}

func main() {
	// Read our username from the environment, end if failure
	username = os.Getenv("TWITCH_USERNAME")
	oauth := os.Getenv("TWITCH_OAUTH_TOKEN")
	connString := os.Getenv("POSTGRES_CONNECTION_STRING")

	if "" == username || "" == connString || "" == oauth {
		log.Println("Missing environment variable")
		return
	}

	// Create a connection to our database, end if failure
	if db, err := sql.Open("postgres", connString); nil != err {
		log.Println("unable to establish connection to database", err)
		return
	} else {
		// boil.DebugMode = true
		boil.SetDB(db)
		defer func() {
			if err := db.Close(); nil != err {
				log.Println("Unable to close database connection:", err)
			}
		}()
	}

	// Create a channel for the OS to notify us of interrupts/signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	irc = ircevent.IRC(username, username)
	defer func() {
		if nil != irc {
			irc.Quit()
			irc.Disconnect()
		}
	}()
	irc.UseTLS = true
	irc.Password = oauth
	irc.AddCallback("001", func(e *ircevent.Event) {
		irc.Join("#" + username)
		// Get a list of all our channel
		if channels, err := models.Channels().AllG(); nil != err {
			log.Println(err)
		} else {
			// Try to join all channels in our channel table
			for _, channel := range channels {
				irc.Join(channel.ChannelName)
			}
		}
	})
	irc.AddCallback("PRIVMSG", func(e *ircevent.Event) {
		handleMessage(e, e.Arguments[0])
	})
	irc.AddCallback("PING", func(e *ircevent.Event) {
		irc.SendRaw("PONG :tmi.twitch.tv")
	})
	if err := irc.Connect("irc.chat.twitch.tv:6697"); nil != err {
		return
	}

	done := make(chan bool, 1)
	go func() {
		irc.Loop()
		done <- true
	}()

	for {
		select {
		case <-done:
			log.Println("program is done")
			return
		case <-interrupt:
			log.Println("interupt received")
			return
		}
	}
}

func handleCommand(channel, sender, text string) string {
	if username == sender { // This is the bot sending a message
		if text == "!restart" {
			go func() {
				time.Sleep(5 * time.Second)
				irc.Quit()
				irc.Disconnect()
				os.Exit(0)
			}()
			return "Restarting in 5 seconds"
		}
		return ""
	}
	if "#"+sender == channel { // This is the channel owner
		if text == "!leave" {
			if err := (&models.Channel{ChannelName: channel}).DeleteG(); nil != err {
				return "Failed to leave channel"
			}

			go func() {
				time.Sleep(1 * time.Second)
				irc.Part(channel)
			}()
			return "Bye bye 👋"
		}
	}
	switch text {
	case "hey", "hello", "howdy", "hi",
		"hey!", "hello!", "howdy!", "hi!",
		"hey.", "hello.", "howdy.", "hi.":
		return greetings[rand.Intn(len(greetings)-1)] + " " + sender + "!"
	case "!metrics", "!metric":
		return watchTimeHandler(channel, sender)
	case "!code":
		return fmt.Sprintf("%v lines of code", 297)
	case "!join": // A user can request meutbot join from any channel
		ch := models.Channel{ChannelName: "#" + sender}
		if err := ch.UpsertG(false, nil, boil.Whitelist(), boil.Infer()); nil != err {
			log.Println("Unable to insert channel:", err)
			return ""
		}

		irc.Join(ch.ChannelName)
		irc.Privmsg(ch.ChannelName, "Hi 🙋")
		return ""
	case "!version":
		return "1.6.3"
	}

	switch strings.SplitN(text, " ", 2)[0] {
	case "!emoji":
		return emojiHandler(channel, sender, text)
	}

	// I'm back functionality
	if strings.HasPrefix(text, "i'm back") ||
		strings.HasPrefix(text, "i am back") ||
		text == "back" ||
		strings.HasPrefix(text, "im back") {
		return "Hi back, I thought your name was " + sender + " 🤔."
	}
	if msg, valid := vulpseHandler(channel, sender, text); valid {
		return msg
	}
	return ""
}

func vulpseHandler(channel, sender, text string) (string, bool) {
	if channel != "#vulpesmusketeer" {
		return "", false
	}

	if (strings.HasPrefix(text, "😴") ||
		strings.Contains(text, "sleep")) &&
		!strings.Contains(text, "no sleep") &&
		!strings.Contains(text, "not sleep") {
		return "No sleep.", true
	}

	if text == "h" {
		channel, err := models.FindChannelG(channel)
		if nil != err {
			log.Println("Unable to find channel", err)
			return "", true
		}

		channel.HiccupCount += 1

		err = channel.UpdateG(boil.Whitelist(models.ChannelColumns.HiccupCount))
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
			count, err := strconv.ParseInt(text[2:], 10, 64)
			if nil != err {
				return "Unable to parse count!: " + err.Error(), true
			}

			if channel, err := models.FindChannelG(channel); nil != err {
				log.Println("Unable to find channel", err)
			} else {
				channel.HiccupCount += count
				complain(channel.UpdateG(boil.Whitelist(models.ChannelColumns.HiccupCount)))
			}
			return "", true
		}
	}

	if strings.HasPrefix(text, "!hiccups") {
		channel, err := models.FindChannelG(channel)
		if nil != err {
			log.Println("Unable to find channel", err)
			return "", true
		}
		return fmt.Sprintf("Casweets has hiccuped %v times!", channel.HiccupCount), true
	}
	return "", false
}

func emojiHandler(channel, sender, text string) string {
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
	).AllG()
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
	complain(user.UpdateG(boil.Whitelist(models.UserColumns.Emoji)))
	return ""
}

func watchTimeHandler(channel, sender string) string {
	if user, err := models.Users(
		Where(
			models.UserColumns.ChannelName+" = ? AND "+models.UserColumns.Sender+" = ?",
			channel, sender),
	).OneG(); nil != err {
		log.Println("Unable to lookup user for watch_time", err)
		return ""
	} else {
		return fmt.Sprintf("%v active time, %v messages, %v words",
			time.Duration(user.WatchTime*1000000000),
			user.MessageCount,
			user.WordCount)
	}
}

func handleMessage(e *ircevent.Event, channel string) {
	// Insert a user if they do not exist
	complain((&models.User{
		ChannelName: channel,
		Sender:      e.Nick,
	}).UpsertG(false, nil, boil.Whitelist(), boil.Infer()))

	// Save message
	complain((&models.Message{
		ChannelName: channel,
		Sender:      e.Nick,
		Message:     e.Message(),
	}).InsertG(boil.Infer()))

	// Get the user to update stats
	if user, err := models.Users(
		Where(
			models.UserColumns.ChannelName+" = ? AND "+models.UserColumns.Sender+" = ?",
			channel, e.Nick),
	).OneG(); nil != err {
		log.Println("Unable to find user to update metrics", err)
	} else {
		// Update user metrics
		user.MessageCount += 1
		user.WordCount += int64(len(strings.Split(e.Message(), " ")))
		if "" == user.TextColor.String {
			user.TextColor = null.String{String: colors[rand.Intn(len(colors)-1)], Valid: true}
		}

		if user.UpdatedAt.Valid {
			diff := time.Now().Sub(user.UpdatedAt.Time)
			if diff.Seconds() < float64(activeInterval) {
				user.WatchTime += int64(diff.Seconds())
			}
		}

		complain(user.UpdateG(boil.Infer()))
	}

	res := handleCommand(channel, e.Nick, strings.ToLower(e.Message()))
	log.Printf("%v:%v:%v < %v\n", channel, e.Nick, e.Message(), res)
	if "" != res {
		irc.Privmsg(channel, res)
	}
}

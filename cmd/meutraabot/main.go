package main

import (
	"context"
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
	mdb "gitlab.com/meutraa/meutraabot/pkg/db"
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

var q *(mdb.Queries)
var ctx = context.Background()

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
	db, err := sql.Open("postgres", connString)
	if nil != err {
		log.Println("unable to establish connection to database", err)
		return
	}
	if db.Ping() != nil {
		log.Println("unable to ping database", err)
		return
	}

	defer func() {
		if err := db.Close(); nil != err {
			log.Println("unable to close database connection:", err)
		}
	}()

	q = mdb.New(db)

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
		c, cancel := context.WithTimeout(ctx, time.Second*5)
		channels, err := q.GetChannelNames(c)
		cancel()
		if nil != err {
			log.Println(err)
		}
		// Try to join all channels in our channel table
		for _, channel := range channels {
			irc.Join(channel)
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
	c, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

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
			if err := q.DeleteChannel(c, channel); nil != err {
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
		if channel == "#hellokiwii" {
			return ""
		}
		return greetings[rand.Intn(len(greetings)-1)] + " " + sender + "!"
	case "!metrics", "!metric":
		metrics, err := q.GetMetrics(c, mdb.GetMetricsParams{channel, sender})
		if nil != err {
			log.Println("unable to lookup user for metrics", err)
			return ""
		}
		return fmt.Sprintf("%v active time, %v messages, %v words",
			time.Duration(metrics.WatchTime*1000000000),
			metrics.MessageCount,
			metrics.WordCount,
		)
	case "!code":
		return fmt.Sprintf("%v lines of code", 295)
	case "!join": // A user can request meutbot join from any channel
		cn := "#" + sender

		if err := q.CreateChannel(c, cn); nil != err {
			log.Println("Unable to insert channel:", err)
			return ""
		}

		irc.Join(cn)
		irc.Privmsg(cn, "Hi 🙋")
		return ""
	case "!version":
		return "1.7.0"
	}

	switch strings.SplitN(text, " ", 2)[0] {
	case "!emoji":
		return emojiHandler(c, channel, sender, text)
	}

	if msg, valid := vulpseHandler(c, channel, sender, text); valid {
		return msg
	}
	return ""
}

func vulpseHandler(c context.Context, channel, sender, text string) (string, bool) {
	if channel != "#vulpesmusketeer" {
		return "", false
	}

	// I'm back functionality
	if strings.HasPrefix(text, "i'm back") ||
		strings.HasPrefix(text, "i am back") ||
		text == "back" ||
		strings.HasPrefix(text, "im back") {
		return "Hi back, I thought your name was " + sender + " 🤔.", true
	}

	if (strings.HasPrefix(text, "😴") ||
		strings.Contains(text, "sleep")) &&
		!strings.Contains(text, "no sleep") &&
		!strings.Contains(text, "not sleep") {
		return "No sleep.", true
	}

	if text == "h" {
		if err := q.UpdateHiccupCount(c, mdb.UpdateHiccupCountParams{
			ChannelName: channel,
			HiccupCount: 1,
		}); nil != err {
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

			if err := q.UpdateHiccupCount(c, mdb.UpdateHiccupCountParams{
				ChannelName: channel,
				HiccupCount: count,
			}); nil != err {
				log.Println("Unable to update hiccup_count", err)
			}

			return "", true
		}
	}

	if strings.HasPrefix(text, "!hiccups") {
		count, err := q.GetHiccupCount(c, channel)
		if nil != err {
			log.Println("unable to find channel", err)
			return "", true
		}
		return fmt.Sprintf("Casweets has hiccuped %v times!", count), true
	}
	return "", false
}

func emojiHandler(c context.Context, channel, sender, text string) string {
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

	rank, err := q.GetWatchTimeRank(c, mdb.GetWatchTimeRankParams{
		ChannelName: channel,
		Sender:      sender,
	})
	if nil != err {
		log.Println("unable to get user watch time rank", err)
	}
	if rank > 3 {
		return "Must be in top 3 of leaderboard to use this command"
	}

	c2, cancel := context.WithTimeout(ctx, time.Second*5)
	if err := q.UpdateEmoji(c2, mdb.UpdateEmojiParams{
		ChannelName: channel,
		Sender:      sender,
		Emoji:       sql.NullString{String: string(rune), Valid: true},
	}); nil != err {
		log.Println("unable to update emoji", err)
		return ""
	}
	cancel()
	return ""
}

func handleMessage(e *ircevent.Event, channel string) {
	c, cancel := context.WithTimeout(ctx, time.Second*5)
	if err := q.CreateUser(c, mdb.CreateUserParams{
		ChannelName: channel,
		Sender:      e.Nick,
		TextColor:   sql.NullString{String: colors[rand.Intn(len(colors)-1)], Valid: true},
	}); nil != err {
		log.Println("unable to create user", err)
	}
	cancel()

	c, cancel = context.WithTimeout(ctx, time.Second*5)
	if err := q.UpdateMetrics(c, mdb.UpdateMetricsParams{
		ChannelName: channel,
		Sender:      e.Nick,
		WordCount:   int64(len(strings.Split(e.Message(), " "))),
	}); nil != err {
		log.Println("unable to update metrics for user", err)
	}
	cancel()

	c, cancel = context.WithTimeout(ctx, time.Second*5)
	if err := q.CreateMessage(c, mdb.CreateMessageParams{
		ChannelName: channel,
		Sender:      e.Nick,
		Message:     e.Message(),
	}); nil != err {
		log.Println("unable to add message", err)
	}
	cancel()

	res := handleCommand(channel, e.Nick, strings.ToLower(e.Message()))
	log.Printf("%v:%v:%v < %v\n", channel, e.Nick, e.Message(), res)
	if "" != res {
		irc.Privmsg(channel, res)
	}
}

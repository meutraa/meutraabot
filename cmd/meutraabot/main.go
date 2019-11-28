package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gitlab.com/meutraa/meutraabot/modules/back"
	"gitlab.com/meutraa/meutraabot/modules/greeting"
	"gitlab.com/meutraa/meutraabot/modules/management"
	"gitlab.com/meutraa/meutraabot/modules/messages"
	"gitlab.com/meutraa/meutraabot/modules/sleep"
	"gitlab.com/meutraa/meutraabot/modules/vulpesmusketeer"
	"gitlab.com/meutraa/meutraabot/modules/watchtime"
	"gitlab.com/meutraa/meutraabot/modules/words"
	"gitlab.com/meutraa/meutraabot/pkg/commands"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/irc"
)

func runFirst(client *irc.Client, db *data.Database, msg *irc.PrivateMessage, funcs ...commands.ResponseFunc) bool {
	for _, function := range funcs {
		res, valid, err := function(db, msg.Channel, msg.Sender, msg.Message)
		if valid {
			if "" != res {
				client.SendMessage(msg.Channel, res)
			}
			if nil != err && errors.Is(err, management.RestartError{}) {
				cleanup(db, client, err)
			}
			return true
		}
	}
	return false
}

func handleMessage(client *irc.Client, db *data.Database, msg *irc.PrivateMessage) {
	db.AddWatchTime(msg.Channel, msg.Sender)

	wordCount := len(strings.Split(msg.Message, " "))
	if err := db.AddToIntUserMetric(msg.Channel, msg.Sender, "word_count", int64(wordCount)); nil != err {
		log.Println(msg, "unable to update word_count for user:", err)
	}

	// Commands that do not contribute to message count
	if runFirst(client, db, msg,
		watchtime.Response,
		words.Response,
		messages.Response,
		management.CodeResponse,
		management.RestartResponse,
		management.VersionResponse,
		vulpesmusketeer.Response,
	) {
		return
	}

	// Update user metrics for word and message count
	if err := db.AddToIntUserMetric(msg.Channel, msg.Sender, "message_count", 1); nil != err {
		log.Println(msg, "unable to update message_count for user:", err)
	}

	// Responses that do count towards the message count
	if runFirst(client, db, msg,
		sleep.Response,
		greeting.Response,
		back.Response,
	) {
		return
	}
}

func cleanup(db *data.Database, client *irc.Client, err error) {
	if nil != err {
		log.Println(err)
	}
	if nil != db {
		log.Println("Cleaning up database")
		if err := db.Close(); nil != err {
			log.Println("Unable to close database connection:", err)
		}
	}
	if nil != client {
		log.Println("Cleaning up IRC client")
		if err := client.Depart(); nil != err {
			log.Println("write close:", err)
		}
	}
	os.Exit(0)
}

func main() {
	// Create a connection to our database, and auto migrate it
	db, err := data.Connection()
	if nil != err {
		cleanup(db, nil, err)
	}
	defer db.Close()

	// Get a count of channels
	channelCount, err := db.ChannelCount()
	if nil != err {
		cleanup(db, nil, err)
	}

	if 0 == channelCount {
		if err := db.Populate(); nil != err {
			cleanup(db, nil, err)
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Set up our irc client
	cred := commands.ReadEnv("TWITCH_OAUTH_TOKEN")
	client, err := irc.NewClient()
	if nil != err {
		cleanup(db, client, err)
	}
	if err := client.Authenticate(cred); nil != err {
		cleanup(db, client, err)
	}

	messages := make(chan *irc.PrivateMessage, 16)
	done := make(chan bool, 1)
	go client.SetMessageChannel(messages, done)

	channels, err := db.Channels()
	if nil != err {
		cleanup(db, client, err)
	}

	for _, channel := range channels {
		if err := client.JoinChannel(channel.ChannelName); nil != err {
			cleanup(db, client, err)
		}
	}

	for {
		select {
		case msg := <-messages:
			go handleMessage(client, db, msg)
		case <-done:
			cleanup(db, client, errors.New("Program is done"))
			return
		case <-interrupt:
			cleanup(db, client, errors.New("Interupt received"))
			return
		}
	}
}

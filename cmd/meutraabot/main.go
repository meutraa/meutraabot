package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/back"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/emoji"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/greeting"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/management"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/messages"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/sleep"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/vulpesmusketeer"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/watchtime"
	"gitlab.com/meutraa/meutraabot/cmd/meutraabot/modules/words"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/env"
	"gitlab.com/meutraa/meutraabot/pkg/irc"
)

type ResponseFunc = func(db *data.Database, text, channel, sender string) (string, bool, error)

func runFirst(client *irc.Client, db *data.Database, msg *irc.PrivateMessage, funcs ...ResponseFunc) bool {
	for _, function := range funcs {
		res, valid, err := function(db, msg.Channel, msg.Sender, msg.Message)
		if valid {
			if "" != res {
				client.SendMessage(msg.Channel, res)
			}
			if nil != err {
				if errors.Is(err, management.RestartError{}) {
					cleanup(db, client, err)
				}
				if errors.Is(err, management.PartError{}) {
					client.PartChannel(msg.Channel)
				}
			}
			return true
		}
	}
	return false
}

func handleMessage(client *irc.Client, db *data.Database, msg *irc.PrivateMessage) {
	// Save message
	if err := db.AddMessage(msg.Channel, msg.Sender, msg.OriginalMessage); nil != err {
		log.Println(msg, err)
	}

	db.UpdateMetrics(msg.Channel, msg.Sender, msg.Message)

	// Commands that do not contribute to message count
	if runFirst(client, db, msg,
		watchtime.Response,
		words.Response,
		messages.Response,
		emoji.Response,
		management.CodeResponse,
		management.RestartResponse,
		management.VersionResponse,
		vulpesmusketeer.Response,
		management.LeaveResponse,
	) {
		return
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

// Function to cleanup database and irc client before closing
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
	// Read our username from the environment, end if failure
	var username, oauth, connectionString string
	var activeInterval int64

	if !env.OauthToken(&oauth) ||
		!env.Username(&username) ||
		!env.ActiveInterval(&activeInterval) ||
		!env.PostgresConnectionString(&connectionString) {
		return
	}

	// Create a connection to our database, end if failure
	db, err := data.Connection(connectionString, activeInterval)
	if nil != err {
		cleanup(db, nil, err)
	}
	defer db.Close()

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
	channels, err := db.Channels()
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

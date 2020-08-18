package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/nicklaw5/helix"
	"github.com/pkg/errors"
	ircevent "github.com/thoj/go-ircevent"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Server struct {
	irc            *ircevent.Connection
	conn           *sql.DB
	send           chan Message
	twitch         *helix.Client
	q              *db.Queries
	activeInterval int
	env            *Environment
}

type Environment struct {
	twitchUsername           string
	twitchOauthToken         string
	twitchClientSecret       string
	twitchClientID           string
	postgresConnectionString string
}

func (s *Server) Close() {
	if nil != s.irc {
		s.irc.Quit()
		s.irc.Disconnect()
	}
	if nil != s.conn {
		s.conn.Close()
	}
}

func (s *Server) PrepareDatabase() error {
	conn, err := sql.Open("postgres", s.env.postgresConnectionString)
	if nil != err {
		return errors.Wrap(err, "unable to establish connection to database")
	}
	s.conn = conn

	if s.conn.Ping() != nil {
		return errors.Wrap(err, "unable to ping database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*5))
	defer cancel()
	queries, err := db.Prepare(ctx, conn)
	if nil != err {
		return errors.Wrap(err, "unable to prepare queries")
	}
	s.q = queries
	return nil
}

func (s *Server) PrepareTwitchClient() error {
	client, err := helix.NewClient(&helix.Options{
		ClientID:     s.env.twitchClientID,
		ClientSecret: s.env.twitchClientSecret,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create twitch api client")
	}
	s.twitch = client

	resp, err := s.twitch.GetAppAccessToken([]string{})
	if err != nil {
		return errors.Wrap(err, "unable to get app access token")
	}

	client, err = helix.NewClient(&helix.Options{
		ClientID:       s.env.twitchClientID,
		ClientSecret:   s.env.twitchClientSecret,
		AppAccessToken: resp.Data.AccessToken,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create twitch api client with app token")
	}
	s.twitch = client
	return nil
}

func (s *Server) PrepareIRC() (chan bool, error) {
	done := make(chan bool, 1)
	s.send = make(chan Message)

	s.irc = ircevent.IRC(s.env.twitchUsername, s.env.twitchUsername)
	s.irc.UseTLS = true
	s.irc.Password = s.env.twitchOauthToken
	s.irc.AddCallback("001", func(e *ircevent.Event) {
		s.irc.Join("#" + s.env.twitchUsername)
		c, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		// Get a list of all our channels
		channels, err := s.q.GetChannelNames(c)
		if nil != err {
			log.Println(err)
		}
		// Try to join all channels in our channel table
		for _, channel := range channels {
			s.irc.Join(channel)
		}
	})

	go func(c chan Message) {
		for msg := range c {
			fmt.Println("sending", msg.Channel, msg.Body)
			if strings.HasPrefix(msg.Body, "/") {
				s.irc.Privmsg(msg.Channel, msg.Body)
			} else {
				s.irc.Privmsg(msg.Channel, "/me "+msg.Body)
			}
			time.Sleep(time.Second * 2)
		}
	}(s.send)

	s.irc.AddCallback("PRIVMSG", func(e *ircevent.Event) {
		s.handleMessage(e)
	})
	s.irc.AddCallback("PING", func(e *ircevent.Event) {
		s.irc.SendRaw("PONG :tmi.twitch.tv")
	})
	if err := s.irc.Connect("irc.chat.twitch.tv:6697"); nil != err {
		return done, errors.Wrap(err, "unable to connect to irc")
	}

	s.irc.SendRaw("CAP REQ :twitch.tv/tags")

	go func() {
		s.irc.Loop()
		done <- true
	}()
	return done, nil
}

func (s *Server) ReadEnvironmentVariables() error {
	// Read our username from the environment, end if failure
	s.env = &Environment{}
	s.env.twitchOauthToken = os.Getenv("TWITCH_OAUTH_TOKEN")
	s.env.twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	s.env.twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	s.env.postgresConnectionString = os.Getenv("POSTGRES_CONNECTION_STRING")
	s.env.twitchUsername = os.Getenv("TWITCH_USERNAME")

	if "" == s.env.twitchUsername ||
		"" == s.env.postgresConnectionString ||
		"" == s.env.twitchClientSecret ||
		"" == s.env.twitchClientID ||
		"" == s.env.twitchOauthToken {
		return errors.New("missing environment variable")
	}
	return nil
}

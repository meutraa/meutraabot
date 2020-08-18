package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/nicklaw5/helix"
	"github.com/pkg/errors"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Server struct {
	irc            *irc.Client
	conn           *sql.DB
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

func (s *Server) OnChannels(ctx context.Context, onChannel func(channel string)) error {
	channels, err := s.q.GetChannelNames(ctx)
	if nil != err && err != sql.ErrNoRows {
		return errors.Wrap(err, "unable to get channels")
	}

	// Try to join all channels in our channel table
	for _, channel := range channels {
		onChannel(strings.TrimPrefix(channel, "#"))
	}
	return nil
}

func (s *Server) PrepareIRC() error {
	s.irc = irc.NewClient(s.env.twitchUsername, s.env.twitchOauthToken)
	s.irc.OnGlobalUserStateMessage(func(m irc.GlobalUserStateMessage) {
		s.irc.Join(s.env.twitchUsername)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		if err := s.OnChannels(ctx, func(channel string) {
			s.irc.Join(channel)
		}); nil != err {
			log.Println(err)
		}
	})

	s.irc.OnUserJoinMessage(func(m irc.UserJoinMessage) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		isBanned, err := s.q.IsUserBanned(ctx, m.User)
		if nil != err {
			log.Println("unable to check if user is banned", err)
			return
		}
		if isBanned {
			s.irc.Say(m.Channel, "/ban "+m.User)
		}
	})

	s.irc.OnPrivateMessage(s.handleMessage)

	return s.irc.Connect()
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

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	irc "github.com/gempir/go-twitch-irc/v3"
	"github.com/nicklaw5/helix/v2"
	"github.com/pkg/errors"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Server struct {
	irc     *irc.Client
	conn    *sql.DB
	twitch  *helix.Client
	q       *db.Queries
	env     *Environment
	history map[string][]string
}

type Environment struct {
	twitchUserID             string
	twitchOwnerID            string
	twitchOauthToken         string
	twitchClientSecret       string
	twitchClientID           string
	openaiKey                string
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

	resp, err := s.twitch.RequestAppAccessToken([]string{})
	if err != nil {
		return errors.Wrap(err, "unable to get app access token")
	}

	client.SetAppAccessToken(resp.Data.AccessToken)
	s.twitch = client

	return nil
}

func (s *Server) OnChannels(ctx context.Context, onChannel func(broadcaster helix.User)) error {
	channels, err := s.q.GetChannels(ctx)
	if nil != err && err != sql.ErrNoRows {
		return errors.Wrap(err, "unable to get channels")
	}

	broadcasters, err := Users(s.twitch, channels)
	if nil != err {
		return err
	}
	for _, broadcaster := range broadcasters {
		onChannel(broadcaster)
	}
	return nil
}

func (s *Server) JoinChannel(channel string) {
	s.irc.Join(channel)

	// Vet all users
	go func() {
		time.Sleep(10 * time.Second)
		users, err := s.irc.Userlist(channel)
		if nil != err {
			log(channel, "", "unable to get user list", err)
			return
		}
		bots, err := getBotList()
		if nil != err {
			log(channel, "", "unable to get known bot list", err)
		} else {
			for _, user := range users {
				go s.checkUser(bots, channel, user)
			}
		}
	}()
}

func (s *Server) PrepareIRC() error {
	self, err := User(s.twitch, s.env.twitchUserID)
	if nil != err {
		fmt.Println("unable to find user for id", s.env.twitchUserID)
		return err
	}

	s.irc = irc.NewClient(self.Login, s.env.twitchOauthToken)
	s.irc.Capabilities = append(s.irc.Capabilities, irc.MembershipCapability)

	fmt.Println("created client")
	s.irc.OnGlobalUserStateMessage(func(m irc.GlobalUserStateMessage) {
		// Get own user id
		s.irc.Join(self.Login)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		if err := s.OnChannels(ctx, func(broadcaster helix.User) {
			s.JoinChannel(broadcaster.Login)
		}); nil != err {
			fmt.Println(err)
		}
	})

	s.irc.OnUserJoinMessage(func(m irc.UserJoinMessage) {
		log(m.Channel, m.User, "joined", nil)
		go s.checkUser(nil, m.Channel, m.User)
	})

	s.irc.OnPrivateMessage(s.handleMessage)

	fmt.Println("connecting")
	return s.irc.Connect()
}

func (s *Server) checkUser(bots *BotsResponse, channel, username string) {
	if nil == bots {
		var err error
		bots, err = getBotList()
		if nil != err {
			log(channel, username, "unable to download known bot list", err)
			return
		}

	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Get channel and user id
	cUser, err := UserByName(s.twitch, channel)
	if nil != err {
		log(channel, username, "unable to get channel", err)
		return
	}

	qUser, err := UserByName(s.twitch, username)
	if nil != err {
		log(channel, username, "unable to get user", err)
		return
	}
	channelID := cUser.ID
	userID := qUser.ID

	approved, err := s.q.IsApproved(ctx, db.IsApprovedParams{
		ChannelID: channelID,
		UserID:    userID,
	})
	if nil != err && err != sql.ErrNoRows {
		log(channel, username, "unable to check approval status", err)
		return
	}
	if approved > 0 {
		log(channel, username, "already approved", nil)
		return
	}
	for _, bot := range bots.Bots {
		b, ok := bot[0].(string)
		if !ok {
			log(channel, username, "unable to cast bot name", nil)
			continue
		}
		if b != username {
			continue
		}

		log(channel, username, "banning as bot", nil)
		s.irc.Say(channel, "/ban "+username+" is a bot")
		return
	}

	// This user is not a bot
	log(channel, username, "not a bot", nil)
	if err := s.q.Approve(ctx, db.ApproveParams{
		ChannelID: channelID,
		UserID:    userID,
	}); nil != err {
		log(channel, username, "unable to approve", err)
		return
	}
}

func (s *Server) ReadEnvironmentVariables() error {
	// Read our username from the environment, end if failure
	s.env = &Environment{}
	s.env.twitchOauthToken = os.Getenv("TWITCH_OAUTH_TOKEN")
	s.env.twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	s.env.twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	s.env.postgresConnectionString = os.Getenv("POSTGRES_CONNECTION_STRING")
	s.env.openaiKey = os.Getenv("OPENAI_KEY")
	s.env.twitchUserID = os.Getenv("TWITCH_USER_ID")
	s.env.twitchOwnerID = os.Getenv("TWITCH_OWNER_ID")

	if s.env.twitchUserID == "" ||
		s.env.twitchOwnerID == "" ||
		s.env.postgresConnectionString == "" ||
		s.env.twitchClientSecret == "" ||
		s.env.twitchClientID == "" ||
		s.env.twitchOauthToken == "" ||
		s.env.openaiKey == "" {
		return errors.New("missing environment variable")
	}
	return nil
}

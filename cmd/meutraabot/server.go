package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/meutraa/helix"
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
	twitchOwner              string
	twitchOauthToken         string
	twitchClientSecret       string
	twitchClientID           string
	twitchSubListen          string
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

func (s *Server) OnChannels(ctx context.Context, onChannel func(channel string)) error {
	channels, err := s.q.GetChannelNames(ctx)
	if nil != err && err != sql.ErrNoRows {
		return errors.Wrap(err, "unable to get channels")
	}

	// Try to join all channels in our channel table
	for _, channel := range channels {
		onChannel(channel)
	}
	return nil
}

func (s *Server) JoinChannel(channel string) {
	s.irc.Join(channel)

	// Vet all users
	users, err := s.irc.Userlist(channel)
	if nil != err {
		log.Println("unable to get user list", err)
		return
	}
	for _, user := range users {
		go s.checkUser(channel, user)
	}

	go s.SubscribeToFollows(channel)
}

func (s *Server) UnsubscribeAll() {
	subs, err := s.twitch.GetWebhookSubscriptions(&helix.WebhookSubscriptionsParams{
		First: 100,
	})
	if nil != err {
		log.Println("unable to get existing subscriptions")
		return
	}
	log.Println("GetWebhookSubscriptions", subs.StatusCode)
	log.Println(len(subs.Data.WebhookSubscriptions), "existing subscriptions")

	for _, sub := range subs.Data.WebhookSubscriptions {
		res, err := s.twitch.PostWebhookSubscription(&helix.WebhookSubscriptionPayload{
			Callback:     sub.Callback,
			Mode:         "unsubscribe",
			Topic:        sub.Topic,
			LeaseSeconds: 86400, // one day, 10 day max
		})
		if err != nil {
			log.Println("unable to unsubscribe to channel follows", err)
		} else {
			log.Printf("code: %v, message: %v, error: %v\n", res.StatusCode, res.ErrorMessage, res.Error)
			log.Printf("unsubscribed from follow notifications %v %v\n", sub.Callback, sub.Topic)
		}
	}
}

func (s *Server) SubscribeToFollows(channel string) {
	u, err := User(s.twitch, channel)
	if nil != err {
		log.Printf("(%v) unable to get user: %v", channel, err)
		return
	}
	topic := "https://api.twitch.tv/helix/users/follows?first=1&to_id=" + u.ID
	log.Println("subscribing to", topic)

	res, err := s.twitch.PostWebhookSubscription(&helix.WebhookSubscriptionPayload{
		Callback:     "https://twitch.lost.host",
		Mode:         "subscribe",
		Topic:        topic,
		LeaseSeconds: 86400, // one day, 10 day max
		Secret:       "q1k1lnfKbIU3RQ20ligrtVv3vHAkPW51",
	})
	if err != nil {
		log.Println("unable to subscribe to channel follows", err)
	} else {
		log.Printf("code: %v, message: %v, error: %v\n", res.StatusCode, res.ErrorMessage, res.Error)
		log.Printf("(%v) subscribed to follow notifications\n", channel)
	}
}

func (s *Server) PrepareIRC() error {
	s.irc = irc.NewClient(s.env.twitchUsername, s.env.twitchOauthToken)
	s.irc.OnGlobalUserStateMessage(func(m irc.GlobalUserStateMessage) {
		s.irc.Join(s.env.twitchUsername)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		s.UnsubscribeAll()
		if err := s.OnChannels(ctx, func(channel string) {
			s.JoinChannel(channel)
		}); nil != err {
			log.Println(err)
		}
	})

	s.irc.OnUserJoinMessage(func(m irc.UserJoinMessage) {
		log.Printf("(%v) %v: joined", m.Channel, m.User)
		go s.checkUser(m.Channel, m.User)
	})

	s.irc.OnUserPartMessage(func(m irc.UserPartMessage) {
		log.Printf("(%v) %v: left", m.Channel, m.User)
	})

	s.irc.OnPrivateMessage(s.handleMessage)

	return s.irc.Connect()
}

func (s *Server) checkUser(channel, user string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	approved, err := s.q.IsApproved(ctx, db.IsApprovedParams{
		ChannelName: channel,
		Username:    user,
	})
	if nil != err && err != sql.ErrNoRows {
		log.Println("unable to check if user is approved", err)
		return
	}
	if approved > 0 {
		log.Printf("(%v) %v: already approved", channel, user)
		return
	}
	bots, err := getBotList()
	if nil != err {
		log.Println("unable to download bot list", err)
		return
	}

	approve := func(context context.Context, channel, user string) {
		log.Printf("(%v) %v: not a bot", channel, user)
		if err := s.q.Approve(ctx, db.ApproveParams{
			ChannelName: channel,
			Username:    user,
		}); nil != err {
			log.Printf("(%v) %v: unable to approve: %v", channel, user, err)
			return
		}
	}

	// Do a naive string search to speed up negative results
	if !strings.Contains(string(bots), user) {
		approve(ctx, channel, user)
		return
	}

	var resp BotResponse
	if err := json.Unmarshal(bots, &resp); err != nil {
		log.Println("unable to unmarshal bot list", err)
		return
	}

	for _, bot := range resp.Bots {
		b, ok := bot[0].(string)
		if !ok {
			log.Println("unable to cast bot name")
		}
		if b != user {
			continue
		}

		if count, ok := bot[1].(float64); !ok {
			log.Println("unable to cast bot count", b)
			s.irc.Say(channel, "/ban "+user)
		} else {
			s.irc.Say(channel, fmt.Sprintf("/ban %v in %v channels", user, int(count)))
		}
		return
	}

	// This user is not a bot
	approve(ctx, channel, user)
}

type BotResponse struct {
	Bots  [][]interface{} `json:"bots"`
	Total int             `json:"_total"`
}

func (s *Server) ReadEnvironmentVariables() error {
	// Read our username from the environment, end if failure
	s.env = &Environment{}
	s.env.twitchSubListen = os.Getenv("TWITCH_SUB_LISTEN")
	s.env.twitchOauthToken = os.Getenv("TWITCH_OAUTH_TOKEN")
	s.env.twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	s.env.twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	s.env.postgresConnectionString = os.Getenv("POSTGRES_CONNECTION_STRING")
	s.env.twitchUsername = os.Getenv("TWITCH_USERNAME")
	s.env.twitchOwner = os.Getenv("TWITCH_OWNER")

	if "" == s.env.twitchUsername ||
		"" == s.env.twitchOwner ||
		"" == s.env.twitchSubListen ||
		"" == s.env.postgresConnectionString ||
		"" == s.env.twitchClientSecret ||
		"" == s.env.twitchClientID ||
		"" == s.env.twitchOauthToken {
		return errors.New("missing environment variable")
	}
	return nil
}

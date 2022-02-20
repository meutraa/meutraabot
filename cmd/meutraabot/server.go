package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/nicklaw5/helix/v2"
	"github.com/pkg/errors"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Server struct {
	irc    *irc.Client
	conn   *sql.DB
	twitch *helix.Client
	q      *db.Queries
	env    *Environment
}

type Environment struct {
	twitchUserID             string
	twitchOwnerID            string
	twitchOauthToken         string
	twitchChallengeSecret    string
	twitchChallengeURL       string
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

	// go s.SubscribeToFollows(channel)

	// Vet all users
	go func() {
		time.Sleep(10 * time.Second)
		users, err := s.irc.Userlist(channel)
		if nil != err {
			log.Println("unable to get user list", err)
			return
		}
		bots, err := getBotList()
		if nil != err {
			log.Println("unable to get bot list on channel join", channel, err)
		} else {
			log.Printf("(%v) found %v users in channel", channel, len(users))
			for _, user := range users {
				go s.checkUser(bots, channel, user)
			}
		}
	}()
}

/*func (s *Server) UnsubscribeAll() {
	subs, err := s.twitch.GetEventSubSubscriptions(&helix.EventSubSubscriptionsParams{})
	if nil != err {
		log.Println("unable to get existing subscriptions")
		return
	}
	log.Println("EventSubSubscriptions", subs.StatusCode)
	log.Println(len(subs.Data.EventSubSubscriptions), "existing subscriptions")

	for _, sub := range subs.Data.EventSubSubscriptions {
		res, err := s.twitch.RemoveEventSubSubscription(sub.ID)
		if err != nil {
			log.Println("unable to unsubscribe to channel follows", err)
		} else {
			log.Printf("code: %v, message: %v, error: %v\n", res.StatusCode, res.ErrorMessage, res.Error)
			log.Printf("unsubscribed from follow notifications %v\n", sub.ID)
		}
	}
}

func (s *Server) SubscribeToFollows(channel string) {
	u, err := UserByName(s.twitch, channel)
	if nil != err {
		log.Printf("(%v) unable to get user: %v", channel, err)
		return
	}
	log.Println("subscribing to", u.ID)

	res, err := s.twitch.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    helix.EventSubTypeChannelFollow,
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: u.ID,
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: s.env.twitchChallengeURL,
			Secret:   s.env.twitchChallengeSecret,
		},
	})
	if err != nil {
		log.Println("unable to subscribe to channel follows", err)
	} else {
		log.Printf("code: %v, message: %v, error: %v\n", res.StatusCode, res.ErrorMessage, res.Error)
		log.Printf("(%v) subscribed to follow notifications\n", channel)
	}
}*/

func (s *Server) PrepareIRC() error {
	self, err := User(s.twitch, s.env.twitchUserID)
	if nil != err {
		log.Println("unable to find user for id", s.env.twitchUserID)
		return err
	}

	s.irc = irc.NewClient(self.Login, s.env.twitchOauthToken)
	log.Println("created client")
	s.irc.OnGlobalUserStateMessage(func(m irc.GlobalUserStateMessage) {
		log.Println("GlobalUserStateMessage recieved")
		// Get own user id
		s.irc.Join(self.Login)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		// s.UnsubscribeAll()
		if err := s.OnChannels(ctx, func(broadcaster helix.User) {
			s.JoinChannel(broadcaster.Login)
		}); nil != err {
			log.Println(err)
		}
	})

	s.irc.OnUserJoinMessage(func(m irc.UserJoinMessage) {
		log.Printf("(%v) %v: joined", m.Channel, m.User)
		go s.checkUser(nil, m.Channel, m.User)
	})

	s.irc.OnPrivateMessage(s.handleMessage)

	log.Println("connecting", s.irc.IrcAddress, s.irc.TLS)
	return s.irc.Connect()
}

func (s *Server) checkUser(bots *BotsResponse, channel, username string) {
	log.Printf("(%v) %v: checking if bot", channel, username)
	if nil == bots {
		var err error
		bots, err = getBotList()
		if nil != err {
			log.Println("unable to download bot list when checking", username, err)
			return
		}

	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Get channel and user id
	cUser, err := UserByName(s.twitch, channel)
	if nil != err {
		log.Println("unable to get channel ID")
		return
	}

	qUser, err := UserByName(s.twitch, username)
	if nil != err {
		log.Println("unable to get user ID")
		return
	}
	channelID := cUser.ID
	userID := qUser.ID

	approved, err := s.q.IsApproved(ctx, db.IsApprovedParams{
		ChannelID: channelID,
		UserID:    userID,
	})
	if nil != err && err != sql.ErrNoRows {
		log.Printf("(%v) %v: unable to check approval status in db: %v", channel, username, err)
		return
	}
	if approved > 0 {
		log.Printf("(%v) %v: already approved", channel, username)
		return
	}
	for _, bot := range bots.Bots {
		b, ok := bot[0].(string)
		if !ok {
			log.Println("unable to cast bot name", bot[0])
			continue
		}
		if b != username {
			continue
		}

		count, ok := bot[1].(int)
		if !ok {
			log.Printf("(%v) %v: banning: is a bot", channel, username)
			s.irc.Say(channel, "/ban "+username+" is a bot")
		} else {
			log.Printf("(%v) %v : banning: is a bot in %v channels", channel, username, count)
			s.irc.Say(channel, "/ban "+username+" is a bot in "+strconv.Itoa(count)+" channels")
		}
		return
	}

	// This user is not a bot
	log.Printf("(%v) %v: not a bot", channel, username)
	if err := s.q.Approve(ctx, db.ApproveParams{
		ChannelID: channelID,
		UserID:    userID,
	}); nil != err {
		log.Printf("(%v) %v: unable to approve: %v", channel, username, err)
		return
	}
}

func (s *Server) ReadEnvironmentVariables() error {
	// Read our username from the environment, end if failure
	s.env = &Environment{}
	s.env.twitchSubListen = os.Getenv("TWITCH_SUB_LISTEN")
	s.env.twitchOauthToken = os.Getenv("TWITCH_OAUTH_TOKEN")
	s.env.twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	s.env.twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	s.env.twitchChallengeSecret = os.Getenv("TWITCH_CHALLENGE_SECRET")
	s.env.twitchChallengeURL = os.Getenv("TWITCH_CHALLENGE_URL")
	s.env.postgresConnectionString = os.Getenv("POSTGRES_CONNECTION_STRING")
	s.env.twitchUserID = os.Getenv("TWITCH_USER_ID")
	s.env.twitchOwnerID = os.Getenv("TWITCH_OWNER_ID")

	if s.env.twitchUserID == "" ||
		s.env.twitchOwnerID == "" ||
		s.env.twitchSubListen == "" ||
		s.env.postgresConnectionString == "" ||
		s.env.twitchChallengeURL == "" ||
		s.env.twitchChallengeSecret == "" ||
		s.env.twitchClientSecret == "" ||
		s.env.twitchClientID == "" ||
		s.env.twitchOauthToken == "" {
		return errors.New("missing environment variable")
	}
	return nil
}

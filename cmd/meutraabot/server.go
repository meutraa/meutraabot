package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	l "log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/samber/lo"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"

	_ "embed"

	irc "github.com/gempir/go-twitch-irc/v3"
	"github.com/meutraa/meutraabot/pkg/db"
	"github.com/nicklaw5/helix/v2"
	"github.com/pkg/errors"
)

type Server struct {
	irc           *irc.Client
	conn          *sql.DB
	twitch        *helix.Client
	q             *db.Queries
	client        *http.Client
	env           *Environment
	oauth         chan string
	selfLogin     string
	selfID        string
	history       map[string][]*irc.PrivateMessage
	conversations map[string][]*irc.PrivateMessage
}

type Environment struct {
	twitchOwnerID      string
	twitchClientSecret string
	twitchRedirectURL  string
	twitchClientID     string
}

func (s *Server) Close() {
	if nil != s.irc {
		s.irc.Disconnect()
	}
	if nil != s.conn {
		s.conn.Close()
	}
}

// //go:embed schema.sql
var ddl string

func (s *Server) PrepareDatabase() error {
	regex := func(re, s string) (bool, error) {
		return regexp.MatchString(re, s)
	}
	sql.Register("sqlite3_extended",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("regexp", regex, true)
			},
		})

	conn, err := sql.Open("sqlite3_extended", "file:db.sql?mode=rwc")
	if nil != err {
		return errors.Wrap(err, "unable to establish connection to database")
	}
	s.conn = conn

	if err := s.conn.Ping(); err != nil {
		return errors.Wrap(err, "unable to ping database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*30))
	defer cancel()

	// create tables
	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		return errors.Wrap(err, "unable to create tables")
	}

	queries, err := db.Prepare(ctx, conn)
	if nil != err {
		return errors.Wrap(err, "unable to prepare queries")
	}
	s.q = queries
	return nil
}

func (s *Server) RefreshUserAccessToken() error {
	dat, err := ioutil.ReadFile("refresh_token")
	if nil != err {
		return errors.Wrap(err, "unable to read refresh_token")
	}

	res, err := s.twitch.RefreshUserAccessToken(string(dat))
	if err != nil {
		return errors.Wrap(err, "unable to refresh user access token")
	}

	l.Println("aquired a refreshed user access token that expires in", res.Data.ExpiresIn, "seconds")

	err = os.WriteFile("user_access_token", []byte(res.Data.AccessToken), 0640)
	if err != nil {
		return errors.Wrap(err, "unable to write user access token to file")
	}

	err = os.WriteFile("refresh_token", []byte(res.Data.RefreshToken), 0640)
	if err != nil {
		return errors.Wrap(err, "unable to write refresh token to file")
	}

	s.twitch.SetUserAccessToken(res.Data.AccessToken)

	return nil
}

func (s *Server) GetUserAccessToken() error {
	// Create a random string to prevent CSRF attacks
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	state := string(b)

	url := s.twitch.GetAuthorizationURL(&helix.AuthorizationURLParams{
		ResponseType: "code",
		State:        state,
		Scopes: []string{
			"chat:edit",
			"chat:read",
			"channel:moderate",
			"moderation:read",
			"moderator:read:chatters",
			"moderator:manage:chat_messages",
			"moderator:manage:banned_users",
			"moderator:manage:announcements",
		},
	})

	fmt.Println("Open", url, "and grant user access token")
	// block waiting for token
	code := <-s.oauth
	stateRes := <-s.oauth

	if state != stateRes {
		return errors.New("state does not match oauth response")
	}

	res, err := s.twitch.RequestUserAccessToken(code)
	if err != nil {
		return errors.Wrap(err, "unable to request user access token")
	}

	l.Println("aquired a user access token that expires in", res.Data.ExpiresIn, "seconds")

	err = os.WriteFile("user_access_token", []byte(res.Data.AccessToken), 0640)
	if err != nil {
		return errors.Wrap(err, "unable to write user access token to file")
	}

	err = os.WriteFile("refresh_token", []byte(res.Data.RefreshToken), 0640)
	if err != nil {
		return errors.Wrap(err, "unable to write refresh token to file")
	}

	s.twitch.SetUserAccessToken(res.Data.AccessToken)

	return nil
}

func (s *Server) PrepareTwitchClient() error {
	l.Println("preparing twitch client")
	client, err := helix.NewClient(&helix.Options{
		ClientID:     s.env.twitchClientID,
		ClientSecret: s.env.twitchClientSecret,
		RedirectURI:  s.env.twitchRedirectURL,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create twitch api client")
	}
	s.client = &http.Client{}
	s.twitch = client

	// Read the user access token from a file, if it exists
	var token string
	dat, err := ioutil.ReadFile("user_access_token")
	if nil != err {
		l.Println("unable to read existing user access token, requesting new one")
		s.GetUserAccessToken()
		token = s.twitch.GetUserAccessToken()
	} else {
		token = string(dat)
		client.SetUserAccessToken(token)
	}

	go s.RefreshUserAccessTokenLoop()

	bot, err := User(s.twitch, "", "")
	if nil != err {
		return errors.Wrap(err, "unable to find user for bot account")
	}

	s.selfLogin = bot.Login
	s.selfID = bot.ID

	return nil
}

func (s *Server) RefreshUserAccessTokenLoop() {
	for {
		l.Println("testing validility of token")
		isValid, res, err := s.twitch.ValidateToken(s.twitch.GetUserAccessToken())
		if err != nil {
			l.Println("unable to validate user access token", err)
		} else if !isValid {
			l.Println("saved token is not valid, requesting new one")
			s.GetUserAccessToken()
		} else {
			if res.Data.ExpiresIn <= 3600 {
				// Refresh token
				err := s.RefreshUserAccessToken()
				if nil != err {
					l.Println("unable to refresh token", err)
				}
			} else {
				l.Println("user access token is valid for another", res.Data.ExpiresIn/60, "minutes")
			}
		}
		l.Println("waiting one hour to validate again")
		time.Sleep(time.Hour)
	}
}

// Channels should always be login usernames, not ids
func (s *Server) JoinChannels(channelnames []string, channelIDs []string) {
	if len(channelnames) != len(channelIDs) {
		l.Println("JoinChannels requires len(usernames) == len(userIDs)")
		return
	}

	s.irc.Join(channelnames...)

	bots, err := getBotList()
	if nil != err {
		l.Println("unable to get known bot list", err)
		return
	}
	l.Printf("found db of %v bots", len(bots))

	// Vet all users
	go func() {
		time.Sleep(5 * time.Second)

		for i, channel := range channelnames {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			log(channel, "", "getting user list to check for bots", nil)
			usernames, err := s.Chatters(ctx, channelIDs[i])
			if nil != err {
				log(channel, "", "unable to get user list", err)
				continue
			}

			log(channel, "", "found chatters: "+fmt.Sprintf("%v", len(usernames)), nil)

			users, err := Users(s.twitch, []string{}, usernames)
			if nil != err {
				log(channel, "", "unable to get users for channel", err)
				continue
			}

			// get the already approved bots
			approvals, err := s.q.GetApprovals(ctx, channelIDs[i])
			if nil != err && err != sql.ErrNoRows {
				l.Println("unable to get approvals", err)
				continue
			}

			// get the users that are bots and not approved
			bots := lo.Filter(users, func(user helix.User, _ int) bool {
				isApproved := lo.ContainsBy(approvals, func(approval db.Approval) bool {
					return approval.UserID == user.ID
				})
				isBot := lo.ContainsBy(bots, func(bot Bot) bool {
					// The api returns the wrong ids
					return bot.Username == user.Login
				})
				return !isApproved && isBot
			})

			log(channel, "", "found unapproved users: "+fmt.Sprintf("%v", len(bots)), nil)

			for _, bot := range bots {
				s.funcBan(ctx, Data{
					Channel:   channel,
					ChannelID: channelIDs[i],
					User:      bot.Login,
					UserID:    bot.ID,
				}, 0, fmt.Sprintf("unapproved bot"))
			}
		}
	}()
}

func (s *Server) PrepareIRC() error {
	/*self, err := User(s.twitch, s.env.twitchUserID, "")
	if nil != err {
		fmt.Println("unable to find user for id", s.env.twitchUserID)
		return err
	}*/

	s.irc = irc.NewClient(s.selfLogin, "oauth:"+s.twitch.GetUserAccessToken())
	s.irc.Capabilities = append(s.irc.Capabilities, irc.MembershipCapability)

	fmt.Println("created client")
	s.irc.OnGlobalUserStateMessage(func(m irc.GlobalUserStateMessage) {
		// Get own user id
		s.irc.Join(s.selfLogin)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()

		// Get the list of channels we should join
		channels, err := s.q.GetChannels(ctx)
		if nil != err && err != sql.ErrNoRows {
			l.Println("unable to get channels", err)
			return
		}
		// convert these ids to logins

		users, err := Users(s.twitch, channels, []string{})
		if nil != err {
			l.Println("unable to get users", err)
			return
		}

		usernames := lo.Map(users, func(user helix.User, _ int) string { return user.Login })
		userIds := lo.Map(users, func(user helix.User, _ int) string { return user.ID })

		s.JoinChannels(usernames, userIds)
	})

	s.irc.OnUserJoinMessage(func(m irc.UserJoinMessage) {
		log(m.Channel, m.User, "joined", nil)
		go s.checkUser(nil, m.Channel, m.User)
	})

	s.irc.OnPrivateMessage(s.handleMessage)

	fmt.Println("connecting")
	return s.irc.Connect()
}

func (s *Server) checkUser(bots []Bot, channel, username string) {
	var err error
	bots, err = getBotList()
	if nil != err {
		log(channel, username, "unable to download known bot list", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Get channel and user id
	cUser, err := User(s.twitch, "", channel)
	if nil != err {
		log(channel, username, "unable to get channel", err)
		return
	}

	qUser, err := User(s.twitch, "", username)
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
		return
	}

	for _, bot := range bots {
		if bot.UserID == userID {
			s.funcBan(ctx, Data{
				Channel:   channel,
				ChannelID: channelID,
				User:      bot.Username,
				UserID:    bot.UserID,
			}, 0, fmt.Sprintf("bot in %v channels", bot.ChannelCount))
			return
		}
	}

	// This user is not a bot
	log(channel, username, "not a bot", nil)
	if err := s.q.Approve(ctx, db.ApproveParams{
		ChannelID: channelID,
		UserID:    userID,
		Manual:    false,
	}); nil != err {
		log(channel, username, "unable to approve", err)
		return
	}
}

func (s *Server) ReadEnvironmentVariables() error {
	// Read our username from the environment, end if failure
	s.env = &Environment{}
	s.env.twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	s.env.twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	s.env.twitchOwnerID = os.Getenv("TWITCH_OWNER_ID")
	s.env.twitchRedirectURL = os.Getenv("TWITCH_REDIRECT_URL")

	if s.env.twitchOwnerID == "" ||
		s.env.twitchClientSecret == "" ||
		s.env.twitchRedirectURL == "" ||
		s.env.twitchClientID == "" {
		return errors.New("missing environment variable")
	}
	return nil
}

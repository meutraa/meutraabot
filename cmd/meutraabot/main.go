package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/nicklaw5/helix"

	_ "github.com/lib/pq"
	ircevent "github.com/thoj/go-ircevent"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

var irc *ircevent.Connection
var username = os.Getenv("TWITCH_USERNAME")

var colors = [...]string{
	"#ff9aa2", "#ffb7b2", "#ffdac1", "#e2f0cb", "#b5ead7", "#c7ceea",
}

type Message struct {
	Channel string
	Body    string
}

const activeInterval = 600

func complain(err error) {
	if nil != err {
		log.Println(err)
	}
}

var q *(db.Queries)
var ctx = context.Background()

func main() {
	// Read our username from the environment, end if failure
	oauth := os.Getenv("TWITCH_OAUTH_TOKEN")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	connString := os.Getenv("POSTGRES_CONNECTION_STRING")

	if "" == username || "" == connString || "" == oauth || "" == clientID || "" == clientSecret {
		log.Println("Missing environment variable")
		return
	}

	// Create a connection to our database, end if failure
	conn, err := sql.Open("postgres", connString)
	if nil != err {
		log.Println("unable to establish connection to database", err)
		return
	}
	if conn.Ping() != nil {
		log.Println("unable to ping database", err)
		return
	}

	defer func() {
		if err := conn.Close(); nil != err {
			log.Println("unable to close database connection:", err)
		}
	}()

	q = db.New(conn)

	client, err := helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		log.Fatalln("unable to create twitch api client", err)
		return
	}

	resp, err := client.GetAppAccessToken()
	if err != nil {
		log.Fatalln("unable to get app access token", err)
		return
	}

	client, err = helix.NewClient(&helix.Options{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		AppAccessToken: resp.Data.AccessToken,
	})
	if err != nil {
		log.Fatalln("unable to create twitch api client with app token", err)
		return
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

	send := make(chan Message)
	go func(c chan Message) {
		for msg := range c {
			fmt.Println("sending", msg.Channel, msg.Body)
			irc.Privmsg(msg.Channel, "/me "+msg.Body)
			time.Sleep(time.Second * 2)
		}
	}(send)

	irc.AddCallback("PRIVMSG", func(e *ircevent.Event) {
		go handleMessage(client, send, e)
	})
	irc.AddCallback("PING", func(e *ircevent.Event) {
		irc.SendRaw("PONG :tmi.twitch.tv")
	})
	if err := irc.Connect("irc.chat.twitch.tv:6697"); nil != err {
		return
	}

	irc.SendRaw("CAP REQ :twitch.tv/tags")

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

func handleCommand(client *helix.Client, channel string, e *ircevent.Event) (string, bool) {
	sender := e.Nick
	text := e.Message()
	isMod := e.Tags["mod"] == "1" || ("#"+e.Nick == channel) || e.Nick == "meutraa"
	isSub := e.Tags["subscriber"] == "1"

	c, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// This is the bot sending a message
	if username == sender {
		if text == "!restart" {
			go func() {
				time.Sleep(5 * time.Second)
				irc.Quit()
				irc.Disconnect()
				os.Exit(0)
			}()
			return "Restarting in 5 seconds", true
		}
	}

	// This is the channel owner
	if "#"+sender == channel {
		if text == "!leave" {
			if err := q.DeleteChannel(c, channel); nil != err {
				return "Failed to leave channel", true
			}

			go func() {
				time.Sleep(1 * time.Second)
				irc.Part(channel)
			}()
			return "Bye bye 👋", true
		}
	}

	if text == "!join" {
		if err := q.CreateChannel(c, "#"+sender); nil != err {
			log.Println("unable to insert channel:", err)
			return "unable to join channel", true
		}

		irc.Join("#" + sender)
		irc.Privmsg("#"+sender, "Hi 🙋")
		return "I will be in #" + sender + " in just a moment", true
	}

	strs := strings.SplitN(text, " ", 2)

	if strs[0] == "!cmd" {
		if len(strs) == 1 {
			return "!cmd set|list|functions|variables", true
		}

		strs = strings.SplitN(strs[1], " ", 2)
		if strs[0] == "list" {
			commands, err := q.GetCommands(c, channel)
			if nil != err && err != sql.ErrNoRows {
				log.Println("unable to get commands", channel, err)
				return "unable to get commands", true
			} else if err == sql.ErrNoRows {
				return "no commands set", true
			}
			return strings.Join(commands, ", "), true
		}

		if strs[0] == "functions" {
			return strings.Join([]string{
				"rank(user string)",
				"points(user string)",
				"activetime(user string)",
				"words(user string)",
				"messages(user string)",
				"counter(name string)",
				"get(url string)",
				"top(count numeric string)",
				"followage(user string)",
				"uptime()",
				"incCounter(name string, count numeric string)",
			}, ", "), true
		}

		if strs[0] == "variables" {
			return strings.Join([]string{
				".User",
				".UserID",
				".Channel",
				".ChannelID",
				".IsMod",
				".IsOwner",
				".IsSub",
				".BotName",
				".Command",
				"index .Arg 0",
				"index .Arg 1 (etc)",
			}, ", "), true
		}

		if isMod {
			if strs[0] == "show" {
				if len(strs) == 1 {
					return "!cmd show command", true
				}
				tmpl, err := q.GetCommand(c, db.GetCommandParams{
					ChannelName: channel,
					Name:        strs[1],
				})
				if nil != err {
					log.Println("unable to get command", channel, strs[1], err)
				}
				return tmpl, true
			}

			if strs[0] == "set" {
				if len(strs) == 1 {
					return "!cmd set command template", true
				}

				strs = strings.SplitN(strs[1], " ", 2)
				tmpl := ""
				if len(strs) > 1 {
					tmpl = strs[1]
				}

				names := strings.Split(strs[0], ",")
				for _, name := range names {
					// Lowercase this for sanity
					name = strings.ToLower(name)

					if tmpl == "" {
						if err := q.DeleteCommand(c, db.DeleteCommandParams{
							ChannelName: channel,
							Name:        name,
						}); nil != err {
							log.Println("unable to delete command", name, err)
							return "unable to delete command", true
						}
						continue
					}
					if err := q.SetCommand(c, db.SetCommandParams{
						ChannelName: channel,
						Name:        name,
						Template:    tmpl,
					}); nil != err {
						log.Println("unable to set command", name, tmpl, err)
						return "unable to set command", true
					}
				}
				return "command set", true
			}
		}
	}

	var tmplStr string
	var command = strings.ToLower(strs[0])
	if command == "!test" && isMod {
		tmplStr = strs[1]
	} else {
		var err error
		tmplStr, err = q.GetCommand(c, db.GetCommandParams{
			ChannelName: channel,
			Name:        command,
		})
		if nil != err && err != sql.ErrNoRows {
			log.Println("unable to get command", err)
			return "", false
		}
	}

	if tmplStr == "" {
		return "", false
	}

	variables := strings.Split(text, " ")[1:]
	data := Data{
		Channel:   strings.TrimPrefix(channel, "#"),
		IsMod:     isMod,
		IsOwner:   "#"+e.Nick == channel,
		IsSub:     isSub,
		User:      sender,
		ChannelID: e.Tags["room-id"],
		UserID:    e.Tags["user-id"],
		BotName:   username,
		Command:   command,
		Arg:       variables,
	}

	tmpl, err := template.New(strs[0]).Funcs(FuncMap(client, e)).Parse(tmplStr)
	if err != nil {
		log.Println("unable to parse template", err)
		return "command template is broken: " + err.Error(), true
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); nil != err {
		log.Println("unable to execute template", err)
		return "command executed wrongly: " + err.Error(), true
	}

	return out.String(), true
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

	rank, err := q.GetWatchTimeRank(c, db.GetWatchTimeRankParams{
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
	defer cancel()
	if err := q.UpdateEmoji(c2, db.UpdateEmojiParams{
		ChannelName: channel,
		Sender:      sender,
		Emoji:       sql.NullString{String: string(rune), Valid: true},
	}); nil != err {
		log.Println("unable to update emoji", err)
		return ""
	}
	return ""
}

func splitRecursive(str string) []string {
	if len(str) <= 480 {
		return []string{str}
	}
	return append([]string{string(str[0:480] + "…")}, splitRecursive(str[480:])...)
}

func handleMessage(client *helix.Client, send chan Message, e *ircevent.Event) {
	channel := e.Arguments[0]
	c, cancel := context.WithTimeout(ctx, time.Second*5)
	if err := q.CreateUser(c, db.CreateUserParams{
		ChannelName: channel,
		Sender:      e.Nick,
		TextColor:   sql.NullString{String: colors[rand.Intn(len(colors)-1)], Valid: true},
	}); nil != err {
		log.Println("unable to create user", err)
	}
	cancel()

	c, cancel = context.WithTimeout(ctx, time.Second*5)
	if err := q.UpdateMetrics(c, db.UpdateMetricsParams{
		ChannelName: channel,
		Sender:      e.Nick,
		WordCount:   int64(len(strings.Split(e.Message(), " "))),
	}); nil != err {
		log.Println("unable to update metrics for user", err)
	}
	cancel()

	c, cancel = context.WithTimeout(ctx, time.Second*2)
	if err := q.CreateMessage(c, db.CreateMessageParams{
		ChannelName: channel,
		Sender:      e.Nick,
		Message:     e.Message(),
	}); nil != err {
		log.Println("unable to add message", err)
	}
	cancel()

	res, match := handleCommand(client, channel, e)
	log.Printf("%v:%v:%v < %v\n", channel, e.Nick, e.Message(), res)
	res = strings.TrimSpace(res)
	log.Println(e.Tags)
	if "" != res {
		isMod := e.Tags["mod"] == "1" || ("#"+e.Nick == channel) || e.Nick == "meutraa"
		if (strings.HasPrefix(e.Message(), "!") || strings.HasPrefix(e.Message(), "h")) && !isMod && match {
			irc.Privmsg(channel, "/delete "+e.Tags["id"])
		}
		for _, msg := range splitRecursive(res) {
			send <- Message{
				Channel: channel,
				Body:    msg,
			}
		}
	}
}

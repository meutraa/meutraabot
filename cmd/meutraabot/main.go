package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/lib/pq"
	"github.com/meutraa/helix"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Message struct {
	Channel string
	Body    string
}

const seperator = " "

func main() {
	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run() error {
	s := Server{}
	if err := s.ReadEnvironmentVariables(); nil != err {
		return err
	}

	if err := s.PrepareDatabase(); nil != err {
		return err
	}
	defer func() {
		s.Close()
	}()

	if err := s.PrepareTwitchClient(); nil != err {
		return err
	}

	go func() {
		r := chi.NewRouter()
		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ch := r.URL.Query().Get("hub.challenge")
			log.Println("hub.challenge =", ch)
			w.Header().Add("content-type", "text/plain")
			w.Write([]byte(ch))
			w.WriteHeader(http.StatusOK)
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var res helix.ManyFollows
			err := json.NewDecoder(r.Body).Decode(&res)
			if nil != err {
				log.Println("unable to parse twitch sub body", err)
				return
			}
			for _, follower := range res.Follows {
				user, err := User(s.twitch, follower.FromName)
				if nil != err {
					log.Println("unable to look up new follower by name", follower.FromName, err)
					continue
				}
				seconds := time.Now().Unix() - user.CreatedAt.Unix()
				if seconds < 86400 {
					log.Printf("(%v) %v banned based on age %v\n", follower.ToName, follower.FromName, seconds/60)
					msg := fmt.Sprintf("/ban %v %v minutes old - send an unban request", follower.FromName, seconds/60)
					log.Printf("(%v) %v", follower.ToName, msg)
					s.irc.Say(follower.ToName, msg)
				} else {
					log.Printf("(%v) %v approved based on age %v\n", follower.ToName, follower.FromName, seconds/60)
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		if err := http.ListenAndServe(s.env.twitchSubListen, r); nil != err {
			log.Println("unable to listen and server", err)
		}
	}()

	if err := s.PrepareIRC(); nil != err {
		return err
	}

	// Create a channel for the OS to notify us of interrupts/signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt
	return nil
}

func (s *Server) handleCommand(ctx context.Context, e *irc.PrivateMessage) string {
	text := e.Message

	isOwner := e.User.Name == s.env.twitchOwner
	isAdmin := (e.User.Name == e.Channel) || isOwner
	isMod := e.Tags["mod"] == "1" || isAdmin
	isSub := e.Tags["subscriber"] == "1"

	split := strings.Split(text, " ")
	command := split[0]
	args := split[1:]
	argCount := len(args)
	var tmplStr string

	data := Data{
		Channel:      e.Channel,
		IsMod:        isMod,
		IsOwner:      e.User.Name == e.Channel,
		IsSub:        isSub,
		MessageID:    e.ID,
		User:         e.User.Name,
		ChannelID:    e.RoomID,
		UserID:       e.User.ID,
		BotName:      s.env.twitchUsername,
		Command:      command,
		Arg:          args,
		SelectedUser: pick(e.User.Name, args),
	}
	functions := s.FuncMap(ctx, data, e)

	// Built-in commands
	switch {
	case command == "!approve" && isAdmin && argCount == 1:
		if err := s.q.Approve(ctx, db.ApproveParams{
			ChannelName: e.Channel, Username: args[0],
		}); nil != err {
			log.Println("unable to approve user", err)
			return "failed to approve user"
		}

		return "/unban " + args[0] + "\n" + args[0] + " approved"
	case command == "!unapprove" && isAdmin && argCount == 1:
		if err := s.q.Unapprove(ctx, db.UnapproveParams{
			ChannelName: e.Channel, Username: args[0],
		}); nil != err {
			log.Println("unable to approve user", err)
			return "failed to approve user"
		}

		return "/ban " + args[0] + "\n" + args[0] + " unapproved"
	case command == "!leave":
		if err := s.q.DeleteChannel(ctx, e.Channel); nil != err {
			return "failed to leave channel"
		}

		go func() {
			time.Sleep(1 * time.Second)
			s.irc.Depart(e.User.Name)
		}()
		return "Bye bye " + e.User.Name + "👋"
	case command == "!join":
		if err := s.q.CreateChannel(ctx, e.User.Name); nil != err {
			log.Println("unable to insert channel:", err)
			return "unable to join channel"
		}

		s.JoinChannel(e.User.Name)
		return "Hi " + e.User.Name + " 👋"
	case command == "!data":
		bytes, _ := json.Marshal(data)
		return string(bytes)
	case command == "!glist":
		commands, err := s.q.GetCommands(ctx, "global")
		if nil != err && err != sql.ErrNoRows {
			return "unable to get commands"
		}
		return strings.Join(commands, seperator)
	case command == "!list":
		commands, err := s.q.GetCommands(ctx, e.Channel)
		if nil != err && err != sql.ErrNoRows {
			return "unable to get commands"
		}
		return strings.Join(commands, seperator)
	case command == "!gget" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelName: "global",
			Name:        args[0],
		})
		if nil != err {
			log.Println("unable to get global command", e.Channel, args[0], err)
			return ""
		}
		return tmpl
	case command == "!get" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelName: e.Channel,
			Name:        args[0],
		})
		if nil != err {
			log.Println("unable to get command", e.Channel, args[0], err)
			return ""
		}
		return tmpl
	case command == "!builtins":
		return strings.Join([]string{
			"!leave",
			"!approve",
			"!unapprove",
			"!join",
			"!get",
			"!set",
			"!unset",
			"!list",
			"!gget",
			"!gset",
			"!gunset",
			"!glist",
			"!functions",
			"!data",
			"!test",
			"!builtins",
		}, seperator)
	case command == "!functions":
		return strings.Join([]string{
			"rank(user string?)",
			"points(user string?)",
			"activetime(user string?)",
			"words(user string?)",
			"messages(user string?)",
			"counter(name string)",
			"get(url string)",
			"json(key string, json string)",
			"top(count numeric string)",
			"followage(user string?)",
			"uptime()",
			"incCounter(name string, count numeric string)",
		}, seperator)
	case command == "!gunset" && isOwner && argCount == 1:
		if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
			ChannelName: "global",
			Name:        args[0],
		}); nil != err {
			log.Println("unable to delete global command", args[0], err)
			return "unable to delete global command"
		}
	case command == "!unset" && isMod && argCount == 1:
		if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
			ChannelName: e.Channel,
			Name:        args[0],
		}); nil != err {
			log.Println("unable to delete command", args[0], err)
			return "unable to delete command"
		}
	case command == "!gset" && isOwner && argCount > 1:
		strs := strings.SplitN(text, " ", 3)[1:]
		if err := s.q.SetCommand(ctx, db.SetCommandParams{
			ChannelName: "global",
			Name:        strs[0],
			Template:    strs[1],
		}); nil != err {
			log.Println("unable to set global command", strs[0], strs[1], err)
			return "unable to set global command"
		}
		return "command set"
	case command == "!set" && isMod && argCount > 1:
		strs := strings.SplitN(text, " ", 3)[1:]
		if err := s.q.SetCommand(ctx, db.SetCommandParams{
			ChannelName: e.Channel,
			Name:        strs[0],
			Template:    strs[1],
		}); nil != err {
			log.Println("unable to set command", strs[0], strs[1], err)
			return "unable to set command"
		}
		return "command set"
	case command == "!test" && isMod:
		tmplStr = strings.Join(args, " ")
	default:
		commands, err := s.q.GetMatchingCommands(ctx, db.GetMatchingCommandsParams{
			ChannelName: e.Channel,
			Message:     strings.ToLower(text),
		})
		if nil != err && err != sql.ErrNoRows {
			log.Println("unable to get command", err)
			return ""
		}

		local := make([]db.GetMatchingCommandsRow, 0, 8)
		global := make([]db.GetMatchingCommandsRow, 0, 8)
		for _, c := range commands {
			if c.ChannelName != "global" {
				local = append(local, c)
			} else {
				global = append(global, c)
			}
		}

		if len(local) > 1 {
			cs := make([]string, len(local))
			for i, c := range local {
				cs[i] = c.Name
			}
			return "message matches multiple local commands: " + strings.Join(cs, seperator)
		} else if len(local) == 1 {
			tmplStr = local[0].Template
		} else if len(global) > 1 {
			cs := make([]string, len(global))
			for i, c := range global {
				cs[i] = c.Name
			}
			return "message matches multiple global commands: " + strings.Join(cs, seperator)
		} else if len(global) == 1 {
			tmplStr = global[0].Template
		}
	}

	if tmplStr == "" {
		return ""
	}

	// Execute command
	tmpl, err := template.New(text).Funcs(functions).Parse(tmplStr)
	if err != nil {
		return "command template is broken: " + err.Error()
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); nil != err {
		return "command executed wrongly: " + err.Error()
	}

	return out.String()
}

func splitRecursive(str string) []string {
	if len(str) == 0 {
		return []string{}
	}
	if len(str) <= 480 {
		return []string{str}
	}
	return append([]string{string(str[0:480] + "…")}, splitRecursive(str[480:])...)
}

func (s *Server) handleMessage(e irc.PrivateMessage) {
	if s.env.twitchUsername == e.User.Name {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.q.CreateUser(ctx, db.CreateUserParams{
		ChannelName: e.Channel,
		Sender:      e.User.Name,
	}); nil != err {
		log.Println("unable to create user", err)
	}

	if err := s.q.UpdateMetrics(ctx, db.UpdateMetricsParams{
		ChannelName: e.Channel,
		Sender:      e.User.Name,
		WordCount:   int64(len(strings.Split(e.Message, " "))),
	}); nil != err {
		log.Println("unable to update metrics for user", err)
	}

	if err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ChannelName: e.Channel,
		Sender:      e.User.Name,
		Message:     e.Message,
	}); nil != err {
		log.Println("unable to add message", err)
	}

	res := s.handleCommand(ctx, &e)
	log.Printf("(%v) %v: %v\n", e.Channel, e.User.Name, e.Message)
	if "" != res {
		log.Printf("(%v) %v: %v\n", e.Channel, s.env.twitchUsername, res)
	}
	res = strings.ReplaceAll(res, "\\n", "\n")
	for _, message := range strings.Split(res, "\n") {
		for _, parts := range splitRecursive(strings.TrimSpace(message)) {
			if parts == "" {
				continue
			}
			s.irc.Say(e.Channel, parts)
		}
	}
}

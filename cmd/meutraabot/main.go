package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/nicklaw5/helix/v2"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Message struct {
	Channel string
	Body    string
}

type eventSubNotification struct {
	Subscription helix.EventSubSubscription `json:"subscription"`
	Challenge    string                     `json:"challenge"`
	Event        json.RawMessage            `json:"event"`
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
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Println(err)
				return
			}
			defer r.Body.Close()
			// verify that the notification came from twitch using the secret.
			if !helix.VerifyEventSubNotification("q1k1lnfKbIU3RQ20ligrtVv3vHAkPW51", r.Header, string(body)) {
				log.Println("no valid signature on subscription")
				return
			} else {
				log.Println("verified signature for subscription")
			}
			var vals eventSubNotification
			err = json.NewDecoder(bytes.NewReader(body)).Decode(&vals)
			if err != nil {
				log.Println(err)
				return
			}
			// if there's a challenge in the request, respond with only the challenge to verify your eventsub.
			if vals.Challenge != "" {
				w.Write([]byte(vals.Challenge))
				return
			}

			var res helix.EventSubChannelFollowEvent
			err = json.NewDecoder(bytes.NewReader(vals.Event)).Decode(&res)
			if nil != err {
				log.Println("unable to parse twitch sub body", err)
				return
			}

			user, err := User(s.twitch, res.UserID)
			if nil != err {
				log.Println("unable to look up new follower by name", res.UserName, err)
				return
			}
			seconds := time.Now().Unix() - user.CreatedAt.Unix()
			msg := ""

			if seconds < 86400 {
				count, err := s.q.GetMessageCount(context.Background(), db.GetMessageCountParams{
					ChannelID: res.BroadcasterUserID,
					SenderID:  res.UserID,
				})
				// The error is not important here, but log it
				if nil != err && err != sql.ErrNoRows {
					log.Println("unable to get message count for user", res.BroadcasterUserName, res.UserName, err)
				}
				if count == 0 {
					msg = fmt.Sprintf("/ban %v %v minutes old - send an unban request", res.UserName, seconds/60)
				}
			}
			if msg != "" {
				log.Printf("(%v) %v", res.BroadcasterUserName, msg)
				s.irc.Say(res.BroadcasterUserName, msg)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		if err := http.ListenAndServe(s.env.twitchSubListen, r); nil != err {
			log.Println("unable to listen and server", err)
		}
	}()

	log.Println("creating irc")
	if err := s.PrepareIRC(); nil != err {
		return err
	}
	log.Println("created irc")

	// Create a channel for the OS to notify us of interrupts/signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt
	return nil
}

func (s *Server) handleCommand(ctx context.Context, e *irc.PrivateMessage) string {
	text := e.Message

	isOwner := e.User.ID == s.env.twitchOwnerID || e.User.ID == ""
	isAdmin := (e.User.Name == e.Channel) || isOwner
	isMod := e.Tags["mod"] == "1" || isAdmin
	isSub := e.Tags["subscriber"] == "1"

	split := strings.Split(text, " ")
	command := split[0]
	args := split[1:]
	argCount := len(args)
	var tmplStr string

	selectedUser := firstOr(args, e.User.Name)
	selectedUserID := e.User.ID
	if selectedUser != e.User.Name {
		sUser, err := UserByName(s.twitch, selectedUser)
		if nil == err {
			selectedUserID = sUser.ID
		}
	}

	data := Data{
		Channel:        e.Channel,
		ChannelID:      e.RoomID,
		User:           e.User.Name,
		UserID:         e.User.ID,
		IsMod:          isMod,
		IsAdmin:        isAdmin,
		IsOwner:        isOwner,
		IsSub:          isSub,
		MessageID:      e.ID,
		BotID:          s.env.twitchUserID,
		Command:        command,
		Arg:            args,
		SelectedUser:   selectedUser,
		SelectedUserID: selectedUserID,
	}
	functions := s.FuncMap(ctx, data, e)

	// Built-in commands
	switch {
	case command == "+approve" && isAdmin && argCount == 1:
		if err := s.q.Approve(ctx, db.ApproveParams{
			ChannelID: e.RoomID, UserID: selectedUserID,
		}); nil != err {
			log.Println("unable to approve user", err)
			return "failed to approve user"
		}

		return "/unban " + selectedUser + "\n" + selectedUser + " approved"
	case command == "+unapprove" && isAdmin && argCount == 1:
		if err := s.q.Unapprove(ctx, db.UnapproveParams{
			ChannelID: e.RoomID, UserID: selectedUserID,
		}); nil != err {
			log.Println("unable to approve user", err)
			return "failed to approve user"
		}

		return "/ban " + selectedUser + "\n" + selectedUser + " unapproved"
	case command == "+leave":
		// TODO: this seems iffy
		if err := s.q.DeleteChannel(ctx, e.RoomID); nil != err {
			return "failed to leave channel"
		}

		go func() {
			time.Sleep(1 * time.Second)
			s.irc.Depart(e.User.Name)
		}()
		return "Bye " + e.User.Name + "👋"
	case command == "+join":
		if e.RoomID != s.env.twitchUserID {
			return ""
		}
		if argCount == 1 && !isOwner {
			return ""
		} else if argCount == 1 {
			if err := s.q.CreateChannel(ctx, selectedUserID); nil != err {
				log.Println("unable to insert channel:", err)
				return "unable to join channel"
			}
			s.JoinChannel(args[0])
			return "Hi " + args[0] + " 👋"
		}

		if err := s.q.CreateChannel(ctx, e.User.ID); nil != err {
			log.Println("unable to insert channel:", err)
			return "unable to join channel"
		}

		s.JoinChannel(e.User.Name)
		return "Hi " + e.User.Name + " 👋"
	case command == "+data":
		bytes, _ := json.Marshal(data)
		return string(bytes)
	case command == "+glist":
		commands, err := s.q.GetCommands(ctx, "0")
		if nil != err && err != sql.ErrNoRows {
			return "unable to get commands"
		}
		return strings.Join(commands, seperator)
	case command == "+list":
		commands, err := s.q.GetCommands(ctx, e.RoomID)
		if nil != err && err != sql.ErrNoRows {
			return "unable to get commands"
		}
		return strings.Join(commands, seperator)
	case command == "+gget" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelID: "0",
			Name:      args[0],
		})
		if nil != err {
			log.Println("unable to get global command", e.RoomID, args[0], err)
			return ""
		}
		return tmpl
	case command == "+get" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelID: e.RoomID,
			Name:      args[0],
		})
		if nil != err {
			log.Println("unable to get command", e.RoomID, args[0], err)
			return ""
		}
		return tmpl
	case command == "+builtins":
		return strings.Join([]string{
			"+leave",
			"+approve",
			"+unapprove",
			"+join",
			"+get",
			"+set",
			"+unset",
			"+list",
			"+gget",
			"+gset",
			"+gunset",
			"+glist",
			"+functions",
			"+data",
			"+test",
			"+builtins",
		}, seperator)
	case command == "+functions":
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
	case command == "+gunset" && isOwner && argCount == 1:
		if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
			ChannelID: "0",
			Name:      args[0],
		}); nil != err {
			log.Println("unable to delete global command", args[0], err)
			return "unable to delete global command"
		}
	case command == "+unset" && isMod && argCount == 1:
		if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
			ChannelID: e.RoomID,
			Name:      args[0],
		}); nil != err {
			log.Println("unable to delete command", args[0], err)
			return "unable to delete command"
		}
	case command == "+gset" && isOwner && argCount > 1:
		strs := strings.SplitN(text, " ", 3)[1:]
		if err := s.q.SetCommand(ctx, db.SetCommandParams{
			ChannelID: "0",
			Name:      strs[0],
			Template:  strs[1],
		}); nil != err {
			log.Println("unable to set global command", strs[0], strs[1], err)
			return "unable to set global command"
		}
		return "command set"
	case command == "+set" && isMod && argCount > 1:
		strs := strings.SplitN(text, " ", 3)[1:]
		if err := s.q.SetCommand(ctx, db.SetCommandParams{
			ChannelID: e.RoomID,
			Name:      strs[0],
			Template:  strs[1],
		}); nil != err {
			log.Println("unable to set command", strs[0], strs[1], err)
			return "unable to set command"
		}
		return "command set"
	case command == "+test" && isMod:
		tmplStr = strings.Join(args, " ")
	default:
		commands, err := s.q.GetMatchingCommands(ctx, db.GetMatchingCommandsParams{
			ChannelID:       e.RoomID,
			ChannelGlobalID: "0",
			Message:         strings.ToLower(text),
		})
		if nil != err && err != sql.ErrNoRows {
			log.Println("unable to get command", err)
			return ""
		}

		local := make([]db.GetMatchingCommandsRow, 0, 8)
		global := make([]db.GetMatchingCommandsRow, 0, 8)
		for _, c := range commands {
			if c.ChannelID != "0" {
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
	if s.env.twitchUserID == e.User.ID {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := s.q.CreateUser(ctx, db.CreateUserParams{
		ChannelID: e.RoomID,
		SenderID:  e.User.ID,
	}); nil != err {
		log.Println("unable to create user", err)
	}

	if err := s.q.UpdateMetrics(ctx, db.UpdateMetricsParams{
		ChannelID: e.RoomID,
		SenderID:  e.User.ID,
		WordCount: int64(len(strings.Split(e.Message, " "))),
	}); nil != err {
		log.Println("unable to update metrics for user", err)
	}

	if err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ChannelID: e.RoomID,
		SenderID:  e.User.ID,
		Message:   e.Message,
	}); nil != err {
		log.Println("unable to add message", err)
	}

	res := s.handleCommand(ctx, &e)
	log.Printf("(%v) %v: %v\n", e.RoomID, e.User.Name, e.Message)
	if res != "" {
		log.Printf("(%v) %v: %v\n", e.RoomID, s.env.twitchUserID, res)
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

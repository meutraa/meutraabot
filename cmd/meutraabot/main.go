package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v3"
	_ "github.com/lib/pq"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Message struct {
	Channel string
	Body    string
}

const seperator = " "

func log(channel, username, message string, err error) {
	fmt.Printf("[%v]", channel)
	if username != "" {
		fmt.Printf(" %v", username)
	}
	if message != "" {
		fmt.Printf(": %v", message)
	}
	if err != nil {
		fmt.Printf(": %v", err.Error())
	}
	fmt.Printf("\n")
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
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
			log(data.Channel, data.User, "unable to approve", err)
			return "failed to approve user"
		}

		return "/unban " + selectedUser + "\n" + selectedUser + " approved"
	case command == "+unapprove" && isAdmin && argCount == 1:
		if err := s.q.Unapprove(ctx, db.UnapproveParams{
			ChannelID: e.RoomID, UserID: selectedUserID,
		}); nil != err {
			log(data.Channel, data.User, "unable to unapprove", err)
			return "failed to approve user"
		}

		return "/ban " + selectedUser + "\n" + selectedUser + " unapproved"
	case command == "+leave":
		// TODO: this seems iffy
		if err := s.q.DeleteChannel(ctx, e.User.ID); nil != err {
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
				log(data.Channel, data.User, "unable to add channel", err)
				return "unable to join channel"
			}
			s.JoinChannel(args[0])
			msg := "Hi " + args[0] + " 👋"
			func() {
				time.Sleep(time.Second * 2)
				s.irc.Say(selectedUser, msg)
			}()
			return msg
		}

		if err := s.q.CreateChannel(ctx, e.User.ID); nil != err {
			log(data.Channel, data.User, "unable to add channel", err)
			return "unable to join channel"
		}

		s.JoinChannel(e.User.Name)
		msg := "Hi " + e.User.Name + " 👋"
		func() {
			time.Sleep(time.Second * 2)
			s.irc.Say(e.User.Name, msg)
		}()
		return msg
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
			log(data.Channel, data.User, "unable to gget "+args[0], err)
			return ""
		}
		return tmpl
	case command == "+get" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelID: e.RoomID,
			Name:      args[0],
		})
		if nil != err {
			log(data.Channel, data.User, "unable to get "+args[0], err)
			return ""
		}
		return tmpl
	case command == "+builtins":
		return strings.Join([]string{
			"+join",
			"+leave",
			"+approve",
			"+unapprove",
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
			"age()",
			"counter(name)",
			"incCounter(name, number)",
			"get(url)",
			"json(key, json)",
			"followage()",
			"uptime()",
		}, seperator)
	case command == "+gunset" && isOwner && argCount == 1:
		if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
			ChannelID: "0",
			Name:      args[0],
		}); nil != err {
			log(data.Channel, data.User, "unable to gunset "+args[0], err)
			return "unable to delete global command"
		}
	case command == "+unset" && isMod && argCount == 1:
		if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
			ChannelID: e.RoomID,
			Name:      args[0],
		}); nil != err {
			log(data.Channel, data.User, "unable to unset "+args[0], err)
			return "unable to delete command"
		}
	case command == "+gset" && isOwner && argCount > 1:
		strs := strings.SplitN(text, " ", 3)[1:]
		if err := s.q.SetCommand(ctx, db.SetCommandParams{
			ChannelID: "0",
			Name:      strs[0],
			Template:  strs[1],
		}); nil != err {
			log(data.Channel, data.User, "unable to gset "+strs[0]+" "+strs[1], err)
			return "unable to set global command"
		}
		return "command set"
	case command == "+set" && isMod && argCount > 0:
		strs := strings.SplitN(text, " ", 3)[1:]
		template := ""
		if argCount > 1 {
			template = strs[1]
		}
		if err := s.q.SetCommand(ctx, db.SetCommandParams{
			ChannelID: e.RoomID,
			Name:      strs[0],
			Template:  template,
		}); nil != err {
			log(data.Channel, data.User, "unable to set "+strs[0]+" "+strs[1], err)
			return "unable to set command"
		}
		return fmt.Sprintf("command %v set", strs[0])
	case command == "+test" && isMod:
		tmplStr = strings.Join(args, " ")
	default:
		message := strings.ToLower(text)
		commands, err := s.q.GetMatchingCommands(ctx, db.GetMatchingCommandsParams{
			ChannelID:       e.RoomID,
			ChannelGlobalID: "0",
			Message:         message,
		})
		if nil != err && err != sql.ErrNoRows {
			log(data.Channel, data.User, "unable to get for "+message, err)
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

	res := s.handleCommand(ctx, &e)
	log(e.Channel, e.User.Name, e.Message, nil)
	if res != "" {
		log(e.Channel, "self", res, nil)
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

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v3"
	"github.com/meutraa/meutraabot/pkg/db"
)

type Message struct {
	Channel string
	Body    string
}

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
	rand.Seed(time.Now().UnixNano())
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	s := Server{}

	s.history = make(map[string][]*irc.PrivateMessage)
	s.conversations = make(map[string][]*irc.PrivateMessage)
	s.oauth = make(chan string)

	if err := s.ReadEnvironmentVariables(); nil != err {
		return err
	}

	if err := s.PrepareDatabase(); nil != err {
		return err
	}
	defer func() {
		s.Close()
	}()

	s.PrepareAPI()

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

	selectedUser := e.User.Name
	isUser := false
	// If the command only has one argument, assume it is a user (not always true)
	if len(args) == 1 && args[0] != "" {
		selectedUser = args[0]
		isUser = true
	}
	selectedUserID := e.User.ID
	if isUser {
		sUser, err := User(s.twitch, "", selectedUser)
		if nil == err {
			selectedUserID = sUser.ID
		}
	}

	data := Data{
		Channel:        e.Channel,
		ChannelID:      e.RoomID,
		User:           e.User.Name,
		UserID:         e.User.ID,
		Event:          e,
		IsMod:          isMod,
		IsAdmin:        isAdmin,
		IsOwner:        isOwner,
		IsSub:          isSub,
		Message:        e.Message,
		MessageID:      e.ID,
		BotID:          s.env.twitchUserID,
		Command:        command,
		Arg:            args,
		SelectedUser:   selectedUser,
		SelectedUserID: selectedUserID,
	}
	if e.Reply != nil {
		data.ReplyingToUser = e.Reply.ParentUserLogin
		data.ReplyingToUserID = e.Reply.ParentUserID
		data.ReplyingToMessage = e.Reply.ParentMsgBody
		data.ReplyingToMessageID = e.Reply.ParentMsgID
	}

	functions := s.FuncMap(ctx, data, e)
	templates := make(map[string]string)

	// Built-in commands
	switch {
	case command == "+approve" && isAdmin && argCount == 1:
		if err := s.q.Approve(ctx, db.ApproveParams{
			ChannelID: e.RoomID,
			UserID:    selectedUserID,
			Manual:    true,
		}); nil != err {
			log(data.Channel, data.User, "unable to approve", err)
			return "failed to approve user"
		}

		// A little hack
		data.User = data.SelectedUser
		data.UserID = data.SelectedUserID

		s.funcUnban(ctx, data)
		return ""
	case command == "+unapprove" && isAdmin && argCount == 1:
		if err := s.q.Unapprove(ctx, db.UnapproveParams{
			ChannelID: e.RoomID, UserID: selectedUserID,
		}); nil != err {
			log(data.Channel, data.User, "unable to unapprove", err)
			return "failed to approve user"
		}

		// A little hack
		data.User = data.SelectedUser
		data.UserID = data.SelectedUserID

		s.funcBan(ctx, data, 0, "bot unapproved")
		return ""
	case command == "+leave":
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
			s.JoinChannels([]string{selectedUser}, []string{selectedUserID})
			msg := "Hi " + selectedUser + " 👋"
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

		s.JoinChannels([]string{e.User.Name}, []string{e.User.ID})
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
		return strings.Join(commands, " ")
	case command == "+list":
		commands, err := s.q.GetCommands(ctx, e.RoomID)
		if nil != err && err != sql.ErrNoRows {
			return "unable to get commands"
		}
		return strings.Join(commands, " ")
	case command == "+gget" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelID: "0",
			Name:      args[0],
		})
		if nil != err {
			log(data.Channel, data.User, "unable to gget "+args[0], err)
			return ""
		}
		return "command: " + tmpl
	case command == "+get" && argCount == 1:
		tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
			ChannelID: e.RoomID,
			Name:      args[0],
		})
		if nil != err {
			log(data.Channel, data.User, "unable to get "+args[0], err)
			return ""
		}
		return "command: " + tmpl
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
		}, " ")
	case command == "+functions":
		return strings.Join([]string{
			"reply(message)",
			"user()",
			"userfollow()",
			"stream()",
			"delete()",
			"clear()",
			"timeout(seconds, reason)",
			"ban(reason)",
			"random(max)",
			"duration(time)",
			"get(url)",
			"json(key, json)",
		}, " ")
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
		templates["test"] = strings.Join(args, " ")
	default:
		message := strings.ToLower(text)
		commands, err := s.q.GetMatchingCommands(ctx,
			e.RoomID,
			message,
		)
		if nil != err && err != sql.ErrNoRows {
			log(data.Channel, data.User, "unable to get for "+message, err)
			return ""
		}

		local := []db.GetMatchingCommandsRow{}
		global := []db.GetMatchingCommandsRow{}
		for _, c := range commands {
			if c.ChannelID != "0" {
				local = append(local, c)
			} else {
				global = append(global, c)
			}
		}

		for _, template := range global {
			templates[template.Name] = template.Template
		}
		for _, template := range local {
			templates[template.Name] = template.Template
		}
	}

	// log(data.Channel, data.User, "Matched templates len "+fmt.Sprint(len(templates)), nil)
	if len(templates) == 0 {
		// get the channel settings
		settings, err := s.q.GetChannel(ctx, data.ChannelID)
		if err != nil {
			log(data.Channel, data.User, "unable to get channel settings", err)
			return ""
		}

		// Not responding, but might send random message
		history, ok := s.history[e.Channel]
		if !ok {
			return ""
		}

		if data.ReplyingToMessageID != "" {
			for _, m := range history {
				if m.Reply != nil && m.Reply.ParentMsgID == data.ReplyingToMessageID {
					if m.User.ID == s.env.twitchUserID {
						return "reply::delay::" + s.funcReplyAuto(ctx, data, data.Message, false, func() string { return "" })
					}
				}
			}
		}

		if !settings.AutoreplyEnabled {
			return ""
		}

		// find how many messages it has been since meuua last said something
		count := 0
		totalMessages := 0
		botMessages := 0
		activeUsers := map[string]bool{}
		messagesChecked := 0
		for i := len(history) - 1; i >= 0; i-- {
			isBot := history[i].User.ID == s.env.twitchUserID
			if isBot && count == 0 {
				count = len(history) - i
			}

			// if message is less than 10 minutes old, count it
			if messagesChecked < 50 {
				activeUsers[history[i].User.DisplayName] = true
				totalMessages++
				if isBot {
					botMessages++
				}
			} else {
				break
			}
		}

		if count == 0 {
			count = totalMessages
		}

		if count > 1 && float64(botMessages) < float64(totalMessages)/(float64(len(activeUsers)+2)*settings.AutoreplyFrequency) {
			res := s.funcReplyAuto(ctx, data, e.Message, true, func() string {
				// get last ten messages from history
				if len(history) > 6 {
					history = history[len(history)-6:]
				}
				p := ""
				for _, m := range history {
					p += m.User.DisplayName + ": " + m.Message + "\n"
				}
				return p
			})
			return res
		}

		return ""
	}

	str := strings.Builder{}
	i := 0
	// Execute command
	for _, tplt := range templates {
		if i > 0 {
			str.WriteByte('\n')
		}
		tmpl, err := template.New(text).Funcs(functions).Parse(tplt)
		if err != nil {
			return "command template is broken: " + err.Error()
		}

		out := bytes.Buffer{}
		if err := tmpl.Execute(&out, data); nil != err {
			return "command executed wrongly: " + err.Error()
		}
		str.WriteString(out.String())
		i++
	}

	return str.String()
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
	// Add event to history
	if _, ok := s.history[e.Channel]; !ok {
		s.history[e.Channel] = make([]*irc.PrivateMessage, 0)
	}
	s.history[e.Channel] = append(s.history[e.Channel], &e)

	if s.env.twitchUserID == e.User.ID {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	res := s.handleCommand(ctx, &e)
	log(e.Channel, e.User.Name, e.Message, nil)
	if res != "" {
		log(e.Channel, "self", res, nil)
	}
	res = strings.ReplaceAll(res, "\\n", "\n")
	go func(res string) {
		for _, message := range strings.Split(res, "\n") {
			for _, parts := range splitRecursive(strings.TrimSpace(message)) {
				if parts == "" {
					continue
				}
				reply := false
				if strings.HasPrefix(parts, "reply::") {
					parts = strings.TrimPrefix(parts, "reply::")
					reply = true
				}
				if strings.HasPrefix(parts, "delay::") {
					parts = strings.TrimPrefix(parts, "delay::")
					// calculate time to type parts in seconds at 80wpm
					chars := len(parts)
					seconds := int(math.Round(float64(chars) / 5))

					time.Sleep(time.Second * time.Duration(seconds))
				}
				if reply {
					if strings.HasPrefix(parts, "@") && strings.Contains(parts, " ") {
						// remove the first word from parts
						parts = strings.SplitN(parts, " ", 2)[1]
					}
				}
				add := irc.PrivateMessage{
					Channel: e.Channel,
					Reply:   &irc.Reply{},
					Message: parts,
					User: irc.User{
						Name:        s.env.twitchUserName,
						DisplayName: s.env.twitchUserName,
						ID:          s.env.twitchUserID,
					},
				}
				if reply {
					add.Reply = &irc.Reply{
						ParentMsgID:       e.ID,
						ParentUserID:      e.User.ID,
						ParentUserLogin:   e.User.Name,
						ParentDisplayName: e.User.DisplayName,
						ParentMsgBody:     e.Message,
					}
				}
				s.history[e.Channel] = append(s.history[e.Channel], &add)
				if reply {
					s.conversations[e.Channel] = append(s.conversations[e.Channel], &add)
					s.irc.Reply(e.Channel, e.ID, parts)
				} else {
					s.irc.Say(e.Channel, parts)
				}
			}
		}
	}(res)
}

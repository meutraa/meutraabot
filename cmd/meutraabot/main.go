package main

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	_ "github.com/lib/pq"
	ircevent "github.com/thoj/go-ircevent"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Message struct {
	Channel string
	Body    string
}

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

	done, err := s.PrepareIRC()
	if nil != err {
		return err
	}

	// Create a channel for the OS to notify us of interrupts/signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
		case <-done:
			return nil
		}
	}
}

func (s *Server) handleCommand(ctx context.Context, channel string, e *ircevent.Event) (string, bool) {
	sender := e.Nick
	text := e.Message()
	isMod := e.Tags["mod"] == "1" || ("#"+e.Nick == channel) || e.Nick == "meutraa"
	isSub := e.Tags["subscriber"] == "1"

	// This is the channel owner
	if "#"+sender == channel {
		if text == "!leave" {
			if err := s.q.DeleteChannel(ctx, channel); nil != err {
				return "Failed to leave channel", true
			}

			go func() {
				time.Sleep(1 * time.Second)
				s.irc.Part(channel)
			}()
			return "Bye bye 👋", true
		}
	}

	if text == "!join" {
		if err := s.q.CreateChannel(ctx, "#"+sender); nil != err {
			log.Println("unable to insert channel:", err)
			return "unable to join channel", true
		}

		s.irc.Join("#" + sender)
		s.irc.Privmsg("#"+sender, "Hi 🙋")
		return "I will be in #" + sender + " in just a moment", true
	}

	strs := strings.SplitN(text, " ", 2)

	if strs[0] == "!cmd" {
		if len(strs) == 1 {
			return "!cmd set|list|functions|variables", true
		}

		strs = strings.SplitN(strs[1], " ", 2)
		if strs[0] == "list" {
			commands, err := s.q.GetCommands(ctx, channel)
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
				"json(key string, json string)",
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
				tmpl, err := s.q.GetCommand(ctx, db.GetCommandParams{
					ChannelName: channel,
					Name:        strs[1],
				})
				if nil != err {
					log.Println("unable to get command", channel, strs[1], err)
				}
				return strs[1] + ": " + tmpl, true
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
						if err := s.q.DeleteCommand(ctx, db.DeleteCommandParams{
							ChannelName: channel,
							Name:        name,
						}); nil != err {
							log.Println("unable to delete command", name, err)
							return "unable to delete command", true
						}
						continue
					}
					if err := s.q.SetCommand(ctx, db.SetCommandParams{
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

	// Check to see if this matches a command
	var tmplStr string
	var command = strings.ToLower(strs[0])
	if command == "!test" && isMod {
		tmplStr = strs[1]
	} else {
		var err error
		commands, err := s.q.GetMatchingCommands(ctx, db.GetMatchingCommandsParams{
			ChannelName: channel,
			Message:     strings.ToLower(text),
		})
		if nil != err && err != sql.ErrNoRows {
			log.Println("unable to get command", err)
			return "", false
		}

		matchingCommands := make([]db.GetMatchingCommandsRow, 0, len(commands))
		for _, c := range commands {
			if c.Match {
				matchingCommands = append(matchingCommands, c)
			}
		}

		if len(matchingCommands) > 1 {
			commands := make([]string, len(matchingCommands))
			for i, c := range matchingCommands {
				commands[i] = c.Name
			}
			return "message matches multiple commands: " + strings.Join(commands, ", "), false
		}

		if len(matchingCommands) != 0 {
			log.Println("matched", matchingCommands[0].Name)
			tmplStr = matchingCommands[0].Template
		}
	}

	// Execute command
	if tmplStr != "" {
		variables := strings.Split(text, " ")[1:]
		data := Data{
			Channel:   strings.TrimPrefix(channel, "#"),
			IsMod:     isMod,
			IsOwner:   "#"+e.Nick == channel,
			IsSub:     isSub,
			MessageID: e.Tags["id"],
			User:      sender,
			ChannelID: e.Tags["room-id"],
			UserID:    e.Tags["user-id"],
			BotName:   s.env.twitchUsername,
			Command:   command,
			Arg:       variables,
		}

		tmpl, err := template.New(strs[0]).Funcs(s.FuncMap(ctx, e)).Parse(tmplStr)
		if err != nil {
			return "command template is broken: " + err.Error(), true
		}

		var out bytes.Buffer
		if err := tmpl.Execute(&out, data); nil != err {
			return "command executed wrongly: " + err.Error(), true
		}

		return out.String(), true
	}

	return "", false
}

func splitRecursive(str string) []string {
	if len(str) <= 480 {
		return []string{str}
	}
	return append([]string{string(str[0:480] + "…")}, splitRecursive(str[480:])...)
}

func (s *Server) handleMessage(e *ircevent.Event) {
	channel := e.Arguments[0]
	if s.env.twitchUsername == e.Nick {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.q.CreateUser(ctx, db.CreateUserParams{
		ChannelName: channel,
		Sender:      e.Nick,
	}); nil != err {
		log.Println("unable to create user", err)
	}

	if err := s.q.UpdateMetrics(ctx, db.UpdateMetricsParams{
		ChannelName: channel,
		Sender:      e.Nick,
		WordCount:   int64(len(strings.Split(e.Message(), " "))),
	}); nil != err {
		log.Println("unable to update metrics for user", err)
	}

	if err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ChannelName: channel,
		Sender:      e.Nick,
		Message:     e.Message(),
	}); nil != err {
		log.Println("unable to add message", err)
	}

	res, _ := s.handleCommand(ctx, channel, e)
	log.Printf("%v:%v:%v < %v\n", channel, e.Nick, e.Message(), res)
	res = strings.TrimSpace(res)
	if "" == res {
		return
	}
	for _, msg := range splitRecursive(res) {
		s.send <- Message{
			Channel: channel,
			Body:    msg,
		}
	}
}

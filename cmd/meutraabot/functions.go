package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v3"
	"github.com/hako/durafmt"
	"github.com/nicklaw5/helix/v2"
)

type Data struct {
	User                string              `json:".User"`
	UserID              string              `json:".UserID"`
	Channel             string              `json:".Channel"`
	ChannelID           string              `json:".ChannelID"`
	Message             string              `json:".Message"`
	MessageID           string              `json:".MessageID"`
	IsMod               bool                `json:".IsMod"`
	IsOwner             bool                `json:".IsOwner"`
	IsAdmin             bool                `json:".IsAdmin"`
	IsSub               bool                `json:".IsSub"`
	BotID               string              `json:".BotID"`
	Event               *irc.PrivateMessage `json:"-"`
	Command             string              `json:".Command"`
	Arg                 []string            `json:".Arg"`
	SelectedUser        string              `json:".SelectedUser"`
	SelectedUserID      string              `json:".SelectedUserID"`
	ReplyingToUser      string              `json:".ReplyingToUser"`
	ReplyingToUserID    string              `json:".ReplyingToUserID"`
	ReplyingToMessage   string              `json:".ReplyingToMessage"`
	ReplyingToMessageID string              `json:".ReplyingToMessageID"`
}

func (s *Server) FuncMap(ctx context.Context, d Data, e *irc.PrivateMessage) template.FuncMap {
	return template.FuncMap{
		"reply":      func(message string) string { return s.funcReply(ctx, d, message) },
		"user":       func() string { return s.funcUser(ctx, d) },
		"timeout":    func(duration int, reason string) string { return s.funcBan(ctx, d, duration, reason) },
		"ban":        func(reason string) string { return s.funcBan(ctx, d, 0, reason) },
		"delete":     func() string { return s.funcDelete(ctx, d, d.MessageID) },
		"clear":      func() string { return s.funcDelete(ctx, d, "") },
		"userfollow": func() string { return s.funcUserFollow(ctx, d) },
		"stream":     func() string { return s.funcStream(ctx, d) },
		"random":     func(max int) string { return s.funcRandom(ctx, d, max) },
		"duration":   func(time string) string { return s.funcDuration(ctx, d, time) },
		"get":        func(url string) string { return s.funcGet(ctx, d, url) },
		"json":       func(key, json string) string { return s.funcJsonParse(ctx, d, key, json) },
	}
}

func UserByName(client *helix.Client, username string) (helix.User, error) {
	resp, err := client.GetUsers(&helix.UsersParams{
		Logins: []string{username},
	})
	if err != nil {
		fmt.Println("unable to get user by id", err)
		return helix.User{}, err
	}
	if len(resp.Data.Users) == 0 {
		return helix.User{}, errors.New("unable to find user")
	}

	return resp.Data.Users[0], nil
}

func User(client *helix.Client, userID string) (helix.User, error) {
	res, err := Users(client, []string{userID})
	if err != nil {
		return helix.User{}, err
	}
	if len(res) == 0 {
		return helix.User{}, errors.New("unable to find user")
	}
	return res[0], nil
}

func Users(client *helix.Client, userIDs []string) ([]helix.User, error) {
	resp, err := client.GetUsers(&helix.UsersParams{
		IDs: userIDs,
	})
	if err != nil {
		fmt.Println("unable to get users by id", err)
		return []helix.User{}, err
	}

	return resp.Data.Users, nil
}

func (s *Server) funcUserFollow(ctx context.Context, d Data) string {
	resp, err := s.twitch.GetUsersFollows(&helix.UsersFollowsParams{
		First:  1,
		FromID: d.SelectedUserID,
		ToID:   d.ChannelID,
	})
	if err != nil {
		log(d.Channel, d.User, "unable to get user follow information for "+d.SelectedUser, err)
		return ""
	}

	if len(resp.Data.Follows) == 0 {
		log(d.Channel, d.User, "no follow information for "+d.SelectedUser, nil)
		return ""
	}

	data, err := json.Marshal(resp.Data.Follows[0])
	if err != nil {
		log(d.Channel, d.User, "unable to marshal user follow "+d.SelectedUser, err)
		return ""
	}
	return string(data)
}

func (s *Server) funcUser(ctx context.Context, d Data) string {
	user, err := User(s.twitch, d.SelectedUserID)
	if err != nil {
		log(d.Channel, d.User, "unable to get user "+d.SelectedUser, err)
		return ""
	}

	data, err := json.Marshal(user)
	if err != nil {
		log(d.Channel, d.User, "unable to marshal user "+d.SelectedUser, err)
		return ""
	}
	return string(data)
}

// https://dev.twitch.tv/docs/api/reference#delete-chat-messages
func (s *Server) funcDelete(ctx context.Context, d Data, messageID string) string {
	req, err := http.NewRequest(http.MethodDelete, "https://api.twitch.tv/helix/moderation/chat", nil)
	if err != nil {
		log(d.Channel, d.User, "unable to create request", err)
		return ""
	}

	req.Header.Add("Authorization", "Bearer "+strings.TrimPrefix(s.env.twitchOauthToken, "oauth:"))
	req.Header.Add("Client-Id", s.env.twitchClientID)

	q := req.URL.Query()
	q.Add("broadcaster_id", d.ChannelID)
	q.Add("moderator_id", s.env.twitchUserID)
	if "" != messageID {
		q.Add("message_id", messageID)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		log(d.Channel, d.User, "unable to do request", err)
		return ""
	}

	switch resp.StatusCode {
	case 204: // Success
	case 400: // Not allowed to delete mod/broadcaster messages
	case 403: // Not a mod
	case 404: // Message not found, or too old
		return ""
	case 401:
		return "unauthorized, check token scope for moderator:manage:chat_messages, or client-id"
	}
	return ""
}

// https://dev.twitch.tv/docs/api/reference#ban-user
func (s *Server) funcBan(ctx context.Context, d Data, duration int, reason string) string {
	type Data struct {
		Duration int    `json:"duration,omitempty"`
		Reason   string `json:"reason"`
		UserID   string `json:"user_id"`
	}
	type Request struct {
		Data Data `json:"data"`
	}

	reqBody, err := json.Marshal(Request{
		Data: Data{
			Duration: duration,
			Reason:   reason,
			UserID:   d.UserID,
		},
	})
	if err != nil {
		log(d.Channel, d.User, "unable to marshal requestbody ", err)
		return ""
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.twitch.tv/helix/moderation/bans",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		log(d.Channel, d.User, "unable to create request", err)
		return ""
	}

	req.Header.Add("Authorization", "Bearer "+strings.TrimPrefix(s.env.twitchOauthToken, "oauth:"))
	req.Header.Add("Client-Id", s.env.twitchClientID)
	req.Header.Add("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("broadcaster_id", d.ChannelID)
	q.Add("moderator_id", s.env.twitchUserID)
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		log(d.Channel, d.User, "unable to do request", err)
		return ""
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 400: // Bad request
		return "bad request"
	case 200: // Success
	case 403:
	case 409:
		return ""
	case 429:
		return "rate limited"
	case 401:
		return "unauthorized, check token scope for moderator:manage:banned_users, or client-id"
	}
	return ""
}

func (s *Server) funcRandom(ctx context.Context, d Data, max int) string {
	return strconv.Itoa(rand.Intn(max + 1))
}

func (s *Server) funcReply(ctx context.Context, d Data, message string) string {
	return "reply::delay::" + s.funcReplyAuto(ctx, d, message, false, func() string { return "" })
}

func (s *Server) funcReplyAuto(ctx context.Context, d Data, message string, useCustomPrompt bool, prompt func() string) string {
	if _, ok := s.conversations[d.Event.Channel]; !ok {
		s.conversations[d.Event.Channel] = make([]*irc.PrivateMessage, 0)
	}
	s.conversations[d.Event.Channel] = append(s.conversations[d.Event.Channel], d.Event)

	// get the channel settings
	settings, err := s.q.GetChannel(ctx, d.ChannelID)
	if err != nil {
		log(d.Channel, d.User, "unable to get channel settings", err)
		return ""
	}

	// Check if token matches valid regex
	if !settings.OpenaiToken.Valid || !regexp.MustCompile(`^sk-\w{48}$`).MatchString(settings.OpenaiToken.String) {
		return ""
	}

	channelData, err := s.Stream(ctx, d.ChannelID)
	if err != nil {
		log(d.Channel, d.User, "unable to get stream data", err)
	}

	safety := ""
	switch settings.ReplySafety {
	case 0:
		safety = "a very friendly viewer"
	case 1:
		safety = "a friendly, yet sassy viewer"
	case 2:
		safety = "an opinionated, sassy, but friendly viewer"
	case 3:
		safety = "an opinionated, sassy viewer"
	}

	p := "Meuua is " + safety + " in " + d.Channel + "'s twitch stream"
	if err == nil {
		p += " (" + channelData.GameName + " - " + channelData.Title + ")"
	}
	p += " chat replying to " + d.User + ".\n\n"

	if useCustomPrompt {
		p += prompt()
	} else {
		if d.ReplyingToMessage != "" {
			// find all the messages in this thread
			history, ok := s.conversations[d.Channel]
			if ok {
				if len(history) > 15 {
					history = history[len(history)-15:]
				}
				messages := []string{}
				for _, message := range history {
					messageStr := message.Message
					// twitch is stupid and replies are to themself
					if strings.HasPrefix(messageStr, "@"+message.User.Name+" ") {
						messageStr = strings.Replace(messageStr, "@"+message.User.Name+" ", "", 1)
					}
					messages = append(messages, message.User.Name+": "+messageStr+"\n")
				}
				for _, message := range messages {
					p += message
				}
			}
		} else {
			p += d.User + ": " + d.Message + "\n"
		}
	}
	p += "meuua:"

	log(d.Channel, d.User, p, nil)

	data := CompletionRequest{
		Prompt:           p,
		MaxTokens:        100,
		FrequencyPenalty: 2.0,
		PresencePenalty:  2.0,
		Temperature:      1.0,
		N:                1,
		TopP:             1.0,
		LogitBias: map[string]int{
			"19091": -100, // " viewer"
			"1177":  -100, // view
			"13120": -100, // friendly
			"8030":  -100, // " friendly"
			"1545":  -100, // " friend"
			"6726":  -100, // friend
			"33757": -100, // avage
			"562":   -100, // ass
			"10705": -100, // ASS
			"4107":  -100, // asy
			"11720": -100, // assy
			"4459":  -100, // " opinion"
			"9317":  -100, // " opinions"
		},
		Stop: ":",
		User: d.Channel,
	}
	jsonData, err := json.Marshal(data)
	log(d.Channel, d.User, "sending completion request"+string(jsonData), nil)
	if nil != err {
		log(d.Channel, d.User, "unable to do marshal ai request", err)
		return ""
	}

	// last chance to check that meuua has not already replied
	hist, okay := s.history[d.Channel]
	if okay && len(hist) > 0 && hist[len(hist)-1].User.ID == s.env.twitchUserID {
		log(d.Channel, d.User, "was last person to respond, so not doing completion request", nil)
		return ""
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/engines/text-davinci-002/completions", bytes.NewBuffer(jsonData))
	if nil != err {
		log(d.Channel, d.User, "unable to create completion request", err)
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+settings.OpenaiToken.String)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log(d.Channel, d.User, "unable to do completion request", err)
		return ""
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log(d.Channel, d.User, "unable to read body of ai response", err)
		return ""
	}

	var completion CompletionResponse
	if err := json.Unmarshal(res, &completion); nil != err {
		log(d.Channel, d.User, "unable to unmarshal ai response", err)
		return ""
	}

	if len(completion.Choices) == 0 {
		log(d.Channel, d.User, "ai has no responses", nil)
		return ""
	}

	str := completion.Choices[0].Text

	// remove the last line of the string
	lines := strings.Split(strings.TrimSpace(str), "\n")
	// remove last element in lines
	if len(lines) > 1 {
		lines = lines[:len(lines)-1]
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *Server) funcStream(ctx context.Context, d Data) string {
	stream, err := s.Stream(ctx, d.ChannelID)
	if err != nil {
		log(d.Channel, d.User, "unable to get stream information for "+d.Channel, err)
		return ""
	}

	data, err := json.Marshal(stream)
	if err != nil {
		log(d.Channel, d.User, "unable to marshal stream "+d.SelectedUser, err)
		return ""
	}
	return string(data)
}

func (s *Server) Stream(ctx context.Context, userId string) (helix.Stream, error) {
	resp, err := s.twitch.GetStreams(&helix.StreamsParams{
		UserIDs: []string{userId},
		First:   1,
	})
	if err != nil {
		return helix.Stream{}, err
	}
	if len(resp.Data.Streams) == 0 {
		return helix.Stream{}, errors.New("unable to find stream for user " + userId)
	}
	return resp.Data.Streams[0], nil
}

func (s *Server) funcDuration(ctx context.Context, d Data, startTime string) string {
	t, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		log(d.Channel, d.User, "unable to parse time "+startTime, err)
		return ""
	}

	return durafmt.Parse(time.Since(t)).LimitFirstN(3).String()
}

type BotsResponse struct {
	Bots  [][]interface{} `json:"bots"`
	Total int             `json:"_total"`
}

func getBotList() (*BotsResponse, error) {
	resp, err := http.Get("https://api.twitchinsights.net/v1/bots/all")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return nil, err
	}
	bots := BotsResponse{}
	if err := json.Unmarshal(body, &bots); nil != err {
		return nil, err
	}
	return &bots, err
}

func (s *Server) funcGet(ctx context.Context, d Data, url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		log(d.Channel, d.User, "unable to create request for "+url, err)
		return ""
	}
	req = req.WithContext(ctx)
	req.Header.Add("Accept", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log(d.Channel, d.User, "unable to get "+url, err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log(d.Channel, d.User, "unable to read body of "+url, err)
		return ""
	}
	str := strings.ReplaceAll(string(body), "\n", " ")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

func (s *Server) funcJsonParse(ctx context.Context, d Data, key, str string) string {
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(str), &m); err != nil {
		log(d.Channel, d.User, "unable to unmarshal", err)
		return ""
	}
	v := m[key]
	str, ok := v.(string)
	if !ok {
		data, err := json.Marshal(v)
		if nil != err {
			log(d.Channel, d.User, "unable to marshal", err)
			return ""
		}
		return string(data)
	}
	return str
}

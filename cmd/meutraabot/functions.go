package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v3"
	"github.com/hako/durafmt"
	"github.com/nicklaw5/helix/v2"
)

type Data struct {
	User           string   `json:".User"`
	UserID         string   `json:".UserID"`
	Channel        string   `json:".Channel"`
	ChannelID      string   `json:".ChannelID"`
	MessageID      string   `json:".MessageID"`
	IsMod          bool     `json:".IsMod"`
	IsOwner        bool     `json:".IsOwner"`
	IsAdmin        bool     `json:".IsAdmin"`
	IsSub          bool     `json:".IsSub"`
	BotID          string   `json:".BotID"`
	Command        string   `json:".Command"`
	Arg            []string `json:".Arg"`
	SelectedUser   string   `json:".SelectedUser"`
	SelectedUserID string   `json:".SelectedUserID"`
}

func firstOr(values []string, fallback string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return fallback
}

func (s *Server) FuncMap(ctx context.Context, d Data, e *irc.PrivateMessage) template.FuncMap {
	return template.FuncMap{
		"user":       func() string { return s.funcUser(ctx, d) },
		"userfollow": func() string { return s.funcUserFollow(ctx, d) },
		"stream":     func() string { return s.funcStream(ctx, d) },
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

func (s *Server) funcStream(ctx context.Context, d Data) string {
	resp, err := s.twitch.GetStreams(&helix.StreamsParams{
		UserIDs: []string{d.SelectedUserID},
		First:   1,
	})
	if err != nil {
		log(d.Channel, d.User, "unable to get stream "+d.SelectedUser, err)
		return ""
	}
	if len(resp.Data.Streams) == 0 {
		log(d.Channel, d.User, "no stream found for "+d.SelectedUser, err)
		return ""
	}
	stream := resp.Data.Streams[0]

	data, err := json.Marshal(stream)
	if err != nil {
		log(d.Channel, d.User, "unable to marshal stream "+d.SelectedUser, err)
		return ""
	}
	return string(data)
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/hako/durafmt"
	"github.com/nicklaw5/helix"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Data struct {
	User      string
	UserID    string
	Channel   string
	ChannelID string
	MessageID string
	IsMod     bool
	IsOwner   bool
	IsSub     bool
	BotName   string
	Command   string
	Arg       []string
}

func (s *Server) FuncMap(ctx context.Context, e *irc.PrivateMessage) template.FuncMap {
	return template.FuncMap{
		"rank":       func(user string) string { return s.funcRank(ctx, e.Channel, user) },
		"points":     func(user string) string { return s.funcPoints(ctx, e.Channel, user) },
		"activetime": func(user string) string { return s.funcActivetime(ctx, e.Channel, user) },
		"words":      func(user string) string { return s.funcWords(ctx, e.Channel, user) },
		"messages":   func(user string) string { return s.funcMessages(ctx, e.Channel, user) },
		"counter":    func(name string) string { return s.funcCounter(ctx, e.Channel, name) },
		"get":        func(url string) string { return s.funcGet(ctx, e.Channel, url) },
		"json":       func(key, json string) string { return s.funcJsonParse(e.Channel, key, json) },
		"top":        func(count string) string { return s.funcTop(ctx, e.Channel, count) },
		"followage":  func(user string) string { return s.funcFollowage(e.RoomID, user) },
		"uptime":     func() string { return s.funcUptime(e.RoomID) },
		"incCounter": func(name, change string) string { return s.funcIncCounter(ctx, e.Channel, name, change) },
	}
}

func (s *Server) funcTop(ctx context.Context, channel, count string) string {
	var c int32 = 5
	cnt, err := strconv.ParseInt(count, 10, 32)
	if nil == err {
		c = int32(cnt)
	}
	top, err := s.q.GetTopWatchers(ctx, db.GetTopWatchersParams{
		ChannelName: "#" + channel,
		Limit:       c,
	})
	if nil != err {
		log.Println("unable to get top watchers", channel, err)
		return ""
	}

	return strings.Join(top, ", ")
}

func (s *Server) funcUptime(channelID string) string {
	resp, err := s.twitch.GetStreams(&helix.StreamsParams{
		First:   1,
		UserIDs: []string{channelID},
	})
	if err != nil {
		log.Println("unable to get stream info", err)
		return ""
	}

	if len(resp.Data.Streams) == 0 {
		log.Println("no stream found for", channelID)
		return "not live"
	}

	start := resp.Data.Streams[0].StartedAt
	return durafmt.Parse(time.Now().Sub(start)).LimitFirstN(2).String()
}

func User(client *helix.Client, user string) (helix.User, error) {
	resp, err := client.GetUsers(&helix.UsersParams{
		Logins: []string{user},
	})
	if err != nil {
		log.Println("unable to get user", err)
		return helix.User{}, err
	}

	if len(resp.Data.Users) == 0 {
		log.Println("no user found for", user)
		return helix.User{}, errors.New("user not found")
	}

	return resp.Data.Users[0], nil
}

func (s *Server) funcFollowage(channelID, user string) string {
	u, err := User(s.twitch, user)
	if nil != err {
		return "can not find user " + user
	}
	resp, err := s.twitch.GetUsersFollows(&helix.UsersFollowsParams{
		First:  1,
		FromID: u.ID,
		ToID:   channelID,
	})
	if err != nil {
		log.Println("unable to get user follower info", err)
		return ""
	}

	if len(resp.Data.Follows) == 0 {
		log.Println("no follow info found for", channelID, user)
		return "user does not follow"
	}

	start := resp.Data.Follows[0].FollowedAt
	return durafmt.Parse(time.Now().Sub(start)).LimitFirstN(2).String()
}

func (s *Server) funcIncCounter(ctx context.Context, channel, name string, change string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()

	count, err := strconv.ParseInt(change, 10, 64)
	if nil != err {
		log.Println("unable to parse change count", err)
	}

	if err := s.q.UpdateCounter(ctx, db.UpdateCounterParams{
		ChannelName: "#" + channel,
		Name:        strings.ToLower(name),
		Value:       count,
	}); nil != err {
		log.Println("unable to update counter", channel, name, err)
	}
	return ""
}

func (s *Server) funcCounter(ctx context.Context, channel, name string) string {
	value, err := s.q.GetCounter(ctx, db.GetCounterParams{
		ChannelName: "#" + channel,
		Name:        strings.ToLower(name),
	})
	if nil != err {
		log.Println("unable to lookup counter", channel, name, err)
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func (s *Server) funcGet(ctx context.Context, channel, url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		log.Println("unable to create request for", url, err)
		return ""
	}
	req = req.WithContext(ctx)
	req.Header.Add("Accept", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("unable to get url", channel, url, err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Println("unable to read body of url", channel, url, err)
		return ""
	}
	str := strings.ReplaceAll(string(body), "\n", " ")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

func (s *Server) funcJsonParse(channel, key, str string) string {
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(str), &m); err != nil {
		log.Println("unable to unmarshal", err)
		return ""
	}
	data, err := json.Marshal(m[key])
	if nil != err {
		log.Println("unable to marshal data", err)
		return ""
	}
	return string(data)
}

func (s *Server) metrics(ctx context.Context, channel, user string, onMetrics func(db.GetMetricsRow) string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()
	metrics, err := s.q.GetMetrics(ctx, db.GetMetricsParams{
		ChannelName: "#" + channel,
		Sender:      strings.ToLower(user),
	})
	if nil != err {
		log.Println("unable to lookup user metrics", err)
		return ""
	}
	return onMetrics(metrics)
}

func (s *Server) funcPoints(ctx context.Context, channel, user string) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		return strconv.FormatInt(m.WatchTime/60+(m.WordCount/8), 10)
	})
}

func (s *Server) funcWords(ctx context.Context, channel, user string) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		return strconv.FormatInt(m.WordCount, 10)
	})
}

func (s *Server) funcActivetime(ctx context.Context, channel, user string) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		return fmt.Sprintf("%v", time.Duration(m.WatchTime*1000000000))
	})
}

func (s *Server) funcMessages(ctx context.Context, channel, user string) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		return strconv.FormatInt(m.MessageCount, 10)
	})
}

func (s *Server) funcRank(ctx context.Context, channel, user string) string {
	rank, err := s.q.GetWatchTimeRank(ctx, db.GetWatchTimeRankParams{
		ChannelName: "#" + channel,
		Sender:      strings.ToLower(user),
	})
	if nil != err {
		log.Println("unable to get user rank", channel, user, err)
		return ""
	}
	return strconv.FormatInt(int64(rank), 10)
}

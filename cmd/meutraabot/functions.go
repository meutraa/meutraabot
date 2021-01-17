package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	irc "github.com/gempir/go-twitch-irc/v2"
	"github.com/hako/durafmt"
	"github.com/meutraa/helix"
	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Data struct {
	User         string   `json:".User"`
	UserID       string   `json:".UserID"`
	Channel      string   `json:".Channel"`
	ChannelID    string   `json:".ChannelID"`
	MessageID    string   `json:".MessageID"`
	IsMod        bool     `json:".IsMod"`
	IsOwner      bool     `json:".IsOwner"`
	IsSub        bool     `json:".IsSub"`
	BotName      string   `json:".BotName"`
	Command      string   `json:".Command"`
	Arg          []string `json:".Arg"`
	SelectedUser string   `json:".SelectedUser"`
}

func pick(fallback string, values []string) string {
	for _, v := range values {
		if "" != v {
			return v
		}
	}
	return fallback
}

func (s *Server) FuncMap(ctx context.Context, d Data, e *irc.PrivateMessage) template.FuncMap {
	ch := strings.TrimLeft(e.Channel, "#")
	return template.FuncMap{
		"rank":               func() string { return s.funcRank(ctx, ch, d.SelectedUser, true) },
		"rank_alltime":       func() string { return s.funcRank(ctx, ch, d.SelectedUser, false) },
		"points":             func() string { return s.funcPoints(ctx, ch, d.SelectedUser, true) },
		"points_alltime":     func() string { return s.funcPoints(ctx, ch, d.SelectedUser, false) },
		"activetime":         func() string { return s.funcActivetime(ctx, ch, d.SelectedUser, false) },
		"activetime_average": func() string { return s.funcActivetime(ctx, ch, d.SelectedUser, true) },
		"words":              func() string { return s.funcWords(ctx, ch, d.SelectedUser, false) },
		"words_average":      func() string { return s.funcWords(ctx, ch, d.SelectedUser, true) },
		"messages":           func() string { return s.funcMessages(ctx, ch, d.SelectedUser, false) },
		"messages_average":   func() string { return s.funcMessages(ctx, ch, d.SelectedUser, true) },
		"counter":            func(name string) string { return s.funcCounter(ctx, ch, name) },
		"get":                func(url string) string { return s.funcGet(ctx, ch, url) },
		"json":               func(key, json string) string { return s.funcJsonParse(ch, key, json) },
		"top":                func() string { return s.funcTop(ctx, ch, pick("5", d.Arg), true) },
		"top_alltime":        func() string { return s.funcTop(ctx, ch, pick("5", d.Arg), false) },
		"followage":          func() string { return s.funcFollowage(e.RoomID, d.SelectedUser) },
		"uptime":             func() string { return s.funcUptime(e.RoomID) },
		"incCounter":         func(name, change string) string { return s.funcIncCounter(ctx, ch, name, change) },
	}
}

func (s *Server) funcTop(ctx context.Context, channel, count string, average bool) string {
	var c int32 = 5
	cnt, err := strconv.ParseInt(count, 10, 32)
	if nil == err {
		c = int32(cnt)
	}
	var top []string
	if !average {
		top, err = s.q.GetTopWatchers(ctx, db.GetTopWatchersParams{
			ChannelName: channel,
			Limit:       c,
		})
	} else {
		top, err = s.q.GetTopWatchersAverage(ctx, db.GetTopWatchersAverageParams{
			ChannelName: channel,
			Limit:       c,
		})
	}
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
	return durafmt.Parse(time.Now().Sub(start)).LimitFirstN(6).String()
}

func (s *Server) funcIncCounter(ctx context.Context, channel, name string, change string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()

	count, err := strconv.ParseInt(change, 10, 64)
	if nil != err {
		log.Println("unable to parse change count", err)
	}

	if err := s.q.UpdateCounter(ctx, db.UpdateCounterParams{
		ChannelName: channel,
		Name:        strings.ToLower(name),
		Value:       count,
	}); nil != err {
		log.Println("unable to update counter", channel, name, err)
	}
	return ""
}

func (s *Server) funcCounter(ctx context.Context, channel, name string) string {
	value, err := s.q.GetCounter(ctx, db.GetCounterParams{
		ChannelName: channel,
		Name:        strings.ToLower(name),
	})
	if nil != err {
		log.Println("unable to lookup counter", channel, name, err)
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func getBotList() ([]byte, error) {
	resp, err := http.Get("https://api.twitchinsights.net/v1/bots/online")
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return []byte{}, err
	}
	return body, err
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
		ChannelName: channel,
		Sender:      strings.ToLower(user),
	})
	if nil != err {
		log.Println("unable to lookup user metrics", err)
		return ""
	}
	return onMetrics(metrics)
}

func (s *Server) funcPoints(ctx context.Context, channel, user string, average bool) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		points := int64(m.Points)
		if average {
			points *= 1000000
			points = int64(math.Ceil(float64(points) / m.Age))
		}
		return strconv.FormatInt(points, 10)
	})
}

func (s *Server) funcWords(ctx context.Context, channel, user string, average bool) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		words := m.WordCount
		if average {
			words *= 1000000
			words = int64(math.Ceil(float64(words) / m.Age))
		}
		return strconv.FormatInt(words, 10)
	})
}

func (s *Server) funcActivetime(ctx context.Context, channel, user string, average bool) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		watch := m.WatchTime * 1000000000
		if average {
			watch = int64(math.Ceil(float64(watch) / m.Age))
		}
		return fmt.Sprintf("%v", time.Duration(watch))
	})
}

func (s *Server) funcMessages(ctx context.Context, channel, user string, average bool) string {
	return s.metrics(ctx, channel, user, func(m db.GetMetricsRow) string {
		messages := m.MessageCount
		if average {
			messages *= 1000000
			messages = int64(math.Ceil(float64(messages) / m.Age))
		}
		return strconv.FormatInt(messages, 10)
	})
}

func (s *Server) funcRank(ctx context.Context, channel, user string, average bool) string {
	var rank int32
	var err error
	if !average {
		rank, err = s.q.GetWatchTimeRank(ctx, db.GetWatchTimeRankParams{
			ChannelName: channel,
			Sender:      strings.ToLower(user),
		})
	} else {
		rank, err = s.q.GetWatchTimeRankAverage(ctx, db.GetWatchTimeRankAverageParams{
			ChannelName: channel,
			Sender:      strings.ToLower(user),
		})
	}
	if nil != err {
		log.Println("unable to get user rank", channel, user, err)
		return ""
	}
	return strconv.FormatInt(int64(rank), 10)
}

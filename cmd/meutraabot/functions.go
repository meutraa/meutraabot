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
	"github.com/nicklaw5/helix/v2"
	"gitlab.com/meutraa/meutraabot/pkg/db"
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
	ch := e.RoomID
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
		"get":                func(url string) string { return s.funcGet(ctx, url) },
		"json":               func(key, json string) string { return s.funcJsonParse(key, json) },
		"top":                func() string { return s.funcTop(ctx, ch, firstOr(d.Arg, "5"), true) },
		"top_alltime":        func() string { return s.funcTop(ctx, ch, firstOr(d.Arg, "5"), false) },
		"followage":          func() string { return s.funcFollowage(e.RoomID, d.SelectedUserID) },
		"uptime":             func() string { return s.funcUptime(e.RoomID) },
		"incCounter":         func(name, change string) string { return s.funcIncCounter(ctx, ch, name, change) },
	}
}

func (s *Server) funcTop(ctx context.Context, channelID, count string, average bool) string {
	var c int32 = 5
	cnt, err := strconv.ParseInt(count, 10, 32)
	if nil == err {
		c = int32(cnt)
	}
	var top []string
	if !average {
		top, err = s.q.GetTopWatchers(ctx, db.GetTopWatchersParams{
			ChannelID: channelID,
			Limit:     c,
		})
	} else {
		top, err = s.q.GetTopWatchersAverage(ctx, db.GetTopWatchersAverageParams{
			ChannelID: channelID,
			Limit:     c,
		})
	}
	if nil != err {
		log.Println("unable to get top watchers", channelID, err)
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
	return durafmt.Parse(time.Since(start)).LimitFirstN(2).String()
}

func UserByName(client *helix.Client, username string) (helix.User, error) {
	resp, err := client.GetUsers(&helix.UsersParams{
		Logins: []string{username},
	})
	if err != nil {
		log.Println("unable to get user by id", err)
		return helix.User{}, err
	}
	if len(resp.Data.Users) == 0 {
		// log.Println("unable to get user by id", err)
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
		log.Println("unable to get users by id", err)
		return []helix.User{}, err
	}

	return resp.Data.Users, nil
}

func (s *Server) funcFollowage(channelID, userID string) string {
	resp, err := s.twitch.GetUsersFollows(&helix.UsersFollowsParams{
		First:  1,
		FromID: userID,
		ToID:   channelID,
	})
	if err != nil {
		log.Println("unable to get user follower info", err)
		return ""
	}

	if len(resp.Data.Follows) == 0 {
		log.Println("no follow info found for", channelID, userID)
		return "user does not follow"
	}

	start := resp.Data.Follows[0].FollowedAt
	return durafmt.Parse(time.Since(start)).LimitFirstN(6).String()
}

func (s *Server) funcIncCounter(ctx context.Context, channelID, name, change string) string {
	context, cancel := context.WithTimeout(ctx, time.Duration(time.Second*2))
	defer cancel()
	count, err := strconv.ParseInt(change, 10, 64)
	if nil != err {
		log.Println("unable to parse change count", err)
	}

	if err := s.q.UpdateCounter(context, db.UpdateCounterParams{
		ChannelID: channelID,
		Name:      strings.ToLower(name),
		Value:     count,
	}); nil != err {
		log.Println("unable to update counter", channelID, name, err)
	}
	return ""
}

func (s *Server) funcCounter(ctx context.Context, channelID, name string) string {
	value, err := s.q.GetCounter(ctx, db.GetCounterParams{
		ChannelID: channelID,
		Name:      strings.ToLower(name),
	})
	if nil != err {
		log.Println("unable to lookup counter", channelID, name, err)
		return ""
	}
	return strconv.FormatInt(value, 10)
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

func (s *Server) funcGet(ctx context.Context, url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		log.Println("unable to create request for", url, err)
		return ""
	}
	req = req.WithContext(ctx)
	req.Header.Add("Accept", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("unable to get url", url, err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		log.Println("unable to read body of url", url, err)
		return ""
	}
	str := strings.ReplaceAll(string(body), "\n", " ")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

func (s *Server) funcJsonParse(key, str string) string {
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

func (s *Server) metrics(ctx context.Context, channelID, userID string, onMetrics func(db.GetMetricsRow) string) string {
	context, cancel := context.WithTimeout(ctx, time.Duration(time.Second*2))
	defer cancel()
	metrics, err := s.q.GetMetrics(context, db.GetMetricsParams{
		ChannelID: channelID,
		SenderID:  userID,
	})
	if nil != err {
		log.Println("unable to lookup user metrics", err)
		return ""
	}
	return onMetrics(metrics)
}

func (s *Server) funcPoints(ctx context.Context, channelID, userID string, average bool) string {
	return s.metrics(ctx, channelID, userID, func(m db.GetMetricsRow) string {
		points := int64(m.Points)
		if average {
			points *= 1000000
			points = int64(math.Ceil(float64(points) / m.Age))
		}
		return strconv.FormatInt(points, 10)
	})
}

func (s *Server) funcWords(ctx context.Context, channelID, userID string, average bool) string {
	return s.metrics(ctx, channelID, userID, func(m db.GetMetricsRow) string {
		words := m.WordCount
		if average {
			words *= 1000000
			words = int64(math.Ceil(float64(words) / m.Age))
		}
		return strconv.FormatInt(words, 10)
	})
}

func (s *Server) funcActivetime(ctx context.Context, channelID, userID string, average bool) string {
	return s.metrics(ctx, channelID, userID, func(m db.GetMetricsRow) string {
		watch := m.WatchTime * 1000000000
		if average {
			watch = int64(math.Ceil(float64(watch) / m.Age))
		}
		return fmt.Sprintf("%v", time.Duration(watch))
	})
}

func (s *Server) funcMessages(ctx context.Context, channelID, userID string, average bool) string {
	return s.metrics(ctx, channelID, userID, func(m db.GetMetricsRow) string {
		messages := m.MessageCount
		if average {
			messages *= 1000000
			messages = int64(math.Ceil(float64(messages) / m.Age))
		}
		return strconv.FormatInt(messages, 10)
	})
}

func (s *Server) funcRank(ctx context.Context, channelID, userID string, average bool) string {
	var rank int32
	var err error
	if !average {
		rank, err = s.q.GetWatchTimeRank(ctx, db.GetWatchTimeRankParams{
			ChannelID: channelID,
			SenderID:  userID,
		})
	} else {
		rank, err = s.q.GetWatchTimeRankAverage(ctx, db.GetWatchTimeRankAverageParams{
			ChannelID: channelID,
			SenderID:  userID,
		})
	}
	if nil != err {
		log.Println("unable to get user rank", channelID, userID, err)
		return ""
	}
	return strconv.FormatInt(int64(rank), 10)
}

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

	"github.com/hako/durafmt"
	"github.com/nicklaw5/helix"
	ircevent "github.com/thoj/go-ircevent"
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

func FuncMap(client *helix.Client, e *ircevent.Event) template.FuncMap {
	channel := e.Arguments[0]
	roomID := e.Tags["room-id"]

	rankFunc := func(user string) string { return rank(channel, user) }
	pointFunc := func(user string) string { return points(channel, user) }
	activetimeFunc := func(user string) string { return activetime(channel, user) }
	wordsFunc := func(user string) string { return words(channel, user) }
	messagesFunc := func(user string) string { return messages(channel, user) }
	counterFunc := func(name string) string { return counter(channel, name) }
	getFunc := func(url string) string { return get(channel, url) }
	jsonParseFunc := func(key, json string) string { return jsonparse(channel, key, json) }
	topFunc := func(count string) string { return top(channel, count) }
	uptimeFunc := func() string { return uptime(client, roomID) }
	followageFunc := func(user string) string { return followage(client, roomID, user) }
	incCounterFunc := func(name, change string) string { return incCounter(channel, name, change) }

	return template.FuncMap{
		"rank":       rankFunc,
		"points":     pointFunc,
		"activetime": activetimeFunc,
		"words":      wordsFunc,
		"messages":   messagesFunc,
		"counter":    counterFunc,
		"get":        getFunc,
		"json":       jsonParseFunc,
		"top":        topFunc,
		"followage":  followageFunc,
		"uptime":     uptimeFunc,
		"incCounter": incCounterFunc,
	}
}

// case "!emoji":
// 	return emojiHandler(c, channel, sender, text)

func top(channel, count string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()

	var c int32 = 5
	cnt, err := strconv.ParseInt(count, 10, 32)
	if nil == err {
		c = int32(cnt)
	}
	top, err := q.GetTopWatchers(ctx, db.GetTopWatchersParams{
		ChannelName: channel,
		Limit:       c,
	})
	if nil != err {
		log.Println("unable to get top watchers", channel, err)
		return ""
	}

	return strings.Join(top, ", ")
}

func uptime(client *helix.Client, channelID string) string {
	resp, err := client.GetStreams(&helix.StreamsParams{
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

func followage(client *helix.Client, channelID, user string) string {
	u, err := User(client, user)
	if nil != err {
		return "can not find user " + user
	}
	resp, err := client.GetUsersFollows(&helix.UsersFollowsParams{
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

func incCounter(channel, name string, change string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()

	count, err := strconv.ParseInt(change, 10, 64)
	if nil != err {
		log.Println("unable to parse change count", err)
	}

	if err := q.UpdateCounter(ctx, db.UpdateCounterParams{
		ChannelName: channel,
		Name:        strings.ToLower(name),
		Value:       count,
	}); nil != err {
		log.Println("unable to update counter", channel, name, err)
	}
	return ""
}

func counter(channel, name string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()
	value, err := q.GetCounter(ctx, db.GetCounterParams{
		ChannelName: channel,
		Name:        strings.ToLower(name),
	})
	if nil != err {
		log.Println("unable to lookup counter", channel, name, err)
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func get(channel, url string) string {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		log.Println("unable to create request for", url, err)
		return ""
	}
	req.Header.Add("Accept", "text/plain")
	resp, err := client.Do(req)
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

func jsonparse(channel, key, str string) string {
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

func metrics(channel, user string, onMetrics func(db.GetMetricsRow) string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()
	metrics, err := q.GetMetrics(ctx, db.GetMetricsParams{
		ChannelName: channel,
		Sender:      strings.ToLower(user),
	})
	if nil != err {
		log.Println("unable to lookup user metrics", err)
		return ""
	}
	return onMetrics(metrics)
}

func points(channel, user string) string {
	return metrics(channel, user, func(m db.GetMetricsRow) string {
		return strconv.FormatInt(m.WatchTime/60+(m.WordCount/8), 10)
	})
}

func words(channel, user string) string {
	return metrics(channel, user, func(m db.GetMetricsRow) string {
		return strconv.FormatInt(m.WordCount, 10)
	})
}

func activetime(channel, user string) string {
	return metrics(channel, user, func(m db.GetMetricsRow) string {
		return fmt.Sprintf("%v", time.Duration(m.WatchTime*1000000000))
	})
}

func messages(channel, user string) string {
	return metrics(channel, user, func(m db.GetMetricsRow) string {
		return strconv.FormatInt(m.MessageCount, 10)
	})
}

func rank(channel, user string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	defer cancel()

	rank, err := q.GetWatchTimeRank(ctx, db.GetWatchTimeRankParams{
		ChannelName: channel,
		Sender:      strings.ToLower(user),
	})
	if nil != err {
		log.Println("unable to get user rank", channel, user, err)
		return ""
	}
	return strconv.FormatInt(int64(rank), 10)
}

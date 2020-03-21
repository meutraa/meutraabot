package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/meutraa/meutraabot/pkg/db"
)

type Data struct {
	User    string
	Channel string
	IsMod   bool
	IsSub   bool
	BotName string
	Arg     []string
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
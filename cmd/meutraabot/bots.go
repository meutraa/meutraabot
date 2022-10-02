package main

import (
	"encoding/json"
	"io/ioutil"
	l "log"
	"net/http"
	"strconv"
)

type BotsResponse struct {
	Bots  [][]interface{} `json:"bots"`
	Total int             `json:"_total"`
}

type Bot struct {
	Username     string
	ChannelCount int
	UserID       string
}

func getBotList() ([]Bot, error) {
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

	list := make([]Bot, 0, len(bots.Bots))

	for _, bot := range bots.Bots {
		username, ok := bot[0].(string)
		if !ok {
			l.Println("unable to cast bot name")
			continue
		}

		userIDVal, ok := bot[2].(float64)
		if !ok {
			l.Println("unable to cast bot id")
			continue
		}
		userID := strconv.Itoa(int(userIDVal))

		channelCount, ok := bot[1].(float64)
		if !ok {
			l.Println("unable to cast bot channel count")
			continue
		}

		list = append(list, Bot{
			Username:     username,
			ChannelCount: int(channelCount),
			UserID:       userID,
		})
	}

	return list, nil
}

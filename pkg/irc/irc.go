package irc

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func (client *Client) SendMessage(channel, message string) error {
	msg := []byte("PRIVMSG " + channel + " :" + message)
	log.Println(">", string(msg))
	err := client.c.WriteMessage(websocket.TextMessage, msg)
	if nil != err {
		return errors.Wrap(err, "Unable to send message:")
	}
	return nil
}

func (client *Client) Depart() error {
	err := client.c.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	if err != nil {
		return errors.Wrap(err, "Unable to send depart message")
	}
	return nil
}

func (client *Client) JoinChannel(channel string) error {
	err := client.c.WriteMessage(
		websocket.TextMessage,
		[]byte("JOIN "+channel),
	)
	if nil != err {
		return errors.Wrap(err, "Unable to join channel")
	}
	return nil
}

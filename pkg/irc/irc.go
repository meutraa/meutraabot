package irc

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func (client *Client) SendMessage(channel, message string) error {
	msg := []byte("PRIVMSG " + channel + " :" + message)
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

// channel requires # preppended to.
func (client *Client) JoinChannel(channel string) error {
	err := client.c.WriteMessage(
		websocket.TextMessage,
		[]byte("JOIN "+channel),
	)
	if nil != err {
		return errors.Wrap(err, "Unable to join channel: "+channel)
	}
	return nil
}

// channel requires # preppended to.
func (client *Client) PartChannel(channel string) error {
	err := client.c.WriteMessage(
		websocket.TextMessage,
		[]byte("PART "+channel),
	)
	if nil != err {
		return errors.Wrap(err, "Unable to part channel: "+channel)
	}
	return nil
}

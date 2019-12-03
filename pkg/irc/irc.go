package irc

import (
	"github.com/gorilla/websocket"
)

func (client *Client) SendMessage(channel, message string) error {
	return client.c.WriteMessage(websocket.TextMessage,
		[]byte("PRIVMSG "+channel+" :"+message))
}

func (client *Client) Depart() error {
	return client.c.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
}

// channel requires # preppended to.
func (client *Client) JoinChannel(channel string) error {
	return client.c.WriteMessage(websocket.TextMessage,
		[]byte("JOIN "+channel),
	)
}

// channel requires # preppended to.
func (client *Client) PartChannel(channel string) error {
	return client.c.WriteMessage(websocket.TextMessage,
		[]byte("PART "+channel),
	)
}

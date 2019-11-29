package irc

import (
	"log"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type Client struct {
	c *websocket.Conn
}

func NewClient() (*Client, error) {
	u := url.URL{
		Scheme: "wss",
		Host:   "irc-ws.chat.twitch.tv",
	}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return &Client{}, errors.Wrap(err, "Can't establish websocket connection")
	}

	return &Client{c: c}, nil
}

func (client *Client) Close() {
	client.c.Close()
}

func (client *Client) Authenticate(oauth string) error {
	if err := client.c.WriteMessage(websocket.TextMessage, []byte("PASS "+oauth)); nil != err {
		return errors.Wrap(err, "Unable to send oauth token")
	}
	if err := client.c.WriteMessage(websocket.TextMessage, []byte("NICK meutraa")); nil != err {
		return errors.Wrap(err, "Unable to send NICK message")
	}

	return nil
}

func (client *Client) SetMessageChannel(messages chan *PrivateMessage, done chan bool) {
	// If we stop reading this socket connection, close the program
	defer func() {
		done <- true
	}()

	for {
		t, message, err := client.c.ReadMessage()
		if err != nil {
			log.Println("Read message error:", err)
			return
		}
		if websocket.TextMessage != t {
			continue
		}

		go client.handleMessage(message, messages)
	}
}

func (client *Client) handleMessage(message []byte, messages chan *PrivateMessage) {
	msg := string(message)
	log.Println("<", msg)

	if strings.HasPrefix(msg, ":") {
		parts := strings.Split(msg, ":")
		sender := strings.Split(parts[1], "!")[0]

		parameters := strings.Split(parts[1], " ")
		command := parameters[1]
		channel := parameters[2]

		switch command {
		case "JOIN":
			log.Println("Joined channel:", channel)
		case "PRIVMSG":
			text := strings.SplitAfterN(msg, ":", 3)[2]
			text = strings.TrimSuffix(text, "\n")
			text = strings.TrimSuffix(text, "\r")
			messages <- &PrivateMessage{
				Channel:         channel,
				Sender:          sender,
				Message:         strings.ToLower(text),
				OriginalMessage: text,
			}
		}
		return
	}

	// If the server is pinging us, respond
	if strings.HasPrefix(msg, "PING") {
		err := client.c.WriteMessage(websocket.TextMessage, []byte("PONG "+msg[5:]))
		if nil != err {
			log.Println("Unable to write PONG message:", err)
		}
		return
	}
}

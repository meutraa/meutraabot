package irc

import (
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type Client struct {
	c *websocket.Conn
}

func NewClient() (*Client, error) {
	u := url.URL{Scheme: "wss", Host: "irc-ws.chat.twitch.tv"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return &Client{c: c}, err
}

func (client *Client) Close() {
	client.c.Close()
}

func (client *Client) Authenticate(username, oauth string) error {
	if err := client.c.WriteMessage(websocket.TextMessage, []byte("PASS "+oauth)); nil != err {
		return errors.Wrap(err, "Unable to send oauth token")
	}
	if err := client.c.WriteMessage(websocket.TextMessage, []byte("NICK "+username)); nil != err {
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

		msg := string(message)
		log.Println("<", msg)

		if strings.HasPrefix(msg, ":") {
			parts := strings.Split(msg, ":")
			parameters := strings.Split(parts[1], " ")

			if parameters[1] == "PRIVMSG" {
				text := strings.TrimSuffix(strings.SplitAfterN(msg, ":", 3)[2], "\r\n")
				messages <- &PrivateMessage{
					Channel:         parameters[2],
					Sender:          strings.Split(parts[1], "!")[0],
					ReceivedTime:    time.Now(),
					Message:         strings.ToLower(text),
					OriginalMessage: text,
				}
			}
			continue
		}

		// If the server is pinging us, respond
		if strings.HasPrefix(msg, "PING") {
			if err := client.c.WriteMessage(websocket.TextMessage, []byte("PONG "+msg[5:])); nil != err {
				log.Println("Unable to write PONG message:", err)
			}
		}
	}
}

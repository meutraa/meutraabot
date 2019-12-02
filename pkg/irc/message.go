package irc

import "time"

type PrivateMessage struct {
	Channel         string
	Sender          string
	Message         string
	ReceivedTime    time.Time
	OriginalMessage string
}

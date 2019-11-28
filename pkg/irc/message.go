package irc

type PrivateMessage struct {
	Channel string
	Sender  string
	Message string
}

func (msg PrivateMessage) String() string {
	return msg.Sender + " (" + msg.Channel + "):"
}

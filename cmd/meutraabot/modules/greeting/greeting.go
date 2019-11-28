package greeting

import (
	"math/rand"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

var greetings = [...]string{
	"Hey",
	"Hello",
	"Howdy",
	"Salutations",
	"Greetings",
	"Hi",
	"Welcome",
	"Good day",
}

func random() string {
	return greetings[rand.Intn(len(greetings)-1)]
}

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if sender == "meutraa" {
		return "", false, nil
	}
	switch text {
	case "hey", "hello", "howdy", "hi":
		fallthrough
	case "hey!", "hello!", "howdy!", "hi!":
		fallthrough
	case "hey.", "hello.", "howdy.", "hi.":
		return random() + " " + sender + "!", true, nil
	}
	return "", false, nil
}

package back

import (
	"strings"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if sender == "meutraa" {
		return "", false, nil
	}
	valid := strings.HasPrefix(text, "i'm back") ||
		strings.HasPrefix(text, "i am back") ||
		text == "back" ||
		strings.HasPrefix(text, "im back")

	if valid {
		return "Hi back, I thought your name was " + sender + " 🤔.", true, nil
	}
	return "", false, nil
}

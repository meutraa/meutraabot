package sleep

import (
	"strings"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if sender == "meutraa" {
		return "", false, nil
	}
	valid := (strings.HasPrefix(text, "😴") ||
		strings.Contains(text, "sleep")) &&
		!strings.Contains(text, "no sleep") &&
		!strings.Contains(text, "not sleep")

	if valid {
		return "No sleep.", true, nil
	}
	return "", false, nil
}

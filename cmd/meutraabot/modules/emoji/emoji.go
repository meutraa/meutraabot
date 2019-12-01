package emoji

import (
	"log"
	"strings"
	"unicode/utf8"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if !strings.HasPrefix(text, "!emoji") {
		return "", false, nil
	}

	parts := strings.Split(text, " ")
	if len(parts) != 2 {
		return "", true, nil
	}

	// Check emoji is a valid character
	rune, _ := utf8.DecodeRuneInString(parts[1])
	if rune == utf8.RuneError {
		log.Println("Unable to parse rune from emoji")
		return "", true, nil
	}

	topUsers, err := db.UsersWithTopWatchTime(channel, 3)
	if nil != err {
		log.Println("Unable to get top 3 users", err)
		return "", true, nil
	}

	// Check user is in top 3
	isTop := false
	for _, user := range topUsers {
		if user.Sender == sender {
			isTop = true
			break
		}
	}
	if !isTop {
		return "Must be in top 3 of leaderboard to use this command", true, nil
	}

	emoji := string(rune)
	log.Println("Setting emoji for user,", sender, "to", emoji)
	err = db.SetEmoji(channel, sender, emoji)
	if nil != err {
		log.Println("Unable to set Emoji for user", sender, ":", err)
		return "", true, nil
	}

	return "", true, nil
}

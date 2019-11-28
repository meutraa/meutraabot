package vulpesmusketeer

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func isSpecialUser(sender string) bool {
	return sender == "casweets" ||
		sender == "meutraa" ||
		sender == "vulpesmusketeer" ||
		sender == "biological" ||
		sender == "tristantwist_"
}

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if channel != "#vulpesmusketeer" {
		return "", false, nil
	}

	if text == "h" {
		db.AddToIntChannel(channel, "hiccup_count", 1)
		return "", true, nil
	}

	// Only commands for these guys
	if isSpecialUser(sender) {
		if len(text) >= 2 && text[:2] == "h " {
			countStr := text[2:]
			count, err := strconv.ParseInt(countStr, 10, 64)
			if nil != err {
				return "Unable to parse count!: " + err.Error(), true, nil
			}

			db.AddToIntChannel(channel, "hiccup_count", count)
			return "", true, nil
		}
	}

	if strings.HasPrefix(text, "!hiccups") {
		count := db.GetIntChannel(channel, "hiccup_count")
		return fmt.Sprintf("Casweets has hiccuped %v times!", count), true, nil
	}
	return "", false, nil
}

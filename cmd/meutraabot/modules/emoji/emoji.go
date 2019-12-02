package emoji

import (
	"log"
	"strings"
	"unicode/utf8"

	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	. "github.com/volatiletech/sqlboiler/queries/qm"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/models"
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

	topUsers, err := models.Users(
		models.UserWhere.ChannelName.EQ(channel),
		OrderBy(models.UserColumns.WatchTime+" DESC"),
		Limit(3),
	).All(db.Context, db.DB)
	if nil != err {
		log.Println("Unable to get top 3 users", err)
		return "", true, nil
	}

	// Check user is in top 3
	var user *models.User
	for _, u := range topUsers {
		if u.Sender == sender {
			user = u
			break
		}
	}
	if nil == user {
		return "Must be in top 3 of leaderboard to use this command", true, nil
	}

	user.Emoji = null.String{String: string(rune), Valid: true}

	err = user.Update(db.Context, db.DB, boil.Whitelist(models.UserColumns.Emoji))
	if nil != err {
		log.Println("Unable to set Emoji for user", sender, ":", err)
		return "", true, nil
	}

	return "", true, nil
}

package messages

import (
	"fmt"
	"log"

	. "github.com/volatiletech/sqlboiler/queries/qm"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/models"
)

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text != "!messages" {
		return "", false, nil
	}

	user, err := models.Users(
		Where(
			models.UserColumns.ChannelName+" = ? AND "+models.UserColumns.Sender+" = ?",
			channel, sender),
	).One(db.Context, db.DB)

	if nil != err {
		log.Println("Unable to lookup userfor message_count", err)
		return "", true, nil
	}

	return fmt.Sprintf(
		"%v has sent %v messages",
		sender,
		user.MessageCount,
	), true, nil
}

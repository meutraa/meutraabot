package vulpesmusketeer

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/volatiletech/sqlboiler/boil"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/models"
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
		channel, err := models.FindChannel(db.Context, db.DB, channel)
		if nil != err {
			log.Println("Unable to find channel", err)
			return "", true, nil
		}

		channel.HiccupCount += 1

		err = channel.Update(db.Context, db.DB, boil.Whitelist(models.ChannelColumns.HiccupCount))
		if nil != err {
			log.Println("Unable to update hiccup_count", err)
		}
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

			channel, err := models.FindChannel(db.Context, db.DB, channel)
			if nil != err {
				log.Println("Unable to find channel", err)
				return "", true, nil
			}

			channel.HiccupCount += count

			err = channel.Update(db.Context, db.DB, boil.Whitelist(models.ChannelColumns.HiccupCount))
			if nil != err {
				log.Println("Unable to update hiccup_count", err)
			}

			return "", true, nil
		}
	}

	if strings.HasPrefix(text, "!hiccups") {
		channel, err := models.FindChannel(db.Context, db.DB, channel)
		if nil != err {
			log.Println("Unable to find channel", err)
			return "", true, nil
		}
		return fmt.Sprintf("Casweets has hiccuped %v times!", channel.HiccupCount), true, nil
	}
	return "", false, nil
}

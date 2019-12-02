package watchtime

import (
	"fmt"
	"log"
	"strings"

	. "github.com/volatiletech/sqlboiler/queries/qm"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/models"
)

func periodString(user string, seconds int64) string {
	rs := seconds
	days := rs / (60 * 60 * 24)
	rs -= days * (60 * 60 * 24)
	hours := rs / (60 * 60)
	rs -= hours * (60 * 60)
	minutes := rs / 60

	str := ""
	if days > 1 {
		str = fmt.Sprintf("%v days", days)
	} else if days == 1 {
		str = fmt.Sprintf("%v day", days)
	}

	if hours > 1 {
		str += fmt.Sprintf(" %v hours", hours)
	} else if hours == 1 {
		str += fmt.Sprintf(" %v hour", hours)
	}

	if minutes > 1 {
		str += fmt.Sprintf(" %v minutes", minutes)
	} else if minutes == 1 {
		str += fmt.Sprintf(" %v minute", minutes)
	}

	str = strings.Trim(strings.ReplaceAll(str, "  ", " "), " ")

	return fmt.Sprintf("%v has been active for %v", user, str)
}

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text != "!watchtime" {
		return "", false, nil
	}

	user, err := models.Users(
		Where(
			models.UserColumns.ChannelName+" = ? AND "+models.UserColumns.Sender+" = ?",
			channel, sender),
	).One(db.Context, db.DB)

	if nil != err {
		log.Println("Unable to lookup user for watch_time", err)
		return "", true, nil
	}

	return periodString(sender, user.WatchTime), true, nil
}

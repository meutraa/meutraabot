package messages

import (
	"fmt"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text != "!messages" {
		return "", false, nil
	}
	return fmt.Sprintf(
		"%v has sent %v messages",
		sender,
		db.GetIntUserMetric(channel, sender, "message_count"),
	), true, nil
}

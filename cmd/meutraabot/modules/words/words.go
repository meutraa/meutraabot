package words

import (
	"fmt"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func Response(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text != "!words" {
		return "", false, nil
	}
	return fmt.Sprintf(
		"%v has written %v words",
		sender,
		db.GetIntUserMetric(channel, sender, "word_count"),
	), true, nil
}

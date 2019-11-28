package commands

import (
	"log"
	"os"

	"gitlab.com/meutraa/meutraabot/data"
)

type ResponseFunc = func(db *data.Database, text, channel, sender string) (string, bool, error)

func ReadEnv(key string) string {
	value := os.Getenv(key)
	if "" == value {
		log.Fatalln("Unable to read", key, "from environment")
	}
	return value
}

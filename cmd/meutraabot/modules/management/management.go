package management

import (
	"os"
	"time"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

type RestartError struct {
}

func (e RestartError) Error() string {
	return "Restart requested"
}

const version = "1.2.1"

var metrics = CodeMetrics{
	Lines:      663,
	Words:      2275,
	Characters: 17740,
}

func VersionResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	if sender != "meutraa" {
		return "", false, nil
	}
	if text == "!version" {
		return version, true, nil
	}
	return "", false, nil
}

func CodeResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text == "!code" {
		return metrics.String(), true, nil
	}
	return "", false, nil
}

func RestartResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	if sender != "meutraa" {
		return "", false, nil
	}
	if text == "!restart" {
		go func() {
			time.Sleep(5 * time.Second)
			os.Exit(0)
		}()
		return "Restarting in 5 seconds", true, RestartError{}
	}
	return "", false, nil
}

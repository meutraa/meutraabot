package management

import (
	"fmt"
	"os"
	"time"

	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/env"
	"gitlab.com/meutraa/meutraabot/pkg/models"
)

type RestartError struct {
}

type PartError struct {
}

func (e RestartError) Error() string {
	return "Restart requested"
}

func (e PartError) Error() string {
	return "Part channel requested"
}

const version = "1.4.0"

const sloc = 2722

func VersionResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text == "!version" {
		return version, true, nil
	}
	return "", false, nil
}

func CodeResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text != "!code" {
		return "", false, nil
	}
	return fmt.Sprintf("%v lines of code", sloc), true, nil
}

func LeaveResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	if text != "!leave" || "#"+sender != channel {
		return "", false, nil
	}

	ch := models.Channel{ChannelName: channel}
	if err := ch.Delete(db.Context, db.DB); nil != err {
		return "Unable to leave channel", false, nil
	}

	return "Bye bye 👋", true, PartError{}
}

func RestartResponse(db *data.Database, channel, sender, text string) (string, bool, error) {
	var username string
	var valid = env.Username(&username)
	if !valid || sender != username {
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

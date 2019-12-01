package env

import (
	"log"
	"os"
	"strconv"
)

const twitchUsername = "TWITCH_USERNAME"
const twitchOauth = "TWITCH_OAUTH_TOKEN"
const activeInterval = "ACTIVE_INTERVAL"
const listenAddress = "LISTEN_ADDRESS"
const postgresConnectionString = "POSTGRES_CONNECTION_STRING"

type EmptyValueError struct {
	key string
}

func (e *EmptyValueError) Error() string {
	return "No environmental variable set for key " + e.key
}

func Username(ref *string) bool                 { return env(twitchUsername, ref) }
func OauthToken(ref *string) bool               { return env(twitchOauth, ref) }
func ListenAddress(ref *string) bool            { return env(listenAddress, ref) }
func PostgresConnectionString(ref *string) bool { return env(postgresConnectionString, ref) }

func ActiveInterval(ref *int64) bool {
	var str string
	if !env(activeInterval, &str) {
		return false
	}

	value, err := strconv.ParseInt(str, 10, 64)
	if nil != err {
		log.Println("unable to parse twitch active interval:", err)
		return false
	}

	*ref = value
	return true
}

func env(key string, ref *string) bool {
	value, valid := os.LookupEnv(key)
	if "" == value || !valid {
		log.Println(&EmptyValueError{key})
		return false
	}
	*ref = value
	return true
}

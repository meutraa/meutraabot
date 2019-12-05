package main

//go:generate sqlboiler --wipe -o pkg/models psql

import (
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func main() {
	// Read our environment variables, end if failure
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	connString := os.Getenv("POSTGRES_CONNECTION_STRING")
	if "" == listenAddress || "" == connString {
		log.Println("Unable to read environment variable")
		return
	}

	t, err := template.New("leaderboard").Parse(templateString)
	if nil != err {
		log.Fatalln("Unable to parse template string")
	}

	ct, err := template.New("chat").Parse(chatTemplateString)
	if nil != err {
		log.Fatalln("Unable to parse template string")
	}

	db, err := data.Connection(connString, 0)
	if nil != err {
		log.Fatalln("Unable to open db connection:", err)
	}
	defer db.Close()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/leaderboard/:user", func(c *gin.Context) { handleLeaderboardRequest(c, db, t) })
	r.GET("/chat/:user/:fontSize", func(c *gin.Context) { handleChatRequest(c, db, ct) })
	r.Run(listenAddress)
}

package main

import (
	"html/template"
	"log"

	"github.com/gin-gonic/gin"

	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/env"
)

func main() {
	// Read our environment variables, end if failure
	var connectionString, listenAddress string
	if !env.ListenAddress(&listenAddress) ||
		!env.PostgresConnectionString(&connectionString) {
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

	db, err := data.Connection(connectionString, 0)
	if nil != err {
		log.Fatalln("Unable to open db connection:", err)
	}
	defer db.Close()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/leaderboard/:user", func(c *gin.Context) { handleLeaderboardRequest(c, db, t) })
	r.GET("/chat/:user", func(c *gin.Context) { handleChatRequest(c, db, ct) })
	r.Run(listenAddress)
}

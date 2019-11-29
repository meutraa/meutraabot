package main

import (
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"gitlab.com/meutraa/meutraabot/pkg/data"
)

func main() {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if "" == listenAddress {
		log.Fatalln("Unable to read LISTEN_ADDRESS from env")
	}

	t, err := template.New("leaderboard").Parse(templateString)
	if nil != err {
		log.Fatalln("Unable to parse template string")
	}

	db, err := data.Connection()
	if nil != err {
		log.Fatalln("Unable to open db connection:", err)
	}
	defer db.Close()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/leaderboard/:user", func(c *gin.Context) { handleLeaderboardRequest(c, db, t) })
	r.Run(listenAddress)
}

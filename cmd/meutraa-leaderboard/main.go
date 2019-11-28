package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"gitlab.com/meutraa/meutraabot/pkg/data"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const templateString = `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8" http-equiv="refresh" content="10"></head>
<style>
td {
color: rgb(255, 255, 255);
font-family: Arial, Helvetica, sans-serif;
font-size: 7.5vw;
padding: 16px;
}
</style>
<body>
<table style="width:100%">
{{range .Users}}
<tr>
<td>{{ .Name }}</td>
<td>💩 {{ .Poops }}</td>
</tr>
{{end}}
</table>
</body>
</html>
`

type UserStat struct {
	Name  template.HTML
	Poops string
}

func getUserString(i int, name string) string {
	if name == "casweets" {
		return "🦊 <font color=\"ff8ba7\">" + name + "</font>"
	} else if name == "shannaboo1" {
		return "👑 <font color=\"ffcccc\">" + name + "</font>"
	} else if i == 0 {
		return "🏆 " + name
	} else if i == 1 {
		return "🥈 " + name
	} else if i == 2 {
		return "🥉 " + name
	}
	return " " + name
}

func handleLeaderboardRequest(c *gin.Context, db *data.Database, t *template.Template) {
	channel := "#" + c.Param("user")
	users, err := db.UsersWithTopWatchTime(channel)
	if nil != err {
		log.Println(err.Error())
		c.String(http.StatusBadRequest, "dummy")
		return
	}

	p := message.NewPrinter(language.English)
	var userstats []UserStat
	for i, metric := range users {
		userstats = append(userstats,
			UserStat{
				template.HTML(getUserString(i, metric.Sender)),
				p.Sprintf("%d", metric.WatchTime/60),
			})
	}

	data := struct {
		Users []UserStat
	}{
		Users: userstats,
	}

	var out bytes.Buffer
	if err := t.Execute(&out, data); nil != err {
		log.Println(err.Error())
		c.String(http.StatusInternalServerError, "no")
		return
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, out.String())
}

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

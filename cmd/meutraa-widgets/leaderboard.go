package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const templateString = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Leaderboard</title>
    <meta http-equiv="refresh" content="10">
    <style>
      td {
        color: rgb(255, 255, 255);
        font-family: Arial, Helvetica, sans-serif;
        font-size: 7.5vw;
        padding: 16px;
      }
      table {
        width: 100%;
      }
    </style>
  </head>
  <body>
    <table>{{range .Users}}
      <tr>
        <td>{{ .Name }}</td>
        <td>💩 {{ .Poops }}</td>
      </tr>{{end}}
    </table>
  </body>
</html>
`

type UserStat struct {
	Name  template.HTML
	Poops string
}

func handleLeaderboardRequest(c *gin.Context, db *data.Database, t *template.Template) {
	channel := "#" + c.Param("user")
	users, err := db.UsersWithTopWatchTime(channel, 8)
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
				template.HTML(getUserString(i, true, &metric)),
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

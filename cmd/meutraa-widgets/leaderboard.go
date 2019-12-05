package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	. "github.com/volatiletech/sqlboiler/queries/qm"
	"gitlab.com/meutraa/meutraabot/pkg/data"
	"gitlab.com/meutraa/meutraabot/pkg/models"
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
        font-family: Noto Sans, Helvetica, sans-serif;
        font-size: 7.5vw;
        padding: 16px;
      }
			.name {
        font-weight: bold;
			}
      table {
        width: 100%;
      }
    </style>
  </head>
  <body>
    <table>{{range .Users}}
      <tr>
        <td class="name" style="color:{{ .TextColor }}">{{ .Name }}</td>
        <td>💩 {{ .Poops }}</td>
      </tr>{{end}}
    </table>
  </body>
</html>
`

type UserStat struct {
	Name      string
	Poops     string
	TextColor string
}

func handleLeaderboardRequest(c *gin.Context, db *data.Database, t *template.Template) {
	channel := "#" + c.Param("user")

	users, err := models.Users(
		models.UserWhere.ChannelName.EQ(channel),
		OrderBy(models.UserColumns.WatchTime+" desc"),
		Limit(8),
	).All(db.Context, db.DB)
	if nil != err {
		log.Println(err.Error())
		c.String(http.StatusBadRequest, "dummy")
		return
	}

	p := message.NewPrinter(language.English)
	var userstats []UserStat
	for i, metric := range users {
		color := "#ffffff"
		if "" != metric.TextColor.String {
			color = metric.TextColor.String
		}
		userstats = append(userstats,
			UserStat{
				getUserString(i, true, metric.Sender, metric.Emoji.String),
				p.Sprintf("%d", metric.WatchTime/60),
				color,
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

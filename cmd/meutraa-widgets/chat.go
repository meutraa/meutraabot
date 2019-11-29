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

const chatTemplateString = `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8" http-equiv="refresh" content="5"></head>
<style>
td {
color: rgb(255, 255, 255);
font-family: Arial, Helvetica, sans-serif;
font-size: 4vw;
padding: 8px;
}
</style>
<body>
<table style="width:100%">
{{range .Users}}
<tr>
<td>{{ .Name }}</td>
<td>{{ .Message }}</td>
</tr>
{{end}}
</table>
</body>
</html>
`

type ChatMessage struct {
	Name    template.HTML
	Message string
}

func getUserString(i int, user *data.UserMetric) string {
	str := ""
	if "" != user.Emoji {
		str += user.Emoji + " "
	} else {
		if i == 0 {
			str += "🏆 "
		} else if i == 1 {
			str += "🥈 "
		} else if i == 2 {
			str += "🥉 "
		}
	}

	if "" != user.TextColor {
		str += "<font color=\"" + user.TextColor + "\">" + user.Sender + "</font>"
	} else {
		str += user.Sender
	}
	return str
}

func handleChatRequest(c *gin.Context, db *data.Database, t *template.Template) {
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
				template.HTML(getUserString(i, &metric)),
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

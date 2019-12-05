package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/meutraa/meutraabot/pkg/data"
)

const chatTemplateString = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Chat</title>
    <meta http-equiv="refresh" content="5">
    <style>
      .name {
        font-weight: bold;
      }
      span {
        font-size: 22px;
        color: rgb(255, 255, 255);
        font-family: Noto Sans, Helvetica, sans-serif;
      }
      html {
        overflow: hidden;
      }
      div {
        border-radius: 1em;
        margin: 4px;
        padding: 6px 16px;
        display: inline-block;
        background-color: rgba(0, 0, 0, 0.2);
      }
    </style>
  </head>
  <body>{{range .Messages}}
		<div style="opacity: {{ .Opacity }};">
      <span class="name" style="color:{{ .TextColor }}">{{ .Name }}</span><br>
      <span>{{ .Message }}</span>
    </div>{{end}}
  </body>
</html>
`

type ChatMessage struct {
	Name      string
	Opacity   float64
	Message   string
	TextColor string
}

func getUserString(i int, hasDef bool, sender, emoji string) string {
	str := ""
	if "" != emoji {
		str += emoji + " "
	} else if hasDef {
		if i == 0 {
			str += "🏆 "
		} else if i == 1 {
			str += "🥈 "
		} else if i == 2 {
			str += "🥉 "
		}
	} else {
		str += "  "
	}
	return str + sender
}

func handleChatRequest(c *gin.Context, db *data.Database, t *template.Template) {
	channel := "#" + c.Param("user")
	fontSize := c.Param("fontSize")

	rows, err := db.DB.QueryContext(db.Context,
		"SELECT messages.message, messages.sender, users.emoji, users.text_color "+
			"FROM messages "+
			"INNER JOIN users ON "+
			"messages.sender = users.sender AND "+
			"messages.channel_name = users.channel_name "+
			"WHERE messages.channel_name = $1 "+
			"ORDER BY messages.created_at DESC "+
			"LIMIT 8", channel,
	)

	if nil != err {
		log.Println(err.Error())
		c.String(http.StatusBadRequest, "dummy")
		return
	}

	var chatMessages []ChatMessage
	for rows.Next() {
		var message, sender string
		var emoji, textColor sql.NullString
		if err := rows.Scan(&message, &sender, &emoji, &textColor); nil != err {
			log.Println("Unable to scan message row", err)
			continue
		}

		color := "#ffffff"
		if "" != textColor.String {
			color = textColor.String
		}
		chatMessages = append(chatMessages,
			ChatMessage{
				getUserString(0, false, sender, emoji.String),
				0.0,
				message,
				color,
			})
	}

	count := len(chatMessages)
	for i := 0; i < count; i++ {
		opacity := 1.0
		if i > 4 {
			opacity = (1.0 - float64(i-4)/float64(count-4))
		}
		chatMessages[i].Opacity = opacity
	}

	data := struct {
		Messages []ChatMessage
		FontSize string
	}{
		Messages: chatMessages,
		FontSize: fontSize,
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

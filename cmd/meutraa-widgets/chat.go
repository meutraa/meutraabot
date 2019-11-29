package main

import (
	"bytes"
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
      td, th {
        color: rgb(255, 255, 255);
        font-family: Arial, Helvetica, sans-serif;
        font-size: 4vw;
        vertical-align: top;
        border: none;
      }
      td {
        padding: 0;
      }
      th {
        white-space: nowrap;
        padding: 6px 24px;
        border-radius: 1em 0em 0em 1em;
        background-color: rgba(0, 0, 0, 0.2);
      }
      .msg {
        background-color: rgba(0, 0, 0, 0.2);
        border-radius: 0em 1em 1em 0em;
        padding: 6px 24px;
        display: inline-block
      }
      html {
        overflow: hidden;
      }
      table {
        border-collapse: seperate;
        border-style: hidden;
        border-spacing: 0 8px;
        width: 100%;
        position: relative;
        bottom: 0;
      }
    </style>
  </head>
  <body>
    <table>{{range .Messages}}
      <tr style="opacity: {{ .Opacity }};">
        <th>{{ .Name }}</th>
        <td><div class="msg">{{ .Message }}</div></td>
      </tr>{{end}}
    </table>
    <script type='text/javascript'>
      window.scrollTo(0, document.body.scrollHeight);
    </script>
  </body>
</html>
`

type ChatMessage struct {
	Name    template.HTML
	Opacity float64
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
	messages, err := db.Messages(channel, 20)
	if nil != err {
		log.Println(err.Error())
		c.String(http.StatusBadRequest, "dummy")
		return
	}

	var chatMessages []ChatMessage
	count := float64(len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		opacity := 1.0
		if i > 5 {
			opacity = (1.0 - float64(i-5)/(count-5))
		}
		chatMessages = append(chatMessages,
			ChatMessage{
				template.HTML(getUserString(i, &(message.User))),
				opacity,
				message.Message,
			})
	}

	data := struct {
		Messages []ChatMessage
	}{
		Messages: chatMessages,
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

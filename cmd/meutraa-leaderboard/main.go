package main

import (
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type UserMetric struct {
	Sender       string `gorm:"primary_key"`
	ChannelName  string `gorm:"primary_key"`
	WordCount    int64
	MessageCount int64
	WatchTime    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func readEnv(key string) string {
	value := os.Getenv(key)
	if "" == value {
		log.Fatalln("Unable to read", key, "from environment")
	}
	return value
}

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

func getLeaderboard(db *gorm.DB, channel string) ([]UserMetric, error) {
	var users []UserMetric
	if err := db.Where("channel_name = ?", channel).
		Order("watch_time desc").
		Order("sender asc").
		Limit(8).
		Find(&users).Error; nil != err {
		return nil, errors.New("Unable to get top users for channel " + channel + ": " + err.Error())
	}
	return users, nil
}

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

func handleLeaderboardRequest(c *gin.Context, db *gorm.DB, t *template.Template) {
	channel := "#" + c.Param("user")
	users, err := getLeaderboard(db, channel)
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
	connectionString := readEnv("POSTGRES_CONNECTION_STRING")
	listenAddress := readEnv("LISTEN_ADDRESS")

	t, err := template.New("leaderboard").Parse(templateString)
	if nil != err {
		log.Fatalln("Unable to parse template string")
	}

	db, err := gorm.Open("postgres", connectionString)
	if nil != err {
		log.Fatalln("Unable to establish connection to database", err.Error())
	}
	defer db.Close()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/leaderboard/:user", func(c *gin.Context) { handleLeaderboardRequest(c, db, t) })
	r.Run(listenAddress)
}

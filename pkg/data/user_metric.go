package data

import "time"

type UserMetric struct {
	Sender       string `gorm:"primary_key"`
	ChannelName  string `gorm:"primary_key"`
	Emoji        string
	TextColor    string
	WordCount    int64
	MessageCount int64
	WatchTime    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

package data

import "github.com/jinzhu/gorm"

type Message struct {
	gorm.Model
	Sender      string
	ChannelName string
	Message     string
	User        UserMetric `gorm:"foreignkey:Sender,ChannelName;association_foreignkey:Sender,ChannelName"`
}

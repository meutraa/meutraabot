package data

import "github.com/jinzhu/gorm"

type Channel struct {
	gorm.Model
	ChannelName string
	HiccupCount int64
}

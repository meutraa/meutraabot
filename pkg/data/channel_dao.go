package data

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func (d *Database) ChannelCount() (count int, err error) {
	err = d.orm.Model(&Channel{}).Count(&count).Error
	if nil != err {
		err = errors.Wrap(err, "unable to get channel count")
		return
	}
	return
}

func (d *Database) Channels() ([]Channel, error) {
	var channels []Channel
	if err := d.orm.Find(&channels).Error; nil != err {
		return nil, errors.Wrap(err, "unable to get list of channels")
	}
	return channels, nil
}

func (d *Database) AddToIntChannel(channelName, field string, value int64) error {
	return d.addToInt(&Channel{}, func(query *gorm.DB) *gorm.DB {
		return query.Where("channel_name = ?", channelName)
	}, channelName, "n/a", field, value)
}

func (d *Database) GetIntChannel(channelName, field string) int64 {
	return d.getInt(&Channel{}, func(query *gorm.DB) *gorm.DB {
		return query.Where("channel_name = ?", channelName)
	}, channelName, "n/a", field)
}


package data

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func (d *Database) Channels() ([]Channel, error) {
	var channels []Channel
	if err := d.orm.Find(&channels).Error; nil != err {
		return nil, errors.Wrap(err, "unable to get list of channels")
	}
	return channels, nil
}

func (d *Database) DeleteChannel(channelName string) error {
	return d.orm.Model(&Channel{}).
		Where("channel_name = ?", channelName).
		Delete(&Channel{}).Error
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

package data

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func (d *Database) UsersWithTopWatchTime(channel string) ([]UserMetric, error) {
	var users []UserMetric
	if err := d.orm.Where("channel_name = ?", channel).
		Order("watch_time desc").
		Order("sender asc").
		Limit(8).
		Find(&users).Error; nil != err {
		return nil, errors.Wrap(err, "unable to get top users for channel "+channel)
	}
	return users, nil
}

func (d *Database) AddWatchTime(channel, sender string) {
	now := time.Now()
	err := d.orm.Model(&UserMetric{}).
		Where("channel_name = ? AND sender = ? AND (? - updated_at) < ?", channel, sender, now, d.activeInterval).
		Update("watch_time", gorm.Expr("watch_time + cast(extract(epoch from (? - updated_at)) AS bigint)", now)).Error
	if nil != err {
		log.Println("Unable to update watch time for user", sender, "in channel", channel, ":", err)
	}
}

func (d *Database) GetIntUserMetric(channelName, name, field string) int64 {
	return d.getInt(&UserMetric{}, func(query *gorm.DB) *gorm.DB {
		return query.Where("channel_name = ? AND sender = ?", channelName, name)
	}, channelName, name, field)
}

func (d *Database) AddToIntUserMetric(channelName, user, field string, value int64) error {
	// If this user does not have a row yet
	userMetric := UserMetric{ChannelName: channelName, Sender: user}
	if err := d.orm.Set("gorm:insert_option", "ON CONFLICT DO NOTHING").
		FirstOrCreate(&userMetric).Error; nil != err {
		return errors.New("Unable to create new UserMetric: " + err.Error())
	}

	return d.addToInt(&UserMetric{}, func(query *gorm.DB) *gorm.DB {
		return query.Where("channel_name = ? AND sender = ?", channelName, user)
	}, channelName, user, field, value)
}

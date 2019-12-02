package data

import (
	"log"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func (d *Database) UsersWithTopWatchTime(channel string, limit int) ([]UserMetric, error) {
	var users []UserMetric
	if err := d.orm.Where("channel_name = ?", channel).
		Order("watch_time desc").
		Order("sender asc").
		Limit(limit).
		Find(&users).Error; nil != err {
		return nil, errors.Wrap(err, "unable to get top users for channel "+channel)
	}
	return users, nil
}

func (d *Database) UpdateMetrics(channel, sender, text string) {
	wordCount := len(strings.Split(text, " "))
	now := time.Now()

	if err := d.orm.Model(&UserMetric{}).
		Where("channel_name = ? AND sender = ?", channel, sender).
		Updates(map[string]interface{}{
			"watch_time":    gorm.Expr("CASE WHEN ((? - updated_at) < ?) THEN watch_time + cast(extract(epoch from (? - updated_at)) AS bigint) ELSE watch_time END", now, d.activeInterval, now),
			"message_count": gorm.Expr("message_count + ?", 1),
			"word_count":    gorm.Expr("word_count + ?", wordCount),
		}).Error; nil != err {
		log.Println("Unable to update user metrics:", err)
	}
}

func (d *Database) SetEmoji(channelName, user, emoji string) error {
	return d.orm.Model(&UserMetric{}).
		Where("channel_name = ? AND sender = ?", channelName, user).
		Update("emoji", emoji).Error
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

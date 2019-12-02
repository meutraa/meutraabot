package data

import "github.com/pkg/errors"

func (d *Database) Messages(channel string, limit int) ([]Message, error) {
	var messages []Message
	if err := d.orm.
		Preload("User").
		Where("channel_name = ?", channel).
		Order("created_at desc").
		Limit(limit).
		Find(&messages).Error; nil != err {
		return nil, errors.Wrap(err, "unable to get messages for channel "+channel)
	}
	return messages, nil
}

func (d *Database) AddMessage(channel, sender, text string) error {
	var user UserMetric
	d.orm.FirstOrCreate(&user, &UserMetric{
		Sender:      sender,
		ChannelName: channel,
	})
	if err := d.orm.
		Create(&Message{
			Sender:      sender,
			ChannelName: channel,
			Message:     text,
		}).Error; nil != err {
		return errors.Wrap(err, "unable to save message for user"+sender)
	}
	return nil
}

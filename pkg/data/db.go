package data

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type Database struct {
	orm            *gorm.DB
	activeInterval int64
}

func readEnv(key string) string {
	value := os.Getenv(key)
	if "" == value {
		log.Fatalln("Unable to read", key, "from environment")
	}
	return value
}

func Connection() (*Database, error) {
	activeIntervalStr := readEnv("ACTIVE_INTERVAL")
	activeInterval, err := strconv.ParseInt(activeIntervalStr, 10, 64)
	if nil != err {
		return nil, errors.New("Unable to parse twitch active interval")
	}

	connectionString := readEnv("POSTGRES_CONNECTION_STRING")
	orm, err := gorm.Open("postgres", connectionString)
	if nil != err {
		return nil, errors.Wrap(err, "unable to establish connection to database")
	}

	orm.AutoMigrate(&Channel{}, &UserMetric{})
	return &Database{
		orm:            orm,
		activeInterval: activeInterval,
	}, nil
}

func (d *Database) Close() error {
	if nil != d.orm {
		return d.orm.Close()
	}
	return nil
}

func (d *Database) Populate() error {
	log.Println("Requested population of database")
	tx := d.orm.Begin()
	if err := tx.Error; nil != err {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			log.Println("Unable to populate database, rolling back")
			tx.Rollback()
		}
	}()

	if err := tx.Create(&Channel{ChannelName: "#vulpesmusketeer", HiccupCount: 643}).Error; nil != err {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&Channel{ChannelName: "#meutraa"}).Error; nil != err {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

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

func (d *Database) addToInt(model interface{}, query func(*gorm.DB) *gorm.DB, channelName, user, field string, value int64) error {
	return query(d.orm.Model(model)).
		Update(field, gorm.Expr(field+" + ?", value)).Error
}

func (d *Database) GetIntChannel(channelName, field string) int64 {
	return d.getInt(&Channel{}, func(query *gorm.DB) *gorm.DB {
		return query.Where("channel_name = ?", channelName)
	}, channelName, "n/a", field)
}

func (d *Database) GetIntUserMetric(channelName, name, field string) int64 {
	return d.getInt(&UserMetric{}, func(query *gorm.DB) *gorm.DB {
		return query.Where("channel_name = ? AND sender = ?", channelName, name)
	}, channelName, name, field)
}

func (d *Database) getInt(model interface{}, query func(*gorm.DB) *gorm.DB, channelName, user, field string) int64 {
	var values []int64
	err := query(d.orm.Model(model)).
		Pluck(field, &values).Error
	if nil != err {
		log.Println("Unable to pluck", field, "for channel", channelName, "user", user, ":", err)
	}
	if len(values) > 0 {
		return values[0]
	}
	return 0
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

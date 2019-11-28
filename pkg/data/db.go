package data

import (
	"log"
	"os"
	"strconv"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
)

type Database struct {
	orm            *gorm.DB
	activeInterval int64
}

func Connection() (*Database, error) {
	activeIntervalStr := os.Getenv("ACTIVE_INTERVAL")
	if "" == activeIntervalStr {
		return nil, errors.New("Unable to read ACTIVE_INTERVAL from env")
	}

	activeInterval, err := strconv.ParseInt(activeIntervalStr, 10, 64)
	if nil != err {
		return nil, errors.New("Unable to parse twitch active interval")
	}

	connectionString := os.Getenv("POSTGRES_CONNECTION_STRING")
	if "" == activeIntervalStr {
		return nil, errors.New("Unable to read POSTGRES_CONNECTION_STRING from env")
	}

	orm, err := gorm.Open("postgres", connectionString)
	orm.LogMode(true)
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

func (d *Database) addToInt(model interface{}, query func(*gorm.DB) *gorm.DB, channelName, user, field string, value int64) error {
	return query(d.orm.Model(model)).
		Update(field, gorm.Expr(field+" + ?", value)).Error
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

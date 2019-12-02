package data

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type Database struct {
	DB             *sql.DB
	Context        context.Context
	ActiveInterval int64
}

func Connection(connectionString string, activeInterval int64) (*Database, error) {
	db, err := sql.Open("postgres", connectionString)
	// orm.LogMode(true)
	if nil != err {
		return nil, errors.Wrap(err, "unable to establish connection to database")
	}

	return &Database{
		DB:             db,
		Context:        context.Background(),
		ActiveInterval: activeInterval,
	}, nil
}

func (d *Database) Close() error {
	if nil != d.DB {
		return d.DB.Close()
	}
	return nil
}

/*
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
*/

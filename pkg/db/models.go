// Code generated by sqlc. DO NOT EDIT.

package db

import (
	"database/sql"
	"time"
)

type Approval struct {
	ChannelID string
	UserID    string
}

type Channel struct {
	ChannelID string
	CreatedAt time.Time
	UpdatedAt sql.NullTime
}

type Command struct {
	ChannelID string
	Name      string
	Template  string
}

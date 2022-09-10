// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0

package db

import (
	"database/sql"
	"time"
)

type Approval struct {
	ChannelID string
	Manual    bool
	UserID    string
}

type Channel struct {
	ChannelID          string
	AutoreplyEnabled   bool
	AutoreplyFrequency float64
	ReplySafety        int64
	OpenaiToken        sql.NullString
	CreatedAt          time.Time
	UpdatedAt          sql.NullTime
}

type Command struct {
	ChannelID string
	Name      string
	Template  string
}

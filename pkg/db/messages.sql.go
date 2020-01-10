// Code generated by sqlc. DO NOT EDIT.
// source: messages.sql

package mdb

import (
	"context"
)

const createMessage = `-- name: CreateMessage :exec
INSERT INTO messages
  (channel_name, sender, created_at, message)
  VALUES ($1, $2, NOW(), $3)
  ON CONFLICT DO NOTHING
`

type CreateMessageParams struct {
	ChannelName string
	Sender      string
	Message     string
}

func (q *Queries) CreateMessage(ctx context.Context, arg CreateMessageParams) error {
	_, err := q.db.ExecContext(ctx, createMessage, arg.ChannelName, arg.Sender, arg.Message)
	return err
}

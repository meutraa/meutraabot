// Code generated by sqlc. DO NOT EDIT.
// source: channels.sql

package db

import (
	"context"
)

const createChannel = `-- name: CreateChannel :exec
INSERT INTO channels (channel_name, created_at)
  VALUES ($1, NOW())
  ON CONFLICT DO NOTHING
`

func (q *Queries) CreateChannel(ctx context.Context, channelName string) error {
	_, err := q.db.ExecContext(ctx, createChannel, channelName)
	return err
}

const deleteChannel = `-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_name = $1
`

func (q *Queries) DeleteChannel(ctx context.Context, channelName string) error {
	_, err := q.db.ExecContext(ctx, deleteChannel, channelName)
	return err
}

const getChannelNames = `-- name: GetChannelNames :many
SELECT channel_name FROM channels
`

func (q *Queries) GetChannelNames(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getChannelNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var channel_name string
		if err := rows.Scan(&channel_name); err != nil {
			return nil, err
		}
		items = append(items, channel_name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

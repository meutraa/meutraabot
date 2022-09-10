// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: channels.sql

package db

import (
	"context"
	"database/sql"
)

const createChannel = `-- name: CreateChannel :exec
INSERT INTO channels (channel_id, created_at)
  VALUES (?, now())
  ON CONFLICT DO NOTHING
`

func (q *Queries) CreateChannel(ctx context.Context, channelID string) error {
	_, err := q.exec(ctx, q.createChannelStmt, createChannel, channelID)
	return err
}

const deleteChannel = `-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_id = ?
`

func (q *Queries) DeleteChannel(ctx context.Context, channelID string) error {
	_, err := q.exec(ctx, q.deleteChannelStmt, deleteChannel, channelID)
	return err
}

const getChannel = `-- name: GetChannel :one
SELECT channel_id, autoreply_enabled, autoreply_frequency, reply_safety, openai_token, created_at, updated_at FROM channels WHERE channel_id = ? ORDER BY created_at DESC
`

func (q *Queries) GetChannel(ctx context.Context, channelID string) (Channel, error) {
	row := q.queryRow(ctx, q.getChannelStmt, getChannel, channelID)
	var i Channel
	err := row.Scan(
		&i.ChannelID,
		&i.AutoreplyEnabled,
		&i.AutoreplyFrequency,
		&i.ReplySafety,
		&i.OpenaiToken,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getChannels = `-- name: GetChannels :many
SELECT channel_id FROM channels
`

func (q *Queries) GetChannels(ctx context.Context) ([]string, error) {
	rows, err := q.query(ctx, q.getChannelsStmt, getChannels)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var channel_id string
		if err := rows.Scan(&channel_id); err != nil {
			return nil, err
		}
		items = append(items, channel_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateChannel = `-- name: UpdateChannel :exec
UPDATE channels
 SET autoreply_enabled = ?,
  autoreply_frequency = ?,
  reply_safety = ?,
  updated_at = now()
 WHERE channel_id = ?
`

type UpdateChannelParams struct {
	AutoreplyEnabled   bool
	AutoreplyFrequency float64
	ReplySafety        int64
	ChannelID          string
}

func (q *Queries) UpdateChannel(ctx context.Context, arg UpdateChannelParams) error {
	_, err := q.exec(ctx, q.updateChannelStmt, updateChannel,
		arg.AutoreplyEnabled,
		arg.AutoreplyFrequency,
		arg.ReplySafety,
		arg.ChannelID,
	)
	return err
}

const updateChannelToken = `-- name: UpdateChannelToken :exec
UPDATE channels
 SET openai_token = ?,
  updated_at = now()
 WHERE channel_id = ?
`

type UpdateChannelTokenParams struct {
	OpenaiToken sql.NullString
	ChannelID   string
}

func (q *Queries) UpdateChannelToken(ctx context.Context, arg UpdateChannelTokenParams) error {
	_, err := q.exec(ctx, q.updateChannelTokenStmt, updateChannelToken, arg.OpenaiToken, arg.ChannelID)
	return err
}

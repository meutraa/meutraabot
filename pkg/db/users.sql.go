// Code generated by sqlc. DO NOT EDIT.
// source: users.sql

package db

import (
	"context"
	"database/sql"
)

const createUser = `-- name: CreateUser :exec
INSERT INTO users
  (channel_name, sender, created_at, message_count, word_count, watch_time, text_color)
  VALUES ($1, $2, NOW(), 0, 0, 0, $3)
  ON CONFLICT DO NOTHING
`

type CreateUserParams struct {
	ChannelName string
	Sender      string
	TextColor   sql.NullString
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) error {
	_, err := q.db.ExecContext(ctx, createUser, arg.ChannelName, arg.Sender, arg.TextColor)
	return err
}

const getMetrics = `-- name: GetMetrics :one
SELECT
  watch_time,
  message_count,
  word_count
  FROM users
  WHERE channel_name = $1
  AND sender = $2
`

type GetMetricsParams struct {
	ChannelName string
	Sender      string
}

type GetMetricsRow struct {
	WatchTime    int64
	MessageCount int64
	WordCount    int64
}

func (q *Queries) GetMetrics(ctx context.Context, arg GetMetricsParams) (GetMetricsRow, error) {
	row := q.db.QueryRowContext(ctx, getMetrics, arg.ChannelName, arg.Sender)
	var i GetMetricsRow
	err := row.Scan(&i.WatchTime, &i.MessageCount, &i.WordCount)
	return i, err
}

const getTopWatchers = `-- name: GetTopWatchers :many
SELECT
  sender
  FROM users
  WHERE channel_name = $1
  ORDER BY watch_time DESC
  LIMIT $2
`

type GetTopWatchersParams struct {
	ChannelName string
	Limit       int32
}

func (q *Queries) GetTopWatchers(ctx context.Context, arg GetTopWatchersParams) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getTopWatchers, arg.ChannelName, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var sender string
		if err := rows.Scan(&sender); err != nil {
			return nil, err
		}
		items = append(items, sender)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getWatchTimeRank = `-- name: GetWatchTimeRank :one
SELECT
  cast(rank AS INTEGER)
  FROM
    (SELECT
      RANK() OVER (ORDER BY watch_time DESC) AS rank,
      sender
      FROM users
      WHERE channel_name = $1
  ) AS ss
  WHERE sender = $2
`

type GetWatchTimeRankParams struct {
	ChannelName string
	Sender      string
}

func (q *Queries) GetWatchTimeRank(ctx context.Context, arg GetWatchTimeRankParams) (int32, error) {
	row := q.db.QueryRowContext(ctx, getWatchTimeRank, arg.ChannelName, arg.Sender)
	var rank int32
	err := row.Scan(&rank)
	return rank, err
}

const updateEmoji = `-- name: UpdateEmoji :exec
UPDATE users
  SET emoji = $3
  WHERE channel_name = $1
  AND sender = $2
`

type UpdateEmojiParams struct {
	ChannelName string
	Sender      string
	Emoji       sql.NullString
}

func (q *Queries) UpdateEmoji(ctx context.Context, arg UpdateEmojiParams) error {
	_, err := q.db.ExecContext(ctx, updateEmoji, arg.ChannelName, arg.Sender, arg.Emoji)
	return err
}

const updateMetrics = `-- name: UpdateMetrics :exec
UPDATE users
  SET
    message_count = message_count + 1,
    word_count = word_count + $3,
    watch_time = CASE
      WHEN NOW() - updated_at < interval '15 minutes'
        THEN watch_time + extract(epoch from (NOW() - updated_at))
      ELSE watch_time
    END,
    updated_at = NOW()
  WHERE channel_name = $1
  AND sender = $2
`

type UpdateMetricsParams struct {
	ChannelName string
	Sender      string
	WordCount   int64
}

func (q *Queries) UpdateMetrics(ctx context.Context, arg UpdateMetricsParams) error {
	_, err := q.db.ExecContext(ctx, updateMetrics, arg.ChannelName, arg.Sender, arg.WordCount)
	return err
}

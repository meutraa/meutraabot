// Code generated by sqlc. DO NOT EDIT.
// source: users.sql

package db

import (
	"context"
	"time"
)

const createUser = `-- name: CreateUser :exec
INSERT INTO users
  (channel_name, sender, created_at, message_count, word_count, watch_time)
  VALUES ($1, $2, NOW(), 0, 0, 0)
  ON CONFLICT DO NOTHING
`

type CreateUserParams struct {
	ChannelName string
	Sender      string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) error {
	_, err := q.exec(ctx, q.createUserStmt, createUser, arg.ChannelName, arg.Sender)
	return err
}

const getMetrics = `-- name: GetMetrics :one
SELECT
  watch_time,
  message_count,
  word_count,
  extract(epoch from (NOW() - created_at)) as age,
  (watch_time/60) + (word_count / 8) as points,
  created_at
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
	Age          float64
	Points       int32
	CreatedAt    time.Time
}

func (q *Queries) GetMetrics(ctx context.Context, arg GetMetricsParams) (GetMetricsRow, error) {
	row := q.queryRow(ctx, q.getMetricsStmt, getMetrics, arg.ChannelName, arg.Sender)
	var i GetMetricsRow
	err := row.Scan(
		&i.WatchTime,
		&i.MessageCount,
		&i.WordCount,
		&i.Age,
		&i.Points,
		&i.CreatedAt,
	)
	return i, err
}

const getTopWatchers = `-- name: GetTopWatchers :many
SELECT
  sender
  FROM users
  WHERE channel_name = $1
  ORDER BY ((watch_time/60) + (word_count / 8)) DESC
  LIMIT $2
`

type GetTopWatchersParams struct {
	ChannelName string
	Limit       int32
}

func (q *Queries) GetTopWatchers(ctx context.Context, arg GetTopWatchersParams) ([]string, error) {
	rows, err := q.query(ctx, q.getTopWatchersStmt, getTopWatchers, arg.ChannelName, arg.Limit)
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

const getTopWatchersAverage = `-- name: GetTopWatchersAverage :many
SELECT
  sender
  FROM users
  WHERE channel_name = $1
  ORDER BY (((watch_time/60) + (word_count / 8)) / extract(epoch from (NOW() - created_at))) DESC
  LIMIT $2
`

type GetTopWatchersAverageParams struct {
	ChannelName string
	Limit       int32
}

func (q *Queries) GetTopWatchersAverage(ctx context.Context, arg GetTopWatchersAverageParams) ([]string, error) {
	rows, err := q.query(ctx, q.getTopWatchersAverageStmt, getTopWatchersAverage, arg.ChannelName, arg.Limit)
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
	row := q.queryRow(ctx, q.getWatchTimeRankStmt, getWatchTimeRank, arg.ChannelName, arg.Sender)
	var rank int32
	err := row.Scan(&rank)
	return rank, err
}

const getWatchTimeRankAverage = `-- name: GetWatchTimeRankAverage :one
SELECT
  cast(rank AS INTEGER)
  FROM
    (SELECT
      RANK() OVER (ORDER BY (watch_time / extract(epoch from (NOW() - created_at))) DESC) AS rank,
      sender
      FROM users
      WHERE channel_name = $1
  ) AS ss
  WHERE sender = $2
`

type GetWatchTimeRankAverageParams struct {
	ChannelName string
	Sender      string
}

func (q *Queries) GetWatchTimeRankAverage(ctx context.Context, arg GetWatchTimeRankAverageParams) (int32, error) {
	row := q.queryRow(ctx, q.getWatchTimeRankAverageStmt, getWatchTimeRankAverage, arg.ChannelName, arg.Sender)
	var rank int32
	err := row.Scan(&rank)
	return rank, err
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
	_, err := q.exec(ctx, q.updateMetricsStmt, updateMetrics, arg.ChannelName, arg.Sender, arg.WordCount)
	return err
}

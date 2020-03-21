// Code generated by sqlc. DO NOT EDIT.
// source: counter.sql

package db

import (
	"context"
)

const getCounter = `-- name: GetCounter :one
SELECT value FROM counter
  WHERE channel_name = $1
  AND name = $2
`

type GetCounterParams struct {
	ChannelName string
	Name        string
}

func (q *Queries) GetCounter(ctx context.Context, arg GetCounterParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, getCounter, arg.ChannelName, arg.Name)
	var value int64
	err := row.Scan(&value)
	return value, err
}

const updateCounter = `-- name: UpdateCounter :exec
INSERT INTO counter (channel_name, name, value)
  VALUES ($1, $2, $3)
  ON CONFLICT
  ON CONSTRAINT counter_pkey DO UPDATE
  SET value = counter.value + $3
`

type UpdateCounterParams struct {
	ChannelName string
	Name        string
	Value       int64
}

func (q *Queries) UpdateCounter(ctx context.Context, arg UpdateCounterParams) error {
	_, err := q.db.ExecContext(ctx, updateCounter, arg.ChannelName, arg.Name, arg.Value)
	return err
}
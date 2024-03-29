// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: numbers.sql

package db

import (
	"context"
)

const addToNumber = `-- name: AddToNumber :exec
INSERT INTO numbers(channel_id, name, value)
VALUES(?, ?, ?)
ON CONFLICT(channel_id, name) 
DO UPDATE SET value = (value + ?)
`

type AddToNumberParams struct {
	ChannelID string
	Name      string
	Value     int64
}

func (q *Queries) AddToNumber(ctx context.Context, arg AddToNumberParams) error {
	_, err := q.exec(ctx, q.addToNumberStmt, addToNumber, arg.ChannelID, arg.Name, arg.Value, arg.Value)
	return err
}

const getNumber = `-- name: GetNumber :one
SELECT channel_id, name, value FROM numbers WHERE channel_id = ? AND name = ?
`

type GetNumberParams struct {
	ChannelID string
	Name      string
}

func (q *Queries) GetNumber(ctx context.Context, arg GetNumberParams) (Number, error) {
	row := q.queryRow(ctx, q.getNumberStmt, getNumber, arg.ChannelID, arg.Name)
	var i Number
	err := row.Scan(&i.ChannelID, &i.Name, &i.Value)
	return i, err
}

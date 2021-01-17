// Code generated by sqlc. DO NOT EDIT.
// source: approvals.sql

package db

import (
	"context"
)

const approve = `-- name: Approve :exec
INSERT INTO
  approvals (channel_name, username)
VALUES
  ($1, $2) ON CONFLICT DO NOTHING
`

type ApproveParams struct {
	ChannelName string
	Username    string
}

func (q *Queries) Approve(ctx context.Context, arg ApproveParams) error {
	_, err := q.exec(ctx, q.approveStmt, approve, arg.ChannelName, arg.Username)
	return err
}

const isApproved = `-- name: IsApproved :one
SELECT
  COUNT(*)
FROM
  approvals
WHERE
  channel_name = $1
  AND username = $2
`

type IsApprovedParams struct {
	ChannelName string
	Username    string
}

func (q *Queries) IsApproved(ctx context.Context, arg IsApprovedParams) (int64, error) {
	row := q.queryRow(ctx, q.isApprovedStmt, isApproved, arg.ChannelName, arg.Username)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const unapprove = `-- name: Unapprove :exec
DELETE FROM
  approvals
WHERE
  channel_name = $1
  AND username = $2
`

type UnapproveParams struct {
	ChannelName string
	Username    string
}

func (q *Queries) Unapprove(ctx context.Context, arg UnapproveParams) error {
	_, err := q.exec(ctx, q.unapproveStmt, unapprove, arg.ChannelName, arg.Username)
	return err
}

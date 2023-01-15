// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: commands.sql

package db

import (
	"context"
)

const deleteCommand = `-- name: DeleteCommand :exec
DELETE FROM commands
  WHERE channel_id = ?
  AND name = ?
`

type DeleteCommandParams struct {
	ChannelID string
	Name      string
}

func (q *Queries) DeleteCommand(ctx context.Context, arg DeleteCommandParams) error {
	_, err := q.exec(ctx, q.deleteCommandStmt, deleteCommand, arg.ChannelID, arg.Name)
	return err
}

const getCommand = `-- name: GetCommand :one
SELECT template
  FROM commands
  WHERE name = ?
  AND channel_id = ?
`

type GetCommandParams struct {
	Name      string
	ChannelID string
}

func (q *Queries) GetCommand(ctx context.Context, arg GetCommandParams) (string, error) {
	row := q.queryRow(ctx, q.getCommandStmt, getCommand, arg.Name, arg.ChannelID)
	var template string
	err := row.Scan(&template)
	return template, err
}

const getCommands = `-- name: GetCommands :many
SELECT name
  FROM commands
  WHERE channel_id = ?
  ORDER BY name ASC
`

func (q *Queries) GetCommands(ctx context.Context, channelID string) ([]string, error) {
	rows, err := q.query(ctx, q.getCommandsStmt, getCommands, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getCommandsByID = `-- name: GetCommandsByID :many
SELECT name, template
  FROM commands
  WHERE channel_id = ?
  ORDER BY name ASC
`

type GetCommandsByIDRow struct {
	Name     string
	Template string
}

func (q *Queries) GetCommandsByID(ctx context.Context, channelID string) ([]GetCommandsByIDRow, error) {
	rows, err := q.query(ctx, q.getCommandsByIDStmt, getCommandsByID, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCommandsByIDRow
	for rows.Next() {
		var i GetCommandsByIDRow
		if err := rows.Scan(&i.Name, &i.Template); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMatchingCommands = `-- name: GetMatchingCommands :many
SELECT
    template,
    name,
    channel_id
  FROM commands
  WHERE (
    channel_id = ?
    OR
    channel_id = '0'
  )
  AND regexp(name, ?)
`

type GetMatchingCommandsRow struct {
	Template  string
	Name      string
	ChannelID string
}

func (q *Queries) GetMatchingCommands(ctx context.Context, regexp ...interface{}) ([]GetMatchingCommandsRow, error) {
	rows, err := q.query(ctx, q.getMatchingCommandsStmt, getMatchingCommands, regexp...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetMatchingCommandsRow
	for rows.Next() {
		var i GetMatchingCommandsRow
		if err := rows.Scan(&i.Template, &i.Name, &i.ChannelID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const setCommand = `-- name: SetCommand :exec
INSERT INTO commands (channel_id, name, template)
  VALUES (?, ?, ?)
  ON CONFLICT(channel_id, name) DO UPDATE
  SET template = ?
`

type SetCommandParams struct {
	ChannelID string
	Name      string
	Template  string
}

func (q *Queries) SetCommand(ctx context.Context, arg SetCommandParams) error {
	_, err := q.exec(ctx, q.setCommandStmt, setCommand, arg.ChannelID, arg.Name, arg.Template, arg.Template)
	return err
}

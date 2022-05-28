// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.13.0
// source: commands.sql

package db

import (
	"context"
)

const deleteCommand = `-- name: DeleteCommand :exec
DELETE FROM commands
  WHERE channel_id = $1
  AND name = $2
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
  WHERE name = $2
  AND channel_id = $1
`

type GetCommandParams struct {
	ChannelID string
	Name      string
}

func (q *Queries) GetCommand(ctx context.Context, arg GetCommandParams) (string, error) {
	row := q.queryRow(ctx, q.getCommandStmt, getCommand, arg.ChannelID, arg.Name)
	var template string
	err := row.Scan(&template)
	return template, err
}

const getCommands = `-- name: GetCommands :many
SELECT name
  FROM commands
  WHERE channel_id = $1
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

const getGlobalCommands = `-- name: GetGlobalCommands :many
SELECT name, template
  FROM commands
  WHERE channel_id = '0'
  ORDER BY name ASC
`

type GetGlobalCommandsRow struct {
	Name     string
	Template string
}

func (q *Queries) GetGlobalCommands(ctx context.Context) ([]GetGlobalCommandsRow, error) {
	rows, err := q.query(ctx, q.getGlobalCommandsStmt, getGlobalCommands)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetGlobalCommandsRow
	for rows.Next() {
		var i GetGlobalCommandsRow
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
    channel_id = $1
    OR
    channel_id = $2
  )
  AND ($3::text ~ name)::bool
`

type GetMatchingCommandsParams struct {
	ChannelID       string
	ChannelGlobalID string
	Message         string
}

type GetMatchingCommandsRow struct {
	Template  string
	Name      string
	ChannelID string
}

func (q *Queries) GetMatchingCommands(ctx context.Context, arg GetMatchingCommandsParams) ([]GetMatchingCommandsRow, error) {
	rows, err := q.query(ctx, q.getMatchingCommandsStmt, getMatchingCommands, arg.ChannelID, arg.ChannelGlobalID, arg.Message)
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
  VALUES ($1, $2, $3)
  ON CONFLICT
  ON CONSTRAINT command_pkey DO UPDATE
  SET template = $3
`

type SetCommandParams struct {
	ChannelID string
	Name      string
	Template  string
}

func (q *Queries) SetCommand(ctx context.Context, arg SetCommandParams) error {
	_, err := q.exec(ctx, q.setCommandStmt, setCommand, arg.ChannelID, arg.Name, arg.Template)
	return err
}

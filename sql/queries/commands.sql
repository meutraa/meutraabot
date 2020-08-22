-- name: GetCommand :one
SELECT template
  FROM commands
  WHERE name = $2
  AND channel_name = $1;

-- name: GetMatchingCommands :many
SELECT
    template,
    name,
    channel_name
  FROM commands
  WHERE (
    channel_name = sqlc.arg('ChannelName')
    OR
    channel_name = '#'
  )
  AND (sqlc.arg('Message')::text ~ name)::bool;

-- name: GetCommands :many
SELECT name
  FROM commands
  WHERE channel_name = $1
  ORDER BY name ASC;

-- name: DeleteCommand :exec
DELETE FROM commands
  WHERE channel_name = $1
  AND name = $2;

-- name: SetCommand :exec
INSERT INTO commands (channel_name, name, template)
  VALUES ($1, $2, $3)
  ON CONFLICT
  ON CONSTRAINT command_pkey DO UPDATE
  SET template = $3;

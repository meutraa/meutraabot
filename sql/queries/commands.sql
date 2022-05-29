-- name: GetCommand :one
SELECT template
  FROM commands
  WHERE name = $2
  AND channel_id = $1;

-- name: GetMatchingCommands :many
SELECT
    template,
    name,
    channel_id
  FROM commands
  WHERE (
    channel_id = sqlc.arg('ChannelID')
    OR
    channel_id = sqlc.arg('ChannelGlobalID')
  )
  AND (sqlc.arg('Message')::text ~ name)::bool;

-- name: GetCommands :many
SELECT name
  FROM commands
  WHERE channel_id = $1
  ORDER BY name ASC;

-- name: GetCommandsByID :many
SELECT name, template
  FROM commands
  WHERE channel_id = $1
  ORDER BY name ASC;

-- name: DeleteCommand :exec
DELETE FROM commands
  WHERE channel_id = $1
  AND name = $2;

-- name: SetCommand :exec
INSERT INTO commands (channel_id, name, template)
  VALUES ($1, $2, $3)
  ON CONFLICT
  ON CONSTRAINT command_pkey DO UPDATE
  SET template = $3;

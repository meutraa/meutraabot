-- name: GetCommand :one
SELECT template
  FROM commands
  WHERE name = ?
  AND channel_id = ?;

-- name: GetMatchingCommands :many
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
  AND regexp(name, ?);

-- name: GetCommands :many
SELECT name
  FROM commands
  WHERE channel_id = ?
  ORDER BY name ASC;

-- name: GetCommandsByID :many
SELECT name, template
  FROM commands
  WHERE channel_id = ?
  ORDER BY name ASC;

-- name: DeleteCommand :exec
DELETE FROM commands
  WHERE channel_id = ?
  AND name = ?;

-- name: SetCommand :exec
INSERT INTO commands (channel_id, name, template)
  VALUES (?, ?, ?)
  ON CONFLICT(channel_id, name) DO UPDATE
  SET template = ?;

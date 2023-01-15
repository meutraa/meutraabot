-- name: GetNumber :one
SELECT * FROM numbers WHERE channel_id = ? AND name = ?;

-- name: AddToNumber :exec
INSERT INTO numbers (channel_id, name, value)
VALUES(?, ?, ?)
ON CONFLICT(channel_id, name) 
DO UPDATE SET value = value + ?;

-- name: SetCommand :exec
INSERT INTO commands (channel_id, name, template)
  VALUES (?, ?, ?)
  ON CONFLICT(channel_id, name) DO UPDATE
  SET template = ?;


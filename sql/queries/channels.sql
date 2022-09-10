-- name: GetChannels :many
SELECT channel_id FROM channels;

-- name: GetChannel :one
SELECT * FROM channels WHERE channel_id = ? ORDER BY created_at DESC;

-- name: UpdateChannel :exec
UPDATE channels
 SET autoreply_enabled = ?,
  autoreply_frequency = ?,
  reply_safety = ?
 WHERE channel_id = ?;

-- name: UpdateChannelToken :exec
UPDATE channels
 SET openai_token = ?
 WHERE channel_id = ?;

-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_id = ?;

-- name: CreateChannel :exec
INSERT INTO channels (channel_id)
  VALUES (?)
  ON CONFLICT DO NOTHING;

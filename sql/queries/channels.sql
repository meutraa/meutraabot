-- name: GetChannels :many
SELECT channel_id FROM channels;

-- name: GetChannel :one
SELECT * FROM channels WHERE channel_id = $1;

-- name: UpdateChannel :exec
UPDATE channels
 SET autoreply_enabled = $2,
  autoreply_frequency = $3,
  reply_safety = $4,
  updated_at = now()
 WHERE channel_id = $1;

-- name: UpdateChannelToken :exec
UPDATE channels
 SET openai_token = $2,
  updated_at = now()
 WHERE channel_id = $1;

-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_id = $1;

-- name: CreateChannel :exec
INSERT INTO channels (channel_id, created_at)
  VALUES ($1, NOW())
  ON CONFLICT DO NOTHING;

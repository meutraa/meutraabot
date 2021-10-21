-- name: GetChannels :many
SELECT channel_id FROM channels;

-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_id = $1;

-- name: CreateChannel :exec
INSERT INTO channels (channel_id, created_at)
  VALUES ($1, NOW())
  ON CONFLICT DO NOTHING;

-- name: GetChannelNames :many
SELECT channel_name FROM channels;

-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_name = $1;

-- name: CreateChannel :exec
INSERT INTO channels (channel_name, created_at)
  VALUES ($1, NOW())
  ON CONFLICT DO NOTHING;

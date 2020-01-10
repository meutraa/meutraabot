-- name: GetChannelNames :many
SELECT channel_name FROM channels;

-- name: DeleteChannel :exec
DELETE FROM channels
  WHERE channel_name = $1;

-- name: CreateChannel :exec
INSERT INTO channels (channel_name, created_at, hiccup_count)
  VALUES ($1, NOW(), 0)
  ON CONFLICT DO NOTHING;

-- name: UpdateHiccupCount :exec
UPDATE channels
  SET hiccup_count = hiccup_count + $2
  WHERE channel_name = $1;

-- name: GetHiccupCount :one
SELECT hiccup_count FROM channels
  WHERE channel_name = $1;

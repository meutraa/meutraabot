-- name: CreateMessage :exec
INSERT INTO messages
  (channel_name, sender, created_at, message)
  VALUES ($1, $2, NOW(), $3)
  ON CONFLICT DO NOTHING;

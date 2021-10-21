-- name: CreateMessage :exec
INSERT INTO messages
  (channel_id, sender_id, created_at, message)
  VALUES ($1, $2, NOW(), $3)
  ON CONFLICT DO NOTHING;

-- name: GetMessageCount :one
SELECT
  COUNT(*)
  FROM messages
  WHERE channel_id = $1
  AND sender_id = $2;

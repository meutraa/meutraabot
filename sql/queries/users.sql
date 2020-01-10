-- name: GetWatchTimeRank :one
SELECT CAST(RANK () OVER (
    PARTITION BY channel_name
    ORDER BY watch_time DESC
  ) AS INTEGER)
  FROM users
  WHERE channel_name = $1
  AND sender = $2;

-- name: UpdateEmoji :exec
UPDATE users
  SET emoji = $3
  WHERE channel_name = $1
  AND sender = $2;

-- name: GetMetrics :one
SELECT
  watch_time,
  message_count,
  word_count
  FROM users
  WHERE channel_name = $1
  AND sender = $2;

-- name: CreateUser :exec
INSERT INTO users
  (channel_name, sender, created_at, message_count, word_count, watch_time, text_color)
  VALUES ($1, $2, NOW(), 0, 0, 0, $3)
  ON CONFLICT DO NOTHING;

-- name: UpdateMetrics :exec
UPDATE users
  SET
    message_count = message_count + 1,
    word_count = word_count + $3,
    watch_time = CASE
      WHEN NOW() - updated_at < interval '15 minutes'
        THEN watch_time + extract(epoch from (NOW() - updated_at))
      ELSE watch_time
    END,
    updated_at = NOW()
  WHERE channel_name = $1
  AND sender = $2;

-- name: GetWatchTimeRank :one
SELECT
  cast(rank AS INTEGER)
  FROM
    (SELECT
      RANK() OVER (ORDER BY watch_time DESC) AS rank,
      sender
      FROM users
      WHERE channel_name = $1
  ) AS ss
  WHERE sender = $2;

-- name: GetWatchTimeRankAverage :one
SELECT
  cast(rank AS INTEGER)
  FROM
    (SELECT
      RANK() OVER (ORDER BY (watch_time / extract(epoch from (NOW() - created_at))) DESC) AS rank,
      sender
      FROM users
      WHERE channel_name = $1
  ) AS ss
  WHERE sender = $2;

-- name: GetTopWatchers :many
SELECT
  sender
  FROM users
  WHERE channel_name = $1
  ORDER BY ((watch_time/60) + (word_count / 8)) DESC
  LIMIT $2;

-- name: GetTopWatchersAverage :many
SELECT
  sender
  FROM users
  WHERE channel_name = $1
  ORDER BY (((watch_time/60) + (word_count / 8)) / extract(epoch from (NOW() - created_at))) DESC
  LIMIT $2;

-- name: GetMetrics :one
SELECT
  watch_time,
  message_count,
  word_count,
  extract(epoch from (NOW() - created_at)) as age,
  (watch_time/60) + (word_count / 8) as points,
  created_at
  FROM users
  WHERE channel_name = $1
  AND sender = $2;

-- name: CreateUser :exec
INSERT INTO users
  (channel_name, sender, created_at, message_count, word_count, watch_time)
  VALUES ($1, $2, NOW(), 0, 0, 0)
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

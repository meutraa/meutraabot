-- name: UpdateCounter :exec
INSERT INTO counter (channel_id, name, value)
  VALUES ($1, $2, $3)
  ON CONFLICT
  ON CONSTRAINT counter_pkey DO UPDATE
  SET value = counter.value + $3;

-- name: GetCounter :one
SELECT value FROM counter
  WHERE channel_id = $1
  AND name = $2;

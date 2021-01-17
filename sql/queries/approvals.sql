-- name: IsApproved :one
SELECT
  COUNT(*)
FROM
  approvals
WHERE
  channel_name = $1
  AND username = $2;

-- name: Approve :exec
INSERT INTO
  approvals (channel_name, username)
VALUES
  ($1, $2) ON CONFLICT DO NOTHING;

-- name: Unapprove :exec
DELETE FROM
  approvals
WHERE
  channel_name = $1
  AND username = $2;
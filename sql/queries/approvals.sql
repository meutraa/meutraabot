-- name: IsApproved :one
SELECT
  COUNT(*)
FROM
  approvals
WHERE
  channel_id = $1
  AND user_id = $2;

-- name: Approve :exec
INSERT INTO
  approvals (channel_id, user_id)
VALUES
  ($1, $2) ON CONFLICT DO NOTHING;

-- name: Unapprove :exec
DELETE FROM
  approvals
WHERE
  channel_id = $1
  AND user_id = $2;
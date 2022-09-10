-- name: IsApproved :one
SELECT
  COUNT(*)
FROM
  approvals
WHERE
  channel_id = ?
  AND user_id = ?;

-- name: GetApprovals :many
SELECT
  *
FROM
  approvals
WHERE
  channel_id = ?
  AND manual = true
ORDER BY user_id DESC;

-- name: Approve :exec
INSERT INTO
  approvals (channel_id, user_id, manual)
VALUES
  (?, ?, ?) ON CONFLICT DO NOTHING;

-- name: Unapprove :exec
DELETE FROM
  approvals
WHERE
  channel_id = ?
  AND user_id = ?;
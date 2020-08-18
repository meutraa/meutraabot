-- name: GetBannedUsers :many
SELECT username FROM bans;

-- name: IsUserBanned :one
SELECT (0 != COUNT(username)) as IsBanned FROM bans
  WHERE username = $1;

-- name: BanUser :exec
INSERT INTO bans (username)
  VALUES ($1)
  ON CONFLICT DO NOTHING;

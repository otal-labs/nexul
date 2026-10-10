-- name: GetGitHubUserLink :one
SELECT * FROM github_user_links WHERE user_id = ?;

-- name: SaveGitHubUserLink :exec
INSERT INTO github_user_links (user_id, access_token, refresh_token, expires_at, refresh_expires_at, needs_reconnect, connected_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
  access_token = excluded.access_token, refresh_token = excluded.refresh_token, expires_at = excluded.expires_at,
  refresh_expires_at = excluded.refresh_expires_at, needs_reconnect = excluded.needs_reconnect,
  connected_at = excluded.connected_at;

-- name: DeleteGitHubUserLink :exec
DELETE FROM github_user_links WHERE user_id = ?;

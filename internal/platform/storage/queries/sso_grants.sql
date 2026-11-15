-- name: UpsertSSOGrant :exec
INSERT INTO sso_grants (user_id, refresh_token, checked_at) VALUES (?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET refresh_token = excluded.refresh_token, checked_at = excluded.checked_at;

-- name: ListStaleSSOGrants :many
SELECT user_id, refresh_token FROM sso_grants WHERE checked_at < ? ORDER BY checked_at, user_id LIMIT ?;

-- name: TouchSSOGrant :exec
UPDATE sso_grants SET refresh_token = COALESCE(sqlc.narg('refresh_token'), refresh_token), checked_at = sqlc.arg('checked_at')
WHERE user_id = sqlc.arg('user_id');

-- name: DeleteSSOGrant :exec
DELETE FROM sso_grants WHERE user_id = ?;

-- name: DeleteSSOGrants :exec
DELETE FROM sso_grants;

-- name: ListSSOSessionsByUser :many
SELECT * FROM sessions WHERE user_id = ? AND sso = 1;

-- name: MoveSSOGrant :exec
UPDATE sso_grants SET user_id = sqlc.arg(to_user_id) WHERE user_id = sqlc.arg(from_user_id);

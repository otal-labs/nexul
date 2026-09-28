-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token_hash, client, platform, label, ip, created_at, last_active_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSessionByHash :one
SELECT * FROM sessions WHERE token_hash = ?;

-- name: ListSessionsByUser :many
SELECT * FROM sessions WHERE user_id = ? ORDER BY last_active_at DESC, id;

-- name: DeleteSession :execrows
DELETE FROM sessions WHERE id = ? AND user_id = ?;

-- name: DeleteOtherSessions :exec
DELETE FROM sessions WHERE user_id = ? AND id != ?;

-- name: TouchSession :exec
UPDATE sessions SET last_active_at = ?, ip = ?, expires_at = ? WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE user_id = ? AND expires_at <= ?;

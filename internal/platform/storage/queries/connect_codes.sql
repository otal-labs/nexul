-- name: DeleteConnectCodesByUser :exec
DELETE FROM connect_codes WHERE user_id = ?;

-- name: CreateConnectCode :exec
INSERT INTO connect_codes (code_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?);

-- name: GetLiveConnectCodeUser :one
SELECT user_id FROM connect_codes WHERE code_hash = ? AND expires_at > ?;

-- name: DeleteConnectCode :exec
DELETE FROM connect_codes WHERE code_hash = ?;

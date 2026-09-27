-- name: DeleteSetupCodes :exec
DELETE FROM setup_codes;

-- name: CreateSetupCode :exec
INSERT INTO setup_codes (code_hash, created_at, expires_at) VALUES (?, ?, ?);

-- name: CountLiveSetupCodes :one
SELECT COUNT(*) FROM setup_codes WHERE code_hash = ? AND expires_at > ?;

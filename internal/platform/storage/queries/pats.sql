-- name: CreatePAT :exec
INSERT INTO personal_access_tokens (id, user_id, name, token_hash, prefix, created_at, computer_id, oauth_client_id, refresh_hash, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPATByHash :one
SELECT * FROM personal_access_tokens WHERE token_hash = ?;

-- name: ListPATsByUser :many
SELECT * FROM personal_access_tokens WHERE user_id = ? ORDER BY created_at DESC;

-- name: RevokePAT :execrows
UPDATE personal_access_tokens SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL;

-- name: TouchPATLastUsed :exec
UPDATE personal_access_tokens SET last_used_at = ? WHERE id = ? AND revoked_at IS NULL;

-- name: GetActiveComputerPAT :one
SELECT * FROM personal_access_tokens
WHERE user_id = ? AND computer_id = ? AND computer_id != '' AND revoked_at IS NULL;

-- name: RotatePATByRefresh :one
UPDATE personal_access_tokens
SET token_hash = sqlc.arg(token_hash), refresh_hash = sqlc.arg(refresh_hash), expires_at = sqlc.arg(expires_at), prefix = sqlc.arg(prefix)
WHERE refresh_hash = sqlc.arg(old_refresh_hash) AND oauth_client_id = sqlc.arg(oauth_client_id) AND revoked_at IS NULL
RETURNING *;

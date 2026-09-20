-- name: CreatePAT :exec
INSERT INTO personal_access_tokens (id, user_id, name, token_hash, prefix, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetPATByHash :one
SELECT * FROM personal_access_tokens WHERE token_hash = ?;

-- name: ListPATsByUser :many
SELECT * FROM personal_access_tokens WHERE user_id = ? ORDER BY created_at DESC;

-- name: RevokePAT :execrows
UPDATE personal_access_tokens SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL;

-- name: TouchPATLastUsed :exec
UPDATE personal_access_tokens SET last_used_at = ? WHERE id = ? AND revoked_at IS NULL;

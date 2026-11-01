-- name: InsertPasskey :exec
INSERT INTO user_passkeys (id, user_id, name, rp_id, credential, created_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetPasskey :one
SELECT * FROM user_passkeys WHERE id = ?;

-- name: ListPasskeysByUser :many
SELECT * FROM user_passkeys WHERE user_id = ? ORDER BY created_at, id;

-- name: UpdatePasskeyUse :execrows
UPDATE user_passkeys SET credential = ?, last_used_at = ? WHERE id = ?;

-- name: RenamePasskey :execrows
UPDATE user_passkeys SET name = ? WHERE id = ? AND user_id = ?;

-- name: DeletePasskey :execrows
DELETE FROM user_passkeys WHERE id = ? AND user_id = ?;

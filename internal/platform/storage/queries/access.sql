-- name: GetOverwrite :one
SELECT * FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? AND user_id = ?;

-- name: ListOverwritesByResource :many
SELECT * FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? ORDER BY user_id;

-- name: UpsertOverwrite :execrows
INSERT INTO permission_overwrites (resource_type, resource_id, user_id, allow, deny, created_at, updated_at)
VALUES (sqlc.arg(resource_type), sqlc.arg(resource_id), sqlc.arg(user_id), sqlc.arg(allow), sqlc.arg(deny), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT(resource_type, resource_id, user_id) DO UPDATE SET
  allow = excluded.allow, deny = excluded.deny, updated_at = excluded.updated_at;

-- name: DeleteOverwrite :execrows
DELETE FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? AND user_id = ?;

-- name: DeleteOverwritesByResource :exec
DELETE FROM permission_overwrites WHERE resource_type = ? AND resource_id = ?;

-- name: CountOverwriteAllowAny :one
SELECT COUNT(*) FROM permission_overwrites
WHERE resource_type = ? AND user_id = ? AND instr(allow, sqlc.arg(quoted_action)) > 0;

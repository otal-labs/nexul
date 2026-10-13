-- name: CreateRole :exec
INSERT INTO roles (id, workspace_id, name, permissions, welcome_steps, is_owner_role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetRole :one
SELECT * FROM roles WHERE id = ?;

-- name: ListRoles :many
SELECT * FROM roles WHERE workspace_id = ? ORDER BY is_owner_role DESC, created_at, id;

-- name: UpdateRole :execrows
UPDATE roles SET name = ?, permissions = ?, welcome_steps = ?, updated_at = ? WHERE id = ?;

-- name: DeleteRole :execrows
DELETE FROM roles WHERE id = ?;

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

-- name: ListRoleWelcomeStepsForUsers :many
SELECT mr.user_id, r.welcome_steps, r.is_owner_role FROM member_roles mr JOIN roles r ON r.id = mr.role_id
WHERE mr.user_id IN (SELECT value FROM json_each(sqlc.arg(user_ids)));

-- name: ListRoleWelcomeStepsForRoles :many
SELECT welcome_steps, is_owner_role FROM roles WHERE id IN (SELECT value FROM json_each(sqlc.arg(role_ids)));

-- name: ListRoleHolders :many
SELECT DISTINCT user_id FROM member_roles WHERE role_id = ? ORDER BY user_id;

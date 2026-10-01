-- name: SetAutomationSecret :exec
INSERT INTO automation_secrets (workspace_id, name, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(workspace_id, name) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at;

-- name: DeleteAutomationSecret :exec
DELETE FROM automation_secrets WHERE workspace_id = ? AND name = ?;

-- name: ListAutomationSecretMeta :many
SELECT name, created_at, updated_at FROM automation_secrets WHERE workspace_id = ? ORDER BY name;

-- name: ListAutomationSecretValues :many
SELECT name, value FROM automation_secrets WHERE workspace_id = ?;

-- name: CreateAutomation :exec
INSERT INTO automations (id, workspace_id, name, description, kind, enabled, subscriptions, config_schema, config_values, scopes, created_by, token_hash, token_prefix, token_revoked_at, host_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAutomation :one
SELECT sqlc.embed(a), EXISTS (SELECT 1 FROM workspaces w WHERE w.id = a.workspace_id AND w.archived_at IS NOT NULL) AS workspace_archived
FROM automations a WHERE a.id = ?;

-- name: GetAutomationByTokenHash :one
SELECT sqlc.embed(a), EXISTS (SELECT 1 FROM workspaces w WHERE w.id = a.workspace_id AND w.archived_at IS NOT NULL) AS workspace_archived
FROM automations a WHERE a.token_hash = ?;

-- name: ListAutomations :many
SELECT sqlc.embed(a), EXISTS (SELECT 1 FROM workspaces w WHERE w.id = a.workspace_id AND w.archived_at IS NOT NULL) AS workspace_archived
FROM automations a ORDER BY a.created_at;

-- name: UpdateAutomation :execrows
UPDATE automations SET name = ?, description = ?, kind = ?, enabled = ?, subscriptions = ?, config_schema = ?, config_values = ?, scopes = ?, token_hash = ?, token_prefix = ?, token_revoked_at = ?, host_id = ?, updated_at = ? WHERE id = ?;

-- name: DeleteAutomation :execrows
DELETE FROM automations WHERE id = ?;

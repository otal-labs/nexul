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

-- name: ListAutomationHealth :many
SELECT a.id,
       lr.outcome AS last_outcome, lr.started_at AS last_started_at, lr.error AS last_error,
       (SELECT COUNT(*) FROM automation_runs r WHERE r.automation_id = a.id AND r.started_at >= sqlc.arg(since)) AS runs_total,
       (SELECT COUNT(*) FROM automation_runs r WHERE r.automation_id = a.id AND r.started_at >= sqlc.arg(since) AND r.outcome <> 'success') AS runs_failed,
       pv.sequence AS pending_sequence
FROM automations a
LEFT JOIN automation_runs lr ON lr.id = (
  SELECT r.id FROM automation_runs r WHERE r.automation_id = a.id ORDER BY r.started_at DESC, r.id DESC LIMIT 1
)
LEFT JOIN automation_versions pv ON pv.automation_id = a.id AND pv.status = 'pending'
WHERE a.id IN (SELECT value FROM json_each(sqlc.arg(ids)));

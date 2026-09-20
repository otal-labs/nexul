-- name: CreateAutomation :exec
INSERT INTO automations (id, name, description, kind, enabled, subscriptions, config_schema, config_values, scopes, created_by, token_hash, token_prefix, token_revoked_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAutomation :one
SELECT * FROM automations WHERE id = ?;

-- name: GetAutomationByTokenHash :one
SELECT * FROM automations WHERE token_hash = ?;

-- name: ListAutomations :many
SELECT * FROM automations ORDER BY created_at;

-- name: UpdateAutomation :execrows
UPDATE automations SET name = ?, description = ?, kind = ?, enabled = ?, subscriptions = ?, config_schema = ?, config_values = ?, scopes = ?, token_hash = ?, token_prefix = ?, token_revoked_at = ?, updated_at = ? WHERE id = ?;

-- name: DeleteAutomation :execrows
DELETE FROM automations WHERE id = ?;

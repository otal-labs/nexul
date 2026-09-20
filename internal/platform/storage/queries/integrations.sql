-- name: CreateIntegrationInstall :exec
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at, revoked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetIntegrationInstall :one
SELECT * FROM integration_installs WHERE id = ?;

-- name: ListIntegrationInstalls :many
SELECT * FROM integration_installs ORDER BY created_at;

-- name: RevokeIntegrationInstall :execrows
UPDATE integration_installs SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL;

-- name: CreateIntegrationToken :exec
INSERT INTO integration_tokens (id, install_id, token_hash, prefix, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetIntegrationTokenByHash :one
SELECT * FROM integration_tokens WHERE token_hash = ?;

-- name: ListIntegrationTokensByInstall :many
SELECT * FROM integration_tokens WHERE install_id = ? ORDER BY created_at DESC;

-- name: RevokeIntegrationTokensByInstall :exec
UPDATE integration_tokens SET revoked_at = ? WHERE install_id = ? AND revoked_at IS NULL;

-- name: TouchIntegrationTokenLastUsed :exec
UPDATE integration_tokens SET last_used_at = ? WHERE id = ? AND revoked_at IS NULL;

-- name: GetUserIDByProvider :one
SELECT id FROM users WHERE provider = ? AND provider_user_id = ?;

-- name: InsertUser :exec
INSERT INTO users (id, provider, provider_user_id, login, name, avatar_url, can_create_workspace, first_login_done, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, 0, 0, ?, ?);

-- name: SyncUser :exec
UPDATE users SET login = ?, name = ?, avatar_url = ?, updated_at = ? WHERE id = ?;

-- name: GetUserByProvider :one
SELECT * FROM users WHERE provider = ? AND provider_user_id = ?;

-- name: SetAccountStatus :execrows
UPDATE users SET account_status = ?, updated_at = ? WHERE id = ?;

-- name: CountActiveAdmins :one
SELECT COUNT(*) FROM users WHERE can_create_workspace = 1 AND account_status = 'active';

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: CountCanCreateWorkspace :one
SELECT COUNT(*) FROM users WHERE can_create_workspace = 1;

-- name: SetCanCreateWorkspace :execrows
UPDATE users SET can_create_workspace = ?, updated_at = ? WHERE id = ?;

-- name: MarkFirstLoginDone :execrows
UPDATE users SET first_login_done = 1, updated_at = ? WHERE id = ?;

-- name: SetProfileOverride :execrows
UPDATE users SET display_name = ?, avatar_override_url = ?, updated_at = ? WHERE id = ?;

-- name: GetUserByLogin :one
SELECT * FROM users WHERE lower(login) = lower(?);

-- name: ListUsers :many
SELECT * FROM users ORDER BY login;

-- name: AddAllowlistMember :exec
INSERT INTO allowlist (login, added_at) VALUES (?, ?);

-- name: RemoveAllowlistMember :execrows
DELETE FROM allowlist WHERE login = ?;

-- name: CountAllowlistMember :one
SELECT COUNT(*) FROM allowlist WHERE login = ?;

-- name: ListAllowlist :many
SELECT login FROM allowlist ORDER BY login;

-- name: GetSettings :one
SELECT * FROM instance_settings WHERE id = 1;

-- name: SetInstanceURL :execrows
UPDATE instance_settings SET instance_url = ?, settings_version = settings_version + 1, updated_at = ? WHERE id = 1;

-- name: SetSettingsMentionChipTemplate :execrows
UPDATE instance_settings SET mention_chip_template = ?, updated_at = ? WHERE id = 1;

-- name: SetSettingsGitHubOAuth :execrows
UPDATE instance_settings SET github_oauth_client_id = ?, github_oauth_client_secret = ?, updated_at = ? WHERE id = 1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: InsertUser :exec
INSERT INTO users (id, login, name, avatar_url, can_create_workspace, first_login_done, created_at, updated_at)
VALUES (?, ?, ?, ?, 0, 0, ?, ?);

-- name: GetUserByIdentity :one
SELECT u.* FROM users u JOIN user_identities i ON i.user_id = u.id
WHERE i.provider = ? AND i.provider_user_id = ?;

-- name: InsertIdentity :exec
INSERT INTO user_identities (user_id, provider, provider_user_id, login, name, avatar_url, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: SyncIdentity :exec
UPDATE user_identities SET login = ?, name = ?, avatar_url = ? WHERE provider = ? AND provider_user_id = ?;

-- name: SyncUserFromFirstIdentity :exec
UPDATE users SET login = ?, name = ?, avatar_url = ?, updated_at = ?
WHERE id = sqlc.arg(user_id)
  AND sqlc.arg(provider) = (SELECT provider FROM user_identities WHERE user_id = sqlc.arg(user_id) ORDER BY created_at, provider LIMIT 1);

-- name: ListIdentitiesByUser :many
SELECT * FROM user_identities WHERE user_id = ? ORDER BY created_at, provider;

-- name: CountIdentitiesByUser :one
SELECT COUNT(*) FROM user_identities WHERE user_id = ?;

-- name: DeleteIdentity :execrows
DELETE FROM user_identities WHERE user_id = ? AND provider = ?;

-- name: SetAccountStatus :execrows
UPDATE users SET account_status = ?, updated_at = ? WHERE id = ?;

-- name: CountActiveAdmins :one
SELECT COUNT(*) FROM users WHERE can_create_workspace = 1 AND account_status = 'active';

-- name: DeleteAccountMemberships :exec
DELETE FROM workspace_members WHERE user_id = ?;

-- name: DeleteAccountOverwrites :exec
DELETE FROM permission_overwrites WHERE user_id = ?;

-- name: RevokeAccountPATs :exec
UPDATE personal_access_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL;

-- name: DeleteAccountPairingComputers :exec
DELETE FROM pairing_computers WHERE user_id = ?;

-- name: DeleteAccountPairingDefaults :exec
DELETE FROM pairing_user_defaults WHERE user_id = ?;

-- name: DeleteAccountInvitations :exec
DELETE FROM invitations WHERE invited_by = ? AND redeemed_at IS NULL;

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

-- name: SetSettingsGitHubOAuth :execrows
UPDATE instance_settings SET github_oauth_client_id = ?, github_oauth_client_secret = ?, updated_at = ? WHERE id = 1;

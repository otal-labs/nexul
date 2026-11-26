-- name: GetSSOSettings :one
SELECT * FROM sso_settings WHERE id = 1;

-- name: SetSSOSettings :exec
INSERT INTO sso_settings (id, issuer_url, client_id, client_secret, display_name, join_role_id, sso_only, updated_by, updated_at)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET issuer_url = excluded.issuer_url, client_id = excluded.client_id,
    client_secret = excluded.client_secret, display_name = excluded.display_name, join_role_id = excluded.join_role_id,
    sso_only = excluded.sso_only, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: DeleteSSOSettings :exec
DELETE FROM sso_settings WHERE id = 1;

-- name: FindUserByEmailIdentity :one
SELECT u.id, u.account_status FROM user_identities i JOIN users u ON u.id = i.user_id
WHERE i.provider IN ('google', 'discord') AND i.login = ? COLLATE NOCASE
ORDER BY i.created_at, u.id LIMIT 1;

-- name: GetSSOJoinRole :one
SELECT is_owner_role FROM roles WHERE id = ? AND workspace_id = ?;

-- name: MoveSSOIdentity :execrows
UPDATE user_identities SET user_id = sqlc.arg(to_user_id) WHERE user_id = sqlc.arg(from_user_id) AND provider IN ('oidc', 'saml');

-- name: ListSSOOldSignIns :many
SELECT u.id, u.login, u.name, u.display_name, u.created_at, i.login AS email
FROM user_identities i JOIN users u ON u.id = i.user_id
WHERE i.provider IN ('oidc', 'saml') AND i.provider <> sqlc.arg(current_provider) AND u.account_status = 'active'
  AND NOT EXISTS (SELECT 1 FROM user_identities c WHERE c.user_id = u.id AND c.provider = sqlc.arg(current_provider))
ORDER BY i.login COLLATE NOCASE, u.id;

-- name: ListSSOOnlyAccounts :many
SELECT u.id, u.login, u.name, u.display_name, u.created_at, i.login AS email
FROM user_identities i JOIN users u ON u.id = i.user_id
WHERE i.provider = sqlc.arg(current_provider) AND u.account_status = 'active'
  AND NOT EXISTS (SELECT 1 FROM user_identities o WHERE o.user_id = u.id AND o.provider <> sqlc.arg(current_provider))
ORDER BY i.login COLLATE NOCASE, u.id;

-- name: ListSSOExpectedLinks :many
SELECT e.user_id, e.email, e.created_by, e.created_at, u.login, u.name, u.display_name
FROM sso_expected_links e JOIN users u ON u.id = e.user_id
ORDER BY e.email, e.user_id;

-- name: GetSSOExpectedLinkByEmail :one
SELECT u.id, u.account_status FROM sso_expected_links e JOIN users u ON u.id = e.user_id WHERE e.email = ?;

-- name: CountSSOIdentitiesOfProvider :one
SELECT COUNT(*) FROM user_identities WHERE user_id = ? AND provider = ?;

-- name: UpsertSSOExpectedLink :exec
INSERT INTO sso_expected_links (user_id, email, created_by, created_at) VALUES (?, ?, ?, ?)
ON CONFLICT (user_id) DO UPDATE SET email = excluded.email, created_by = excluded.created_by, created_at = excluded.created_at;

-- name: GetSSOExpectedLinkEmail :one
SELECT email FROM sso_expected_links WHERE user_id = ?;

-- name: DeleteSSOExpectedLink :execrows
DELETE FROM sso_expected_links WHERE user_id = ?;

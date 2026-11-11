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

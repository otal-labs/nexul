-- name: CreateInvitation :exec
INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at)
VALUES (?, ?, ?, ?, ?);

-- name: CreateInvitationGrant :exec
INSERT INTO invitation_grants (invitation_id, workspace_id, role_id, allow_json, deny_json)
VALUES (?, ?, ?, ?, ?);

-- name: GetInvitationByTokenHash :one
SELECT id, token_hash, invited_by, created_at, expires_at, redeemed_at, redeemed_by
FROM invitations
WHERE token_hash = ?;

-- name: GetInvitationByID :one
SELECT id, token_hash, invited_by, created_at, expires_at, redeemed_at, redeemed_by
FROM invitations
WHERE id = ?;

-- name: ListInvitationsByActor :many
SELECT id, token_hash, invited_by, created_at, expires_at, redeemed_at, redeemed_by
FROM invitations
WHERE invited_by = ?
ORDER BY created_at, id;

-- name: ListAllInvitations :many
SELECT id, token_hash, invited_by, created_at, expires_at, redeemed_at, redeemed_by
FROM invitations
ORDER BY created_at, id;

-- name: ListManageableInvitationHeaders :many
SELECT i.id, i.token_hash, i.invited_by, i.created_at, i.expires_at, i.redeemed_at, i.redeemed_by
FROM invitations i
WHERE i.redeemed_at IS NULL
  AND i.expires_at > ?
  AND NOT EXISTS (
      SELECT 1
      FROM invitation_grants g
      WHERE g.invitation_id = i.id
        AND NOT EXISTS (
            SELECT 1
            FROM workspace_members m
            JOIN roles r ON r.id = m.role_id
            LEFT JOIN permission_overwrites po
              ON po.resource_type = 'workspace'
             AND po.resource_id = g.workspace_id
             AND po.user_id = m.user_id
            WHERE m.workspace_id = g.workspace_id
              AND m.user_id = ?
              AND (
                  r.is_owner_role = 1
                  OR (
                      instr(COALESCE(po.deny, '[]'), '"members:write"') = 0
                      AND (
                          instr(r.permissions, '"members:write"') > 0
                          OR instr(COALESCE(po.allow, '[]'), '"members:write"') > 0
                      )
                  )
              )
        )
  )
ORDER BY i.created_at, i.id
LIMIT ?;

-- name: ListInvitationGrantsByInvitation :many
SELECT invitation_id, workspace_id, role_id, allow_json, deny_json
FROM invitation_grants
WHERE invitation_id = ?
ORDER BY workspace_id;

-- name: DeleteInvitation :execrows
DELETE FROM invitations WHERE id = ?;

-- name: DeleteExpiredInvitations :exec
DELETE FROM invitations WHERE expires_at <= ?;

-- name: GetWorkspaceForInvitationGrant :one
SELECT w.id, w.name
FROM workspaces w
WHERE w.id = ?;

-- name: GetRoleForInvitationGrant :one
SELECT id, workspace_id, name, is_owner_role, created_at, updated_at, permissions
FROM roles
WHERE id = ? AND workspace_id = ?;

-- name: GetInvitationCreatorWorkspaceAccess :one
SELECT r.is_owner_role, r.permissions, po.allow, po.deny
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
LEFT JOIN permission_overwrites po
  ON po.resource_type = 'workspace'
 AND po.resource_id = m.workspace_id
 AND po.user_id = m.user_id
WHERE m.workspace_id = ? AND m.user_id = ?;

-- name: GetInvitationMember :one
SELECT role_id
FROM workspace_members
WHERE workspace_id = ? AND user_id = ?;

-- name: AddInvitationMember :exec
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, workspace_id) DO NOTHING;

-- name: AddInvitationOverwrite :exec
INSERT INTO permission_overwrites (resource_type, resource_id, user_id, allow, deny, created_at, updated_at)
VALUES ('workspace', ?, ?, ?, ?, ?, ?)
ON CONFLICT(resource_type, resource_id, user_id) DO NOTHING;

-- name: GetUserByProviderForInvitation :one
SELECT * FROM users WHERE provider = ? AND provider_user_id = ?;

-- name: InsertUserForInvitation :exec
INSERT INTO users (id, provider, provider_user_id, login, name, avatar_url, first_login_done, created_at, updated_at, can_create_workspace, account_status)
VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, 0, 'active');

-- name: DeleteInvitationBeforeRedeem :execrows
DELETE FROM invitations
WHERE id = ? AND token_hash = ? AND expires_at > ?;

-- name: MarkInvitationRedeemed :execrows
UPDATE invitations
SET redeemed_at = ?, redeemed_by = ?
WHERE id = ? AND redeemed_at IS NULL AND expires_at > ?;

-- name: SetInvitationRedeemedBy :execrows
UPDATE invitations SET redeemed_by = ? WHERE id = ? AND redeemed_at IS NOT NULL;

-- name: GetOAuthHandoffByState :one
SELECT id, invitation_id, oauth_state_hash, acceptance_hash, provider, provider_user_id, login, name, avatar_url, existing_user_id, admitted_user_id, completed_at, created_at, expires_at
FROM invitation_oauth_handoffs
WHERE oauth_state_hash = ?;

-- name: GetOAuthHandoffByAcceptanceHash :one
SELECT id, invitation_id, oauth_state_hash, acceptance_hash, provider, provider_user_id, login, name, avatar_url, existing_user_id, admitted_user_id, completed_at, created_at, expires_at
FROM invitation_oauth_handoffs
WHERE acceptance_hash = ?;

-- name: CreateOAuthHandoff :exec
INSERT INTO invitation_oauth_handoffs (id, invitation_id, oauth_state_hash, acceptance_hash, provider, created_at, expires_at)
VALUES (?, ?, ?, NULL, ?, ?, ?);

-- name: CompleteOAuthHandoffTransition :execrows
UPDATE invitation_oauth_handoffs
SET oauth_state_hash = NULL,
    acceptance_hash = ?,
    provider_user_id = ?,
    login = ?,
    name = ?,
    avatar_url = ?,
    existing_user_id = ?,
    expires_at = ?
WHERE oauth_state_hash = ? AND expires_at > ? AND acceptance_hash IS NULL;

-- name: CompleteOAuthHandoffRedemption :execrows
UPDATE invitation_oauth_handoffs
SET admitted_user_id = ?, completed_at = ?
WHERE acceptance_hash = ? AND completed_at IS NULL AND expires_at > ?;

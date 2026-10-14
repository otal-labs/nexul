-- name: CreateInvitation :exec
INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at)
VALUES (?, ?, ?, ?, ?);

-- name: CreateInvitationGrant :exec
INSERT INTO invitation_grants (invitation_id, workspace_id, role_id) VALUES (?, ?, ?);

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
            FROM member_roles mr
            JOIN roles r ON r.id = mr.role_id
            WHERE mr.workspace_id = g.workspace_id
              AND mr.user_id = ?
              AND (r.is_owner_role = 1 OR instr(r.permissions, '"members:write"') > 0)
        )
  )
ORDER BY i.created_at, i.id
LIMIT ?;

-- name: ListManageableInvitationHistory :many
-- The links accepted, or expired or ended early, since a time, newest first, among those the actor may manage.
SELECT i.id, i.token_hash, i.invited_by, i.created_at, i.expires_at, i.redeemed_at, i.redeemed_by
FROM invitations i
WHERE (i.redeemed_at > sqlc.arg(since) OR (i.redeemed_at IS NULL AND i.expires_at > sqlc.arg(since) AND i.expires_at <= sqlc.arg(now)))
  AND EXISTS (SELECT 1 FROM invitation_grants g WHERE g.invitation_id = i.id)
  AND NOT EXISTS (
      SELECT 1
      FROM invitation_grants g
      WHERE g.invitation_id = i.id
        AND NOT EXISTS (
            SELECT 1
            FROM member_roles mr
            JOIN roles r ON r.id = mr.role_id
            WHERE mr.workspace_id = g.workspace_id
              AND mr.user_id = sqlc.arg(user_id)
              AND (r.is_owner_role = 1 OR instr(r.permissions, '"members:write"') > 0)
        )
  )
ORDER BY COALESCE(i.redeemed_at, i.expires_at) DESC, i.id
LIMIT sqlc.arg(limit);

-- name: ListInvitationJoins :many
-- Each person's first accepted invitation; SQLite takes invited_by from the row MIN picks.
SELECT redeemed_by, invited_by, MIN(redeemed_at) AS redeemed_at
FROM invitations
WHERE redeemed_by IS NOT NULL AND redeemed_at IS NOT NULL
GROUP BY redeemed_by;

-- name: ListInvitationGrantsByInvitation :many
SELECT *
FROM invitation_grants
WHERE invitation_id = ?
ORDER BY workspace_id, role_id;

-- name: EndInvitation :execrows
-- Ending a link early moves its expiry to now, so it stays on record as an expired link.
UPDATE invitations SET expires_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND redeemed_at IS NULL AND expires_at > sqlc.arg(now);

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

-- name: ListInvitationCreatorWorkspaceAccess :many
SELECT COALESCE(r.is_owner_role, 0) AS is_owner_role, COALESCE(r.permissions, '[]') AS permissions
FROM workspace_members m
LEFT JOIN member_roles mr ON mr.workspace_id = m.workspace_id AND mr.user_id = m.user_id
LEFT JOIN roles r ON r.id = mr.role_id
WHERE m.workspace_id = ? AND m.user_id = ?;

-- name: AddInvitationMember :exec
INSERT INTO workspace_members (user_id, workspace_id, created_at)
VALUES (?, ?, ?)
ON CONFLICT(user_id, workspace_id) DO NOTHING;

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

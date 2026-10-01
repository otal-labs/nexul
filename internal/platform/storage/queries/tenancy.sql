-- name: CreateWorkspace :exec
INSERT INTO workspaces (id, name, slug, mention_chip_template, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateWorkspace :execrows
UPDATE workspaces SET name = ?, slug = ?, mention_chip_template = ?, updated_at = ? WHERE id = ?;

-- name: GetWorkspace :one
SELECT id, name, created_at, updated_at, mention_chip_template, slug, decisions_check_enabled FROM workspaces WHERE id = ?;

-- name: GetWorkspaceBySlug :one
SELECT id, name, created_at, updated_at, mention_chip_template, slug, decisions_check_enabled FROM workspaces WHERE slug = ?;

-- name: ListWorkspacesForUser :many
SELECT w.id, w.name, w.created_at, w.updated_at, w.mention_chip_template, w.slug, w.decisions_check_enabled
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.user_id = ?
ORDER BY w.created_at, w.id;

-- name: AddWorkspaceMember :exec
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at) VALUES (?, ?, ?, ?);

-- name: GetWorkspaceMemberRole :one
SELECT role_id FROM workspace_members WHERE workspace_id = ? AND user_id = ?;

-- name: ListWorkspaceMembers :many
SELECT user_id, workspace_id, role_id, created_at FROM workspace_members WHERE workspace_id = ? ORDER BY created_at, user_id;

-- name: RemoveWorkspaceMember :execrows
DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?;

-- name: SetWorkspaceMemberRole :execrows
UPDATE workspace_members SET role_id = ? WHERE workspace_id = ? AND user_id = ?;

-- name: UpsertWorkspaceInvite :exec
INSERT INTO workspace_invites (workspace_id, login, role_id, invited_by, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(workspace_id, login) DO UPDATE SET
    role_id = excluded.role_id,
    invited_by = excluded.invited_by,
    created_at = excluded.created_at;

-- name: DeleteWorkspaceInvite :exec
DELETE FROM workspace_invites WHERE workspace_id = ? AND login = ?;

-- name: ListWorkspaceInvitesByWorkspace :many
SELECT workspace_id, login, role_id, invited_by, created_at
FROM workspace_invites WHERE workspace_id = ? ORDER BY created_at, login;

-- name: ListWorkspaceInvitesByLogin :many
SELECT workspace_id, login, role_id, invited_by, created_at
FROM workspace_invites WHERE login = ? ORDER BY created_at, workspace_id;

-- name: ListTeamMemberships :many
SELECT m.user_id, m.workspace_id, w.name AS workspace_name, m.role_id, r.name AS role_name, r.is_owner_role,
       COALESCE(po.allow, '[]') AS allow, COALESCE(po.deny, '[]') AS deny
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id
JOIN roles r ON r.id = m.role_id
LEFT JOIN permission_overwrites po
  ON po.resource_type = 'workspace'
 AND po.resource_id = m.workspace_id
 AND po.user_id = m.user_id
ORDER BY w.created_at, w.id;

-- name: ListTeamWorkspaces :many
SELECT id, name FROM workspaces ORDER BY created_at, id;

-- name: ListTeamRoles :many
SELECT id, workspace_id, name, is_owner_role FROM roles ORDER BY is_owner_role DESC, created_at, id;

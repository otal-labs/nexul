-- name: CreateWorkspace :exec
INSERT INTO workspaces (id, name, slug, mention_chip_template, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateWorkspace :execrows
UPDATE workspaces SET name = ?, slug = ?, mention_chip_template = ?, updated_at = ? WHERE id = ?;

-- name: GetWorkspace :one
SELECT id, name, created_at, updated_at, mention_chip_template, slug, decisions_check_enabled, auto_play_daily_cap, access_setup_finished, access_setup_by, access_setup_at, archived_at FROM workspaces WHERE id = ?;

-- name: GetWorkspaceBySlug :one
SELECT id, name, created_at, updated_at, mention_chip_template, slug, decisions_check_enabled, auto_play_daily_cap, access_setup_finished, access_setup_by, access_setup_at, archived_at FROM workspaces WHERE slug = ?;

-- name: ListWorkspacesForUser :many
-- An archived workspace is not one its members belong to until it is restored.
SELECT w.id, w.name, w.created_at, w.updated_at, w.mention_chip_template, w.slug, w.decisions_check_enabled, w.auto_play_daily_cap, w.access_setup_finished, w.access_setup_by, w.access_setup_at, w.archived_at
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.user_id = ? AND w.archived_at IS NULL
ORDER BY w.created_at, w.id;

-- name: ListArchivedWorkspaces :many
SELECT id, name, created_at, updated_at, mention_chip_template, slug, decisions_check_enabled, auto_play_daily_cap, access_setup_finished, access_setup_by, access_setup_at, archived_at
FROM workspaces WHERE archived_at IS NOT NULL ORDER BY archived_at DESC, id;

-- name: ArchiveWorkspace :execrows
UPDATE workspaces SET archived_at = ?, updated_at = ? WHERE id = ? AND archived_at IS NULL;

-- name: RestoreWorkspace :execrows
UPDATE workspaces SET archived_at = NULL, updated_at = ? WHERE id = ? AND archived_at IS NOT NULL;

-- name: AddWorkspaceMember :exec
INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES (?, ?, ?);

-- name: AddMemberRole :exec
INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES (?, ?, ?, ?);

-- name: DeleteMemberRole :exec
DELETE FROM member_roles WHERE workspace_id = ? AND user_id = ? AND role_id = ?;

-- name: ListMemberRoleIDs :many
SELECT mr.role_id FROM member_roles mr
JOIN roles r ON r.id = mr.role_id
WHERE mr.workspace_id = ? AND mr.user_id = ?
ORDER BY mr.created_at, r.is_owner_role DESC, mr.role_id;

-- name: ListWorkspaceMemberRoles :many
SELECT mr.user_id, mr.role_id FROM member_roles mr
JOIN roles r ON r.id = mr.role_id
WHERE mr.workspace_id = ?
ORDER BY mr.user_id, mr.created_at, r.is_owner_role DESC, mr.role_id;

-- name: ListMemberRoleGrants :many
SELECT m.user_id, r.id AS role_id, r.is_owner_role, r.permissions
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.archived_at IS NULL
LEFT JOIN member_roles mr ON mr.workspace_id = m.workspace_id AND mr.user_id = m.user_id
LEFT JOIN roles r ON r.id = mr.role_id
WHERE m.workspace_id = ? AND m.user_id = ?
ORDER BY mr.created_at, r.is_owner_role DESC, mr.role_id;

-- name: ListWorkspaceMembers :many
SELECT * FROM workspace_members WHERE workspace_id = ? ORDER BY created_at, user_id;

-- name: GetWorkspaceMember :one
SELECT m.* FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.archived_at IS NULL
WHERE m.workspace_id = ? AND m.user_id = ?;

-- name: ListWorkspaceIDsForUser :many
SELECT m.workspace_id FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.archived_at IS NULL
WHERE m.user_id = ? ORDER BY m.created_at, m.workspace_id;

-- name: DeleteMemberProjectAccess :exec
DELETE FROM permission_overwrites
WHERE resource_type = 'project' AND user_id = ?
  AND resource_id IN (SELECT id FROM projects WHERE workspace_id = ?);

-- name: ListProjectMemberIDs :many
SELECT m.user_id FROM workspace_members m
WHERE m.workspace_id = sqlc.arg(workspace_id)
  AND (EXISTS (SELECT 1 FROM member_roles mr JOIN roles r ON r.id = mr.role_id
               WHERE mr.workspace_id = m.workspace_id AND mr.user_id = m.user_id AND r.is_owner_role = 1)
    OR (NOT EXISTS (SELECT 1 FROM permission_overwrites po, json_each(po.deny) d
                    WHERE po.resource_type = 'project' AND po.resource_id = sqlc.arg(project_id) AND po.target = 'person'
                      AND po.user_id = m.user_id AND d.value = 'projects:open')
      AND (EXISTS (SELECT 1 FROM permission_overwrites po, json_each(po.allow) a
                   WHERE po.resource_type = 'project' AND po.resource_id = sqlc.arg(project_id) AND po.target = 'person'
                     AND po.user_id = m.user_id AND a.value = 'projects:open')
        OR EXISTS (SELECT 1 FROM member_roles mr
                   JOIN permission_overwrites pr ON pr.role_id = mr.role_id, json_each(pr.allow) a
                   WHERE mr.workspace_id = m.workspace_id AND mr.user_id = m.user_id
                     AND pr.resource_type = 'project' AND pr.resource_id = sqlc.arg(project_id) AND a.value = 'projects:open'))))
ORDER BY m.created_at, m.user_id;

-- name: ListProjectRoleIDs :many
SELECT po.role_id FROM permission_overwrites po, json_each(po.allow) a
WHERE po.resource_type = 'project' AND po.resource_id = sqlc.arg(project_id) AND po.target = 'role' AND a.value = 'projects:open'
ORDER BY po.created_at, po.role_id;

-- name: RemoveWorkspaceMember :execrows
DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?;

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
SELECT m.user_id, m.workspace_id, w.name AS workspace_name, r.id AS role_id, r.name AS role_name, r.is_owner_role
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id AND w.archived_at IS NULL
LEFT JOIN member_roles mr ON mr.workspace_id = m.workspace_id AND mr.user_id = m.user_id
LEFT JOIN roles r ON r.id = mr.role_id
ORDER BY w.created_at, w.id, m.user_id, mr.created_at, r.is_owner_role DESC, mr.role_id;

-- name: ListTeamWorkspaces :many
SELECT id, name FROM workspaces WHERE archived_at IS NULL ORDER BY created_at, id;

-- name: ListTeamRoles :many
SELECT id, workspace_id, name, is_owner_role FROM roles ORDER BY is_owner_role DESC, created_at, id;

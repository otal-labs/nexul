-- name: CreateWorkspace :exec
INSERT INTO workspaces (id, name, created_at, updated_at) VALUES (?, ?, ?, ?);

-- name: UpdateWorkspace :execrows
UPDATE workspaces SET name = ?, updated_at = ? WHERE id = ?;

-- name: GetWorkspace :one
SELECT id, name, created_at, updated_at FROM workspaces WHERE id = ?;

-- name: ListWorkspacesForUser :many
SELECT w.id, w.name, w.created_at, w.updated_at
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

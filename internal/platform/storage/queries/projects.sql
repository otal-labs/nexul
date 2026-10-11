-- name: CreateProject :exec
INSERT INTO projects (id, name, prefix, position, workspace_id, icon, setup_finished, setup_steps, setup_stack_id, setup_env_keys, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertProjectCreatorRoleRules :exec
INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at)
SELECT 'project', sqlc.arg(project_id), 'role', mr.role_id, '["projects:open"]', '[]', sqlc.arg(now), sqlc.arg(now)
FROM member_roles mr JOIN roles r ON r.id = mr.role_id
WHERE mr.user_id = sqlc.arg(user_id) AND mr.workspace_id = sqlc.arg(workspace_id) AND r.is_owner_role = 0;

-- name: SeedProjectStatus :exec
INSERT INTO statuses (id, project_id, name, position, kind, icon, created_at, updated_at) VALUES (?, ?, ?, ?, ?, '', ?, ?);

-- name: SeedProjectTicketType :exec
INSERT INTO ticket_types (id, project_id, name, position, color, body_template, created_at, updated_at) VALUES (?, ?, ?, ?, '', ?, ?, ?);

-- name: GetProject :one
SELECT * FROM projects WHERE id = ?;

-- name: ListProjectsByWorkspace :many
SELECT * FROM projects WHERE workspace_id = ? ORDER BY position, id;

-- name: UpdateProject :execrows
UPDATE projects SET name = ?, prefix = ?, icon = ?, tests_location = ?, updated_at = ? WHERE id = ?;

-- name: UpdateProjectSetup :execrows
UPDATE projects SET setup_finished = ?, setup_steps = ?, setup_stack_id = ?, setup_env_keys = ?, updated_at = ? WHERE id = ?;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE id = ?;

-- name: ReorderProjectPosition :exec
UPDATE projects SET position = ?, updated_at = ? WHERE id = ?;

-- name: AddProjectRepo :exec
INSERT INTO project_repos (project_id, owner, name, full_name, connector_id, role, added_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: RemoveProjectRepo :execrows
DELETE FROM project_repos WHERE owner = ? AND name = ?;

-- name: ListProjectRepos :many
SELECT owner, name, full_name, connector_id, role FROM project_repos WHERE project_id = ? ORDER BY name;

-- name: ListAllProjectRepos :many
SELECT owner, name, full_name, connector_id, role FROM project_repos ORDER BY full_name;

-- name: GetProjectRepoByOwnerAndName :one
SELECT project_id, owner, name, full_name, connector_id, role FROM project_repos WHERE owner = ? AND name = ?;

-- name: MoveTicketProject :execrows
UPDATE tickets SET project_id = ? WHERE id = ?;


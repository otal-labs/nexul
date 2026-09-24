-- name: CreateMemory :exec
INSERT INTO memories (id, workspace_id, project_id, kind, title, when_to_use, body, always_included, version, created_by, created_at, updated_by, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetMemory :one
SELECT * FROM memories WHERE id = ?;

-- name: ListMemoriesByWorkspace :many
SELECT * FROM memories WHERE workspace_id = ? ORDER BY project_id IS NOT NULL, project_id, created_at;

-- name: ListMemoriesByProject :many
-- The project's own memories plus its workspace's workspace-scoped ones, workspace-scoped first.
SELECT * FROM memories
WHERE project_id = ? OR (project_id IS NULL AND workspace_id = ?)
ORDER BY project_id IS NOT NULL, project_id, created_at;

-- name: ListWorkspaceScopedMemories :many
SELECT * FROM memories WHERE workspace_id = ? AND project_id IS NULL ORDER BY created_at;

-- name: UpdateMemory :execrows
UPDATE memories SET title = ?, when_to_use = ?, body = ?, always_included = ?, version = ?, updated_by = ?, updated_at = ? WHERE id = ?;

-- name: DeleteMemory :execrows
DELETE FROM memories WHERE id = ?;

-- name: InsertMemoryVersion :exec
INSERT INTO memory_versions (id, memory_id, version, title, when_to_use, body, always_included, author_id, author_via, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListMemoryVersions :many
SELECT id, memory_id, version, title, when_to_use, body, always_included, author_id, author_via, created_at
FROM memory_versions WHERE memory_id = ? ORDER BY version DESC;

-- name: GetMemoryVersion :one
SELECT id, memory_id, version, title, when_to_use, body, always_included, author_id, author_via, created_at
FROM memory_versions WHERE memory_id = ? AND version = ?;

-- name: GetMemoryByProjectKind :one
SELECT * FROM memories WHERE project_id = ? AND kind = ?;

-- name: GetInterviewTemplate :one
SELECT * FROM interview_templates WHERE workspace_id = ?;

-- name: UpsertInterviewTemplate :exec
INSERT INTO interview_templates (workspace_id, body, updated_by, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT (workspace_id) DO UPDATE SET body = excluded.body, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

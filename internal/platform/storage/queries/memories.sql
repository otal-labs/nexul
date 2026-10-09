-- name: CreateMemory :exec
INSERT INTO memories (id, workspace_id, project_id, kind, title, when_to_use, body, always_included, footer, version, created_by, created_at, updated_by, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetMemory :one
SELECT * FROM memories WHERE id = ?;

-- name: ListMemoriesByWorkspace :many
SELECT * FROM memories WHERE workspace_id = ? ORDER BY project_id, created_at;

-- name: ListMemoriesByProject :many
SELECT * FROM memories WHERE project_id = ? ORDER BY created_at;

-- name: ListMemoriesPage :many
SELECT * FROM memories WHERE project_id = ? ORDER BY created_at, id LIMIT ? OFFSET ?;

-- name: CountMemoriesByProject :one
SELECT COUNT(*) FROM memories WHERE project_id = ?;

-- name: UpdateMemory :execrows
UPDATE memories SET title = ?, when_to_use = ?, body = ?, always_included = ?, footer = ?, version = ?, updated_by = ?, updated_at = ? WHERE id = ?;

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

-- name: DeleteInterviewTemplate :exec
DELETE FROM interview_templates WHERE workspace_id = ?;

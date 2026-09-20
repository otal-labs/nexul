-- name: CreateDoc :exec
INSERT INTO docs (id, project_id, title, body, body_md, version, archived, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDoc :one
SELECT * FROM docs WHERE id = ?;

-- name: ListDocs :many
SELECT * FROM docs ORDER BY created_at;

-- name: ListDocsByProject :many
SELECT * FROM docs WHERE project_id = ? ORDER BY created_at;

-- name: UpdateDoc :execrows
UPDATE docs SET title = ?, body = ?, body_md = ?, version = ?, updated_at = ? WHERE id = ?;

-- name: CommitDocBody :execrows
UPDATE docs SET title = ?, body = ?, body_md = ?, version = ?, updated_at = ? WHERE id = ?;

-- name: SetDocArchived :execrows
UPDATE docs SET archived = ?, updated_at = ? WHERE id = ?;

-- name: DeleteDoc :execrows
DELETE FROM docs WHERE id = ?;

-- name: ListDocVersions :many
SELECT doc_id, version, title, body, name, author_id, created_at FROM doc_versions WHERE doc_id = ? ORDER BY version DESC;

-- name: GetDocVersion :one
SELECT doc_id, version, title, body, name, author_id, created_at FROM doc_versions WHERE doc_id = ? AND version = ?;

-- name: InsertDocVersion :exec
INSERT INTO doc_versions (doc_id, version, title, body, created_at) VALUES (?, ?, ?, ?, ?);

-- name: InsertNamedDocVersion :exec
INSERT INTO doc_versions (doc_id, version, title, body, name, author_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: BumpDocVersion :execrows
UPDATE docs SET version = ? WHERE id = ?;

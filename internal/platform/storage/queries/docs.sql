-- name: CreateDoc :exec
INSERT INTO docs (id, project_id, folder_id, title, body, body_md, version, archived, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDoc :one
SELECT * FROM docs WHERE id = ?;

-- name: ListDocs :many
SELECT * FROM docs ORDER BY created_at;

-- name: ListDocsByIDs :many
SELECT * FROM docs WHERE id IN (sqlc.slice('ids'));

-- name: ListDocsByProject :many
SELECT * FROM docs WHERE project_id = ? ORDER BY created_at;

-- name: ListDocScopes :many
SELECT d.id, COALESCE(d.project_id, '') AS project_id, COALESCE(p.workspace_id, '') AS workspace_id, d.folder_id
FROM docs d LEFT JOIN projects p ON p.id = d.project_id
WHERE d.id IN (SELECT value FROM json_each(sqlc.arg(ids)));

-- name: UpdateDoc :execrows
UPDATE docs SET title = ?, body = ?, body_md = ?, version = ?, updated_at = ? WHERE id = ?;

-- name: CommitDocBody :execrows
UPDATE docs SET title = ?, body = ?, body_md = ?, version = ?, updated_at = ? WHERE id = ?;

-- name: SetDocArchived :execrows
UPDATE docs SET archived = ?, updated_at = ? WHERE id = ?;

-- name: SetDocLocked :execrows
UPDATE docs SET locked = ? WHERE id = ?;

-- name: DeleteDoc :execrows
DELETE FROM docs WHERE id = ?;

-- name: ListDocVersions :many
SELECT doc_id, version, title, body, name, author_id, editor_ids, via, created_at FROM doc_versions WHERE doc_id = ? ORDER BY version DESC;

-- name: ListDocVersionSummaries :many
SELECT version, name, editor_ids, via, created_at FROM doc_versions WHERE doc_id = ? ORDER BY version DESC;

-- name: GetDocVersion :one
SELECT doc_id, version, title, body, name, author_id, editor_ids, via, created_at FROM doc_versions WHERE doc_id = ? AND version = ?;

-- name: InsertDocVersion :exec
INSERT INTO doc_versions (doc_id, version, title, body, author_id, editor_ids, via, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertNamedDocVersion :exec
INSERT INTO doc_versions (doc_id, version, title, body, name, author_id, editor_ids, via, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DocMatchesLatestVersion :one
SELECT EXISTS(
    SELECT 1 FROM doc_versions v
    WHERE v.doc_id = sqlc.arg(for_doc) AND v.title = sqlc.arg(want_title) AND v.body = sqlc.arg(want_body)
      AND v.version = (SELECT MAX(l.version) FROM doc_versions l WHERE l.doc_id = sqlc.arg(for_doc))
);

-- name: PruneDocVersions :exec
DELETE FROM doc_versions
WHERE doc_versions.doc_id = sqlc.arg(for_doc) AND doc_versions.name = ''
  AND doc_versions.version != (SELECT d.version FROM docs d WHERE d.id = sqlc.arg(for_doc))
  AND doc_versions.version NOT IN (
    SELECT k.version FROM doc_versions k WHERE k.doc_id = sqlc.arg(for_doc) AND k.name = '' ORDER BY k.version DESC LIMIT sqlc.arg(keep)
  );

-- name: BumpDocVersion :execrows
UPDATE docs SET version = ? WHERE id = ?;

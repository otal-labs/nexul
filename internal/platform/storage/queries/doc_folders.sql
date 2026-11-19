-- name: CreateDocFolder :exec
INSERT INTO doc_folders (id, project_id, parent_id, name, is_default, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetDocFolder :one
SELECT * FROM doc_folders WHERE id = ?;

-- name: ListDocFolderChain :many
WITH RECURSIVE up AS (
  SELECT f.id AS id, f.parent_id AS parent_id, 0 AS depth FROM doc_folders f WHERE f.id = ?
  UNION ALL
  SELECT f.id, f.parent_id, up.depth + 1 FROM up JOIN doc_folders f ON f.id = up.parent_id WHERE up.depth < 100
)
SELECT up.id, up.depth, f.project_id, p.workspace_id FROM up JOIN doc_folders f ON f.id = up.id JOIN projects p ON p.id = f.project_id
ORDER BY up.depth DESC;

-- name: ListDocFolderSubtree :many
WITH RECURSIVE down AS (
  SELECT CAST(sqlc.arg(id) AS TEXT) AS id
  UNION
  SELECT f.id FROM down JOIN doc_folders f ON f.parent_id = down.id
)
SELECT id FROM down;

-- name: DeleteDocFolderRules :exec
DELETE FROM permission_overwrites WHERE resource_type = 'folder' AND resource_id IN (SELECT value FROM json_each(sqlc.arg(ids)));

-- name: GetDefaultDocFolder :one
SELECT * FROM doc_folders WHERE project_id = ? AND is_default = 1;

-- name: ListDocFoldersByProject :many
SELECT * FROM doc_folders WHERE project_id = ? ORDER BY is_default DESC, created_at, id;

-- name: UpdateDocFolder :execrows
UPDATE doc_folders SET name = ?, parent_id = ?, updated_at = ? WHERE id = ?;

-- name: DeleteDocFolders :execrows
DELETE FROM doc_folders WHERE id IN (SELECT value FROM json_each(sqlc.arg(ids)));

-- name: MoveDocsToFolder :exec
UPDATE docs SET folder_id = sqlc.arg(to_folder_id) WHERE folder_id IN (SELECT value FROM json_each(sqlc.arg(from_folder_ids)));

-- name: SetDocFolder :execrows
UPDATE docs SET folder_id = ? WHERE id = ?;

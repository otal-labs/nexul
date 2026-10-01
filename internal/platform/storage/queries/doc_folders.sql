-- name: CreateDocFolder :exec
INSERT INTO doc_folders (id, project_id, name, is_default, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetDocFolder :one
SELECT * FROM doc_folders WHERE id = ?;

-- name: GetDefaultDocFolder :one
SELECT * FROM doc_folders WHERE project_id = ? AND is_default = 1;

-- name: ListDocFoldersByProject :many
SELECT * FROM doc_folders WHERE project_id = ? ORDER BY is_default DESC, created_at, id;

-- name: RenameDocFolder :execrows
UPDATE doc_folders SET name = ?, updated_at = ? WHERE id = ?;

-- name: DeleteDocFolder :execrows
DELETE FROM doc_folders WHERE id = ?;

-- name: MoveDocsToFolder :exec
UPDATE docs SET folder_id = sqlc.arg(to_folder_id) WHERE folder_id = sqlc.arg(from_folder_id);

-- name: SetDocFolder :execrows
UPDATE docs SET folder_id = ? WHERE id = ?;

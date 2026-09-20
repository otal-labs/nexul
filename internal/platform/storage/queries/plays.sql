-- name: CreatePlay :exec
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPlay :one
SELECT * FROM plays WHERE id = ?;

-- name: ListPlays :many
SELECT * FROM plays WHERE workspace_id = ? ORDER BY label;

-- name: UpdatePlay :execrows
UPDATE plays SET label = ?, description = ?, instructions = ?, enabled = ?, show_when_stage = ?, excluded_project_ids = ?, updated_at = ?
WHERE id = ?;

-- name: DeletePlay :execrows
DELETE FROM plays WHERE id = ?;

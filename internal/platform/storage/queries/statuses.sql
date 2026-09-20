-- name: CreateStatus :exec
INSERT INTO statuses (id, project_id, name, position, kind, icon, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetStatus :one
SELECT * FROM statuses WHERE id = ?;

-- name: ListStatusesByProject :many
SELECT * FROM statuses WHERE project_id = ?
    ORDER BY CASE kind WHEN 'backlog' THEN 0 WHEN 'progress' THEN 1 WHEN 'review' THEN 2 WHEN 'testing' THEN 3 ELSE 4 END, position, id;

-- name: UpdateStatus :execrows
UPDATE statuses SET name = ?, kind = ?, icon = ?, updated_at = ? WHERE id = ?;

-- name: DeleteStatus :execrows
DELETE FROM statuses WHERE id = ?;

-- name: ReorderStatusPosition :execrows
UPDATE statuses SET position = ?, updated_at = ? WHERE id = ? AND project_id = ?;

-- name: CountStatusTickets :one
SELECT COUNT(*) FROM tickets WHERE status = ?;

-- name: StatusExists :one
SELECT COUNT(*) FROM statuses WHERE id = ?;

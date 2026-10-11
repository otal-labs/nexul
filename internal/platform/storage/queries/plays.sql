-- name: CreatePlay :exec
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at, builtin_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPlay :one
SELECT * FROM plays WHERE id = ?;

-- name: ListPlays :many
SELECT * FROM plays WHERE workspace_id = ? ORDER BY label;

-- name: GetPlayByLabel :one
SELECT * FROM plays WHERE workspace_id = ? AND label = ? COLLATE NOCASE;

-- name: UpdatePlay :execrows
UPDATE plays SET label = ?, description = ?, instructions = ?, enabled = ?, show_when_stage = ?, excluded_project_ids = ?, updated_at = ?
WHERE id = ?;

-- name: DeletePlay :execrows
DELETE FROM plays WHERE id = ?;

-- name: ListAutomationNamesByPlayLabel :many
-- Equalities on status, not IN, so each automation reads only its active and pending versions through their partial indexes.
SELECT a.name FROM automations a
WHERE a.workspace_id = sqlc.arg(workspace_id) AND EXISTS (
    SELECT 1 FROM automation_versions v
    WHERE v.automation_id = a.id AND (v.status = 'active' OR v.status = 'pending')
      AND (instr(lower(CAST(v.code AS TEXT)), '"' || lower(CAST(sqlc.arg(label) AS TEXT)) || '"') > 0
        OR instr(lower(CAST(v.code AS TEXT)), '''' || lower(CAST(sqlc.arg(label) AS TEXT)) || '''') > 0
        OR instr(lower(CAST(v.code AS TEXT)), '`' || lower(CAST(sqlc.arg(label) AS TEXT)) || '`') > 0)
)
ORDER BY a.name, a.id;

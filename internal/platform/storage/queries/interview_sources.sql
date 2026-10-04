-- name: ListInterviewSources :many
SELECT * FROM interview_sources WHERE project_id = ? ORDER BY added_at, id;

-- name: GetInterviewSource :one
SELECT * FROM interview_sources WHERE id = ?;

-- name: CountInterviewSources :one
SELECT COUNT(*) FROM interview_sources WHERE project_id = ?;

-- name: InsertInterviewSource :exec
INSERT INTO interview_sources (id, workspace_id, project_id, kind, ref, label, body, stance, added_by, added_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateInterviewSource :execrows
UPDATE interview_sources SET stance = ?, label = ?, updated_at = ? WHERE id = ?;

-- name: DeleteInterviewSource :execrows
DELETE FROM interview_sources WHERE id = ?;

-- name: ListInterviewDrafts :many
SELECT * FROM interview_drafts WHERE project_id = ? ORDER BY drafted_at, id;

-- name: GetInterviewDraft :one
SELECT * FROM interview_drafts WHERE id = ?;

-- name: UpsertInterviewDraft :exec
INSERT INTO interview_drafts (id, workspace_id, project_id, question, selected, free_text, source_ids, where_line, trail_id, drafted_by, drafted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (project_id, question) DO UPDATE SET
    id = excluded.id, selected = excluded.selected, free_text = excluded.free_text, source_ids = excluded.source_ids,
    where_line = excluded.where_line, trail_id = excluded.trail_id, drafted_by = excluded.drafted_by, drafted_at = excluded.drafted_at;

-- name: DeleteInterviewDraft :execrows
DELETE FROM interview_drafts WHERE id = ?;

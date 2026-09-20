-- name: GetAutomationCursor :one
SELECT last_created_at, last_event_id FROM automation_cursors WHERE automation_id = ?;

-- name: SetAutomationCursor :exec
INSERT INTO automation_cursors (automation_id, last_created_at, last_event_id, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(automation_id) DO UPDATE SET last_created_at = excluded.last_created_at, last_event_id = excluded.last_event_id, updated_at = excluded.updated_at;

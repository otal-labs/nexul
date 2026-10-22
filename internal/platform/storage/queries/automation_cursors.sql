-- name: GetAutomationCursor :one
SELECT last_created_at, last_event_id FROM automation_cursors WHERE automation_id = ?;

-- name: SetAutomationCursor :exec
-- Only ever advances: a run report that lands after a skip must not rewind past it.
INSERT INTO automation_cursors (automation_id, last_created_at, last_event_id, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(automation_id) DO UPDATE SET last_created_at = excluded.last_created_at, last_event_id = excluded.last_event_id, updated_at = excluded.updated_at
WHERE (excluded.last_created_at, excluded.last_event_id) > (automation_cursors.last_created_at, automation_cursors.last_event_id);

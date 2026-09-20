-- name: CreateAutomationRun :exec
INSERT INTO automation_runs (id, automation_id, event_topic, event_id, outcome, error, started_at, finished_at, duration_ms, logs, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAutomationRun :one
SELECT * FROM automation_runs WHERE id = ?;

-- name: ListAutomationRunsByAutomation :many
SELECT * FROM automation_runs WHERE automation_id = ? ORDER BY started_at DESC, id DESC LIMIT ?;

-- name: DeleteAutomationRunsOlderThan :execrows
DELETE FROM automation_runs WHERE created_at < ?;

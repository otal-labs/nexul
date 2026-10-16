-- name: AddStackRequestMinute :exec
INSERT INTO stack_request_minutes (stack_id, minute, requests, errors_5xx) VALUES (?, ?, ?, ?)
ON CONFLICT (stack_id, minute) DO UPDATE SET
    requests = COALESCE(stack_request_minutes.requests, 0) + excluded.requests,
    errors_5xx = COALESCE(stack_request_minutes.errors_5xx, 0) + excluded.errors_5xx;

-- name: MarkStackRequestMinuteMissing :exec
INSERT OR IGNORE INTO stack_request_minutes (stack_id, minute) VALUES (?, ?);

-- name: ListStackRequestMinutes :many
SELECT * FROM stack_request_minutes WHERE stack_id = ? AND minute >= ? ORDER BY minute;

-- name: PruneStackRequestMinutes :exec
DELETE FROM stack_request_minutes WHERE minute < ?;

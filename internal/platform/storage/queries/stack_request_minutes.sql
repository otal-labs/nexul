-- name: AddStackRequestMinute :exec
INSERT INTO stack_request_minutes (stack_id, source, minute, requests, errors_5xx) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (stack_id, source, minute) DO UPDATE SET
    requests = COALESCE(stack_request_minutes.requests, 0) + excluded.requests,
    errors_5xx = COALESCE(stack_request_minutes.errors_5xx, 0) + excluded.errors_5xx;

-- name: SetStackRequestMinute :exec
INSERT INTO stack_request_minutes (stack_id, source, minute, requests, errors_5xx) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (stack_id, source, minute) DO UPDATE SET requests = excluded.requests, errors_5xx = excluded.errors_5xx;

-- name: MarkStackRequestMinuteMissing :exec
INSERT OR IGNORE INTO stack_request_minutes (stack_id, source, minute) VALUES (?, ?, ?);

-- A minute's count is the sum of its sources; counted_sources is 0 when none could count it, which is not zero requests.
-- name: ListStackRequestMinutes :many
SELECT minute, COUNT(requests) AS counted_sources, CAST(COALESCE(SUM(requests), 0) AS INTEGER) AS requests,
       CAST(COALESCE(SUM(errors_5xx), 0) AS INTEGER) AS errors_5xx
FROM stack_request_minutes WHERE stack_id = ? AND minute >= ? GROUP BY minute ORDER BY minute;

-- name: PruneStackRequestMinutes :exec
DELETE FROM stack_request_minutes WHERE minute < ?;

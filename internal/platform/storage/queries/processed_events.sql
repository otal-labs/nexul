-- name: RecordProcessedEvent :exec
INSERT OR IGNORE INTO processed_events (event_id, processed_at) VALUES (?, ?);

-- name: CountProcessedEvent :one
SELECT COUNT(*) FROM processed_events WHERE event_id = ?;

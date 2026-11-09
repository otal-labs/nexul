-- name: DeleteProcessedEventsBefore :execrows
DELETE FROM processed_events WHERE rowid IN (SELECT p.rowid FROM processed_events AS p WHERE p.processed_at < sqlc.arg(before) LIMIT sqlc.arg(max_rows));

-- name: DeleteOutboxBefore :execrows
-- Keeps every row an enabled automation has not yet read: its cursor's second is the floor, and rows in that second stay.
DELETE FROM outbox WHERE id IN (
    SELECT o.id FROM outbox AS o
    WHERE o.published = 1 AND o.created_at < sqlc.arg(before)
      AND o.created_at < COALESCE((SELECT MIN(c.last_created_at) FROM automation_cursors AS c JOIN automations AS a ON a.id = c.automation_id WHERE a.enabled = 1), 9223372036854775807)
    LIMIT sqlc.arg(max_rows));

-- name: DeleteDeployLogLinesBefore :execrows
DELETE FROM deploy_log_lines WHERE seq IN (SELECT l.seq FROM deploy_log_lines AS l WHERE l.ts < sqlc.arg(before) LIMIT sqlc.arg(max_rows));

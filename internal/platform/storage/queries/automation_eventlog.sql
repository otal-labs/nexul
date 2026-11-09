-- name: ListOutboxAfterCursor :many
SELECT id, topic, payload, created_at FROM outbox
WHERE topic IN (SELECT value FROM json_each(sqlc.arg(topics))) AND (created_at > sqlc.arg(cursor_created_at) OR (created_at = sqlc.arg(cursor_created_at) AND id > sqlc.arg(cursor_id)))
ORDER BY created_at, id LIMIT sqlc.arg('limit');

-- name: LatestOutboxPosition :one
SELECT id, created_at FROM outbox ORDER BY created_at DESC, id DESC LIMIT 1;

-- name: LatestSettledOutboxPosition :one
-- Rows older than the newest row's second are final: a later commit starts after that row's insert, so it cannot land below it.
SELECT id, created_at FROM outbox
WHERE created_at < (SELECT MAX(created_at) FROM outbox)
ORDER BY created_at DESC, id DESC LIMIT 1;

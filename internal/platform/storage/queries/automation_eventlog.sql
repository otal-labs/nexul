-- name: ListOutboxAfterCursor :many
SELECT id, topic, payload, created_at FROM outbox
WHERE topic IN (sqlc.slice('topics')) AND (created_at > sqlc.arg(cursor_created_at) OR (created_at = sqlc.arg(cursor_created_at) AND id > sqlc.arg(cursor_id)))
ORDER BY created_at, id LIMIT sqlc.arg('limit');

-- name: LatestOutboxPosition :one
SELECT id, created_at FROM outbox ORDER BY created_at DESC, id DESC LIMIT 1;

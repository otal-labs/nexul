-- name: ListDocWatchers :many
SELECT * FROM doc_watchers WHERE doc_id = ? AND watching = 1 ORDER BY created_at, user_id;

-- name: AddAutoDocWatcher :exec
-- DO NOTHING keeps an opt-out: someone who stopped watching is not re-added by their own edit.
INSERT INTO doc_watchers (doc_id, user_id, watching, source, created_at, updated_at)
VALUES (sqlc.arg(doc_id), sqlc.arg(user_id), 1, 'auto', sqlc.arg(at), sqlc.arg(at))
ON CONFLICT(doc_id, user_id) DO NOTHING;

-- name: SetDocWatching :exec
INSERT INTO doc_watchers (doc_id, user_id, watching, source, created_at, updated_at)
VALUES (sqlc.arg(doc_id), sqlc.arg(user_id), sqlc.arg(watching), 'manual', sqlc.arg(at), sqlc.arg(at))
ON CONFLICT(doc_id, user_id) DO UPDATE SET watching = excluded.watching, updated_at = excluded.updated_at;

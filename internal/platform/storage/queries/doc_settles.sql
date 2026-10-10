-- name: UpsertDocSettle :exec
INSERT INTO doc_settles (doc_id, due_at, actor_id, first) VALUES (?, ?, ?, ?)
ON CONFLICT(doc_id) DO UPDATE SET due_at = excluded.due_at, actor_id = excluded.actor_id;

-- name: NextDocSettleDue :one
SELECT MIN(due_at) FROM doc_settles;

-- name: ListDueDocSettles :many
SELECT s.doc_id, s.first, s.actor_id, d.project_id, d.title
FROM doc_settles s
JOIN docs d ON d.id = s.doc_id
WHERE s.due_at <= ? AND d.archived = 0
ORDER BY s.due_at, s.doc_id;

-- name: DeleteDueDocSettles :exec
DELETE FROM doc_settles WHERE due_at <= ?;

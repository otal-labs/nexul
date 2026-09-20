-- name: EnqueueOutbox :exec
INSERT INTO outbox (id, topic, payload) VALUES (?, ?, ?);

-- name: ListUnpublishedOutbox :many
SELECT id, topic, payload, created_at FROM outbox WHERE published = 0 ORDER BY created_at LIMIT ?;

-- name: MarkOutboxPublished :execrows
UPDATE outbox SET published = 1 WHERE id = ?;

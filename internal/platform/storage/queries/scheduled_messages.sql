-- name: ScheduledMessageInsert :exec
INSERT INTO scheduled_messages (id, conversation_id, author_id, body, starts_at, send_at, repeats, timezone, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ScheduledMessageGet :one
SELECT * FROM scheduled_messages WHERE id = ?;

-- name: ScheduledMessageUpdate :execrows
UPDATE scheduled_messages SET body = ?, starts_at = ?, send_at = ?, repeats = ?, timezone = ?, updated_at = ? WHERE id = ?;

-- name: ScheduledMessageDelete :execrows
DELETE FROM scheduled_messages WHERE id = ?;

-- name: ScheduledMessagesPage :many
SELECT * FROM scheduled_messages
WHERE conversation_id = ? AND author_id = ?
ORDER BY send_at, id
LIMIT ? OFFSET ?;

-- name: ScheduledMessagesCount :one
SELECT COUNT(*) FROM scheduled_messages WHERE conversation_id = ? AND author_id = ?;

-- name: ScheduledMessagesDue :many
SELECT * FROM scheduled_messages WHERE send_at <= ? ORDER BY send_at, id LIMIT ?;

-- name: ScheduledMessageNext :one
SELECT MIN(send_at) FROM scheduled_messages;

-- name: ScheduledMessageAdvance :execrows
UPDATE scheduled_messages SET send_at = sqlc.arg(next_send_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND send_at = sqlc.arg(send_at) AND body = sqlc.arg(body) AND repeats = sqlc.arg(repeats);

-- name: ScheduledMessageDeleteSent :execrows
DELETE FROM scheduled_messages WHERE id = sqlc.arg(id) AND send_at = sqlc.arg(send_at) AND body = sqlc.arg(body) AND repeats = sqlc.arg(repeats);

-- name: PutDeadLetter :exec
INSERT INTO dead_letters (id, topic, payload, error, attempts) VALUES (?, ?, ?, ?, ?);

-- name: GetDeadLetter :one
SELECT * FROM dead_letters WHERE id = ?;

-- name: ListDeadLetters :many
SELECT * FROM dead_letters ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: DeleteDeadLetter :execrows
DELETE FROM dead_letters WHERE id = ?;

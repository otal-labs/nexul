-- name: CountStackUnwatcher :one
SELECT COUNT(*) FROM stack_unwatchers WHERE stack_id = ? AND user_id = ?;

-- name: ListStackUnwatcherIDs :many
SELECT user_id FROM stack_unwatchers WHERE stack_id = ? ORDER BY user_id;

-- name: AddStackUnwatcher :exec
INSERT INTO stack_unwatchers (stack_id, user_id, created_at) VALUES (?, ?, ?)
ON CONFLICT(stack_id, user_id) DO NOTHING;

-- name: DeleteStackUnwatcher :exec
DELETE FROM stack_unwatchers WHERE stack_id = ? AND user_id = ?;

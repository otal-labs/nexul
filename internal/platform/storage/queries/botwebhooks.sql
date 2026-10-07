-- name: CreateBotwebhook :exec
INSERT INTO botwebhooks (id, conversation_id, name, avatar, token, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetBotwebhook :one
SELECT * FROM botwebhooks WHERE id = ?;

-- name: ListBotwebhooks :many
SELECT * FROM botwebhooks WHERE conversation_id = ? AND (deleted_at IS NOT NULL) = sqlc.arg(deleted) ORDER BY created_at, id;

-- name: UpdateBotwebhook :execrows
UPDATE botwebhooks SET name = ?, avatar = ?, token = ?, updated_at = ?, deleted_at = ?, deleted_by = ? WHERE id = ?;

-- name: CountBotwebhookPost :execrows
UPDATE botwebhooks SET post_count = post_count + 1, last_post_at = ? WHERE id = ? AND deleted_at IS NULL;

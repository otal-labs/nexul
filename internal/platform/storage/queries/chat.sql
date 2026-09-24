-- name: CreateConversation :exec
INSERT INTO conversations (id, workspace_id, kind, name, ticket_id, doc_id, project_id, parent_message_id, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertConversationParticipant :exec
INSERT INTO conversation_participants (conversation_id, user_id, created_at) VALUES (?, ?, ?)
    ON CONFLICT(conversation_id, user_id) DO NOTHING;

-- name: GetConversation :one
SELECT * FROM conversations WHERE id = ?;

-- name: GetChannelByName :one
SELECT * FROM conversations WHERE workspace_id = ? AND kind = 'channel' AND name = ?;

-- name: GetTicketThread :one
SELECT * FROM conversations WHERE ticket_id = ?;

-- name: GetDocThread :one
SELECT * FROM conversations WHERE doc_id = ?;

-- name: GetInterviewThread :one
SELECT * FROM conversations WHERE project_id = ?;

-- name: ListTicketIDsWithThreads :many
SELECT ticket_id FROM conversations WHERE ticket_id IN (sqlc.slice('ids'));

-- name: ListConversationsForUser :many
SELECT DISTINCT c.* FROM conversations c
LEFT JOIN conversation_participants p ON p.conversation_id = c.id AND p.user_id = ?
WHERE c.workspace_id = ? AND (c.kind IN ('channel', 'voice_channel', 'doc_thread') OR p.user_id IS NOT NULL)
ORDER BY c.created_at;

-- name: ListParticipantsByConversationIDs :many
SELECT conversation_id, user_id FROM conversation_participants WHERE conversation_id IN (sqlc.slice('ids'));

-- name: CreateMessage :exec
INSERT INTO messages (id, conversation_id, author_id, author_kind, body, mentions, attachment_id, edited_at, deleted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?);

-- name: SetAgentThread :execrows
UPDATE conversations SET agent_thread_id = ? WHERE id = ?;

-- name: SetAgentSyncedAt :execrows
UPDATE conversations SET agent_synced_at = ? WHERE id = ?;

-- name: GetMessage :one
SELECT * FROM messages WHERE id = ?;

-- name: ListMessages :many
SELECT * FROM messages WHERE conversation_id = ? ORDER BY created_at LIMIT ?;

-- name: ListMessagesSince :many
SELECT * FROM messages WHERE conversation_id = ? AND deleted_at IS NULL AND created_at > ? ORDER BY created_at;

-- name: UpdateMessage :execrows
UPDATE messages SET body = ?, mentions = ?, edited_at = ?, updated_at = ? WHERE id = ?;

-- name: DeleteMessage :execrows
UPDATE messages SET body = '', mentions = '[]', deleted_at = ?, updated_at = ? WHERE id = ?;

-- name: MarkRead :exec
INSERT INTO conversation_unread_state (user_id, conversation_id, last_read_at) VALUES (?, ?, ?)
    ON CONFLICT(user_id, conversation_id) DO UPDATE SET last_read_at = excluded.last_read_at;

-- name: UnreadCounts :many
SELECT c.id, c.kind, c.doc_id, COUNT(m.id) AS count FROM conversations c
LEFT JOIN conversation_participants p ON p.conversation_id = c.id AND p.user_id = sqlc.arg(user_id)
LEFT JOIN conversation_unread_state u ON u.conversation_id = c.id AND u.user_id = sqlc.arg(user_id)
LEFT JOIN messages m ON m.conversation_id = c.id AND m.deleted_at IS NULL AND m.author_id != sqlc.arg(user_id) AND m.created_at > COALESCE(u.last_read_at, 0)
WHERE c.workspace_id = sqlc.arg(workspace_id) AND (c.kind IN ('channel', 'voice_channel', 'doc_thread') OR p.user_id IS NOT NULL)
GROUP BY c.id;

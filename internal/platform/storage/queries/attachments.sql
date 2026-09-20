-- name: CreateAttachment :exec
INSERT INTO attachments (id, doc_id, ticket_id, conversation_id, memory_id, name, content_type, size, uploaded_by, created_at, data)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAttachment :one
SELECT * FROM attachments WHERE id = ?;

-- name: ListAttachmentsByOwner :many
SELECT id, doc_id, ticket_id, conversation_id, memory_id, name, content_type, size, uploaded_by, created_at
FROM attachments WHERE doc_id = ? OR ticket_id = ? OR conversation_id = ? OR memory_id = ? ORDER BY created_at, id;

-- name: DeleteAttachment :execrows
DELETE FROM attachments WHERE id = ?;

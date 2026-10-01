-- name: CreateNotificationIfAbsent :execrows
-- NOT EXISTS collapses repeats while an unread row exists in the same inbox, so rapid edits don't flood it.
INSERT INTO notifications (id, user_id, workspace_id, kind, subject_type, subject_id, subject_title, read, read_at, created_at)
SELECT sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(workspace_id), sqlc.arg(kind), sqlc.arg(subject_type), sqlc.arg(subject_id), sqlc.arg(subject_title), sqlc.arg(read), sqlc.narg(read_at), sqlc.arg(created_at)
WHERE NOT EXISTS (
  SELECT 1 FROM notifications
  WHERE user_id = sqlc.arg(user_id) AND workspace_id = sqlc.arg(workspace_id) AND kind = sqlc.arg(kind) AND subject_type = sqlc.arg(subject_type) AND subject_id = sqlc.arg(subject_id) AND read = 0
);

-- name: ListNotifications :many
-- An empty workspace_id lists every workspace, the unscoped inbox older clients still ask for.
-- A doc's folder is joined in at read time, never stored on the row, because the doc can move after it was sent.
SELECT sqlc.embed(n), COALESCE(f.id, '') AS folder_id, COALESCE(f.name, '') AS folder_name, COALESCE(f.is_default, 0) AS folder_is_default
FROM notifications n
LEFT JOIN docs d ON n.subject_type = 'doc' AND d.id = n.subject_id
LEFT JOIN doc_folders f ON f.id = d.folder_id
WHERE n.user_id = sqlc.arg(user_id) AND (sqlc.arg(workspace_id) = '' OR n.workspace_id = sqlc.arg(workspace_id))
ORDER BY n.created_at DESC LIMIT sqlc.arg(limit);

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE user_id = sqlc.arg(user_id) AND read = 0 AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id));

-- name: MarkNotificationRead :execrows
-- Reading an already read row keeps its first read time, so marking it again never extends its retention.
UPDATE notifications SET read = 1, read_at = COALESCE(read_at, sqlc.arg(read_at)) WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read = 1, read_at = sqlc.arg(read_at)
WHERE user_id = sqlc.arg(user_id) AND read = 0 AND (sqlc.arg(workspace_id) = '' OR workspace_id = sqlc.arg(workspace_id));

-- name: DeleteNotificationsCreatedBefore :execrows
DELETE FROM notifications WHERE created_at < ?;

-- name: DeleteNotificationsReadBefore :execrows
DELETE FROM notifications WHERE read_at IS NOT NULL AND read_at < ?;

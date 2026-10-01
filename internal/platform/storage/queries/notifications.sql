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
-- The subject's project is joined in too, so a notice about a project its reader can no longer open is left out at read time.
SELECT sqlc.embed(n), COALESCE(f.id, '') AS folder_id, COALESCE(f.name, '') AS folder_name, COALESCE(f.is_default, 0) AS folder_is_default,
       CAST(COALESCE(t.project_id, d.project_id, m.project_id, '') AS TEXT) AS project_id
FROM notifications n
LEFT JOIN docs d ON n.subject_type = 'doc' AND d.id = n.subject_id
LEFT JOIN doc_folders f ON f.id = d.folder_id
LEFT JOIN tickets t ON n.subject_type = 'ticket' AND t.id = n.subject_id
LEFT JOIN memories m ON n.subject_type = 'memory' AND m.id = n.subject_id
WHERE n.user_id = sqlc.arg(user_id) AND (sqlc.arg(workspace_id) = '' OR n.workspace_id = sqlc.arg(workspace_id))
ORDER BY n.created_at DESC LIMIT sqlc.arg(limit);

-- name: CountUnreadNotificationsByProject :many
SELECT n.workspace_id, CAST(COALESCE(t.project_id, d.project_id, m.project_id, '') AS TEXT) AS project_id, COUNT(*) AS unread
FROM notifications n
LEFT JOIN docs d ON n.subject_type = 'doc' AND d.id = n.subject_id
LEFT JOIN tickets t ON n.subject_type = 'ticket' AND t.id = n.subject_id
LEFT JOIN memories m ON n.subject_type = 'memory' AND m.id = n.subject_id
WHERE n.user_id = sqlc.arg(user_id) AND n.read = 0 AND (sqlc.arg(workspace_id) = '' OR n.workspace_id = sqlc.arg(workspace_id))
GROUP BY n.workspace_id, 2;

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

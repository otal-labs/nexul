-- name: CreateNotificationIfAbsent :execrows
-- NOT EXISTS collapses repeats while an unread row exists, so rapid edits don't flood the inbox.
INSERT INTO notifications (id, user_id, kind, subject_type, subject_id, subject_title, read, created_at)
SELECT sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(subject_type), sqlc.arg(subject_id), sqlc.arg(subject_title), sqlc.arg(read), sqlc.arg(created_at)
WHERE NOT EXISTS (
  SELECT 1 FROM notifications
  WHERE user_id = sqlc.arg(user_id) AND kind = sqlc.arg(kind) AND subject_type = sqlc.arg(subject_type) AND subject_id = sqlc.arg(subject_id) AND read = 0
);

-- name: ListNotifications :many
SELECT * FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ?;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read = 0;

-- name: MarkNotificationRead :execrows
UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read = 1 WHERE user_id = ? AND read = 0;

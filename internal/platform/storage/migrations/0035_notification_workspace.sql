-- A notification belongs to its subject's workspace, so each workspace's inbox and unread count show only their own.
ALTER TABLE notifications ADD COLUMN workspace_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_notifications_user_workspace ON notifications(user_id, workspace_id, created_at DESC);

-- A notification records when it was read, so retention can count a read one's days from then (ADR 0100).
ALTER TABLE notifications ADD COLUMN read_at INTEGER;
-- A row read before this column has no known read time, so it counts from creation, the earliest it could have been.
UPDATE notifications SET read_at = created_at WHERE read = 1 AND read_at IS NULL;
-- Serve the daily retention deletes, which match on the read time or the creation time across every inbox.
CREATE INDEX IF NOT EXISTS idx_notifications_read_at ON notifications(read_at) WHERE read_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);

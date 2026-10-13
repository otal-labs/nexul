-- Who finished a workspace's one-time access setup and when, which the Access page reads back (ADR 0148); empty for
-- a workspace that started set up.
ALTER TABLE workspaces ADD COLUMN access_setup_by TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ADD COLUMN access_setup_at INTEGER NOT NULL DEFAULT 0;

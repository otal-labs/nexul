ALTER TABLE tickets RENAME COLUMN assignee TO developer;
ALTER TABLE tickets ADD COLUMN tester TEXT NOT NULL DEFAULT '';
ALTER TABLE tickets ADD COLUMN reporter_kind TEXT NOT NULL DEFAULT 'user';
ALTER TABLE tickets ADD COLUMN reporter_login TEXT NOT NULL DEFAULT '';
ALTER TABLE tickets ADD COLUMN reporter_automation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tickets ADD COLUMN reporter_automation_name TEXT NOT NULL DEFAULT '';

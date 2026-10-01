-- The instance layer of every template (ADR 0103): a row exists only once someone edits it; none means the code default.
CREATE TABLE instance_templates (
    kind       TEXT NOT NULL,
    key        TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL,
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (kind, key)
);

-- A seeded play keeps a stable key through renames, so a clone finds the same play in another workspace.
ALTER TABLE plays ADD COLUMN builtin_key TEXT NOT NULL DEFAULT '';
UPDATE plays SET builtin_key = 'fix-with-ai' WHERE rowid IN (
    SELECT MIN(rowid) FROM plays WHERE label = 'Fix with AI' AND type = 'ticket' GROUP BY workspace_id);
UPDATE plays SET builtin_key = 'to-tickets-via-ai' WHERE rowid IN (
    SELECT MIN(rowid) FROM plays WHERE label = 'To tickets via AI' AND type = 'doc' GROUP BY workspace_id);
UPDATE plays SET builtin_key = 'interview' WHERE rowid IN (
    SELECT MIN(rowid) FROM plays WHERE label = 'Interview' AND type = 'interview' GROUP BY workspace_id);
UPDATE plays SET builtin_key = 'test-with-ai' WHERE rowid IN (
    SELECT MIN(rowid) FROM plays WHERE label = 'Test with AI' AND type = 'ticket' GROUP BY workspace_id);
CREATE UNIQUE INDEX idx_plays_builtin_key ON plays(workspace_id, builtin_key) WHERE builtin_key != '';

-- An empty mention chip template follows the instance's; a workspace still on the old default never chose its own.
UPDATE workspaces SET mention_chip_template = '' WHERE mention_chip_template = '{ticket.Ticket} {ticket.Status}';

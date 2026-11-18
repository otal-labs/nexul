-- What Nexul writes back to GitHub issues (ADR 0200). The ticket's own view of its fields when it was last in step with
-- its issue, so a change GitHub itself made is never written back.
ALTER TABLE github_issue_links ADD COLUMN last_ticket_hash TEXT NOT NULL DEFAULT '';

-- A ticket whose fields wait to be written to its issue, or, with create, to a new issue in owner/repo. Every change
-- moves due_at (unix milliseconds) to the end of a quiet spell, so a burst of edits makes one write. tried_at is when
-- a create was first sent, so a retry looks for the issue it may already have made before making another.
CREATE TABLE IF NOT EXISTS github_issue_pushes (
    ticket_id TEXT PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
    owner     TEXT NOT NULL,
    repo      TEXT NOT NULL,
    create_issue INTEGER NOT NULL DEFAULT 0,
    due_at    INTEGER NOT NULL,
    tried_at  INTEGER,
    attempts  INTEGER NOT NULL DEFAULT 0
);
-- Serves DueIssuePushes and NextIssuePushDue.
CREATE INDEX IF NOT EXISTS idx_github_issue_pushes_due ON github_issue_pushes(due_at, ticket_id);

-- A person's ticket-thread message waiting to be mirrored to its issue as a comment, edited there, or deleted there.
CREATE TABLE IF NOT EXISTS github_comment_pushes (
    message_id TEXT PRIMARY KEY,
    ticket_id  TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    due_at     INTEGER NOT NULL,
    tried_at   INTEGER,
    attempts   INTEGER NOT NULL DEFAULT 0
);
-- Serves DueCommentPushes and NextCommentPushDue.
CREATE INDEX IF NOT EXISTS idx_github_comment_pushes_due ON github_comment_pushes(due_at, message_id);

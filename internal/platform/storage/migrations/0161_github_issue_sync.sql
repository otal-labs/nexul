-- A repository's GitHub issues sync to its project's tickets (ADR 0200). The sync goes with the repository's
-- attachment, so detaching it or moving it to another project turns the sync off for good.
CREATE TABLE IF NOT EXISTS github_issue_sync (
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    owner         TEXT NOT NULL,
    repo          TEXT NOT NULL,
    label         TEXT NOT NULL DEFAULT 'nexul',
    mode          TEXT NOT NULL DEFAULT 'label' CHECK (mode IN ('label', 'all')),
    enabled       INTEGER NOT NULL DEFAULT 0,
    last_swept_at INTEGER,
    UNIQUE (owner, repo),
    FOREIGN KEY (owner, repo) REFERENCES project_repos(owner, name) ON DELETE CASCADE
);
-- Serves ListIssueSyncsByProject: project_get and the project's repository rows.
CREATE INDEX IF NOT EXISTS idx_github_issue_sync_project ON github_issue_sync(project_id);

-- One GitHub issue per ticket; a deleted ticket takes its link, a deleted or transferred issue leaves the ticket.
CREATE TABLE IF NOT EXISTS github_issue_links (
    owner              TEXT NOT NULL,
    repo               TEXT NOT NULL,
    issue_number       INTEGER NOT NULL,
    issue_id           INTEGER NOT NULL,
    ticket_id          TEXT NOT NULL UNIQUE REFERENCES tickets(id) ON DELETE CASCADE,
    sync_paused        INTEGER NOT NULL DEFAULT 0,
    last_inbound_hash  TEXT NOT NULL DEFAULT '',
    last_outbound_hash TEXT NOT NULL DEFAULT '',
    issue_updated_at   INTEGER NOT NULL,
    linked_at          INTEGER NOT NULL,
    PRIMARY KEY (owner, repo, issue_number)
);

-- A GitHub comment and the ticket-thread message it became, so an edit or delete on GitHub follows.
CREATE TABLE IF NOT EXISTS github_comment_links (
    github_comment_id INTEGER PRIMARY KEY,
    message_id        TEXT NOT NULL UNIQUE,
    ticket_id         TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE
);
-- Serves DeleteIssueCommentLinksByTicket when an issue's link goes.
CREATE INDEX IF NOT EXISTS idx_github_comment_links_ticket ON github_comment_links(ticket_id);

-- A project's GitHub issues become its tickets through a label (ADR 0200). One sync per attached repository, which
-- goes when the repository leaves the project; last_swept_at is NULL until the first sweep links the open issues.
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
-- Serves GitHubIssueSyncsOfProject: a project's syncs on project_get and its settings.
CREATE INDEX IF NOT EXISTS idx_github_issue_sync_project ON github_issue_sync(project_id, owner, repo);

-- One linked issue per ticket; a paused link keeps its ticket while the issue lacks the label.
CREATE TABLE IF NOT EXISTS github_issue_links (
    owner              TEXT NOT NULL,
    repo               TEXT NOT NULL,
    issue_number       INTEGER NOT NULL,
    issue_id           INTEGER NOT NULL,
    ticket_id          TEXT NOT NULL UNIQUE REFERENCES tickets(id) ON DELETE CASCADE,
    sync_paused        INTEGER NOT NULL DEFAULT 0,
    last_inbound_hash  TEXT NOT NULL DEFAULT '',
    last_outbound_hash TEXT NOT NULL DEFAULT '',
    issue_updated_at   INTEGER NOT NULL DEFAULT 0,
    linked_at          INTEGER NOT NULL,
    PRIMARY KEY (owner, repo, issue_number)
);

-- A GitHub comment and the ticket-thread message it became, so an edit or delete on GitHub reaches the message.
CREATE TABLE IF NOT EXISTS github_comment_links (
    github_comment_id INTEGER PRIMARY KEY,
    message_id        TEXT NOT NULL UNIQUE,
    ticket_id         TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE
);

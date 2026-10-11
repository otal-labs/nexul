-- A pending invite gives its one role and nothing else, so deleting the role deletes the invite (ADR 0148).
CREATE TABLE workspace_invites_new (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    login        TEXT NOT NULL,
    role_id      TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    invited_by   TEXT NOT NULL REFERENCES users(id),
    created_at   INTEGER NOT NULL,
    PRIMARY KEY (workspace_id, login)
);
INSERT INTO workspace_invites_new (workspace_id, login, role_id, invited_by, created_at)
SELECT workspace_id, login, role_id, invited_by, created_at FROM workspace_invites
WHERE role_id IN (SELECT id FROM roles);
DROP TABLE workspace_invites;
ALTER TABLE workspace_invites_new RENAME TO workspace_invites;
CREATE INDEX idx_workspace_invites_login ON workspace_invites(login);

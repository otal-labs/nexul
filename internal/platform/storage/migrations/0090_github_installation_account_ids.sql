-- An installation's assignment is keyed by GitHub's numeric account id (ADR 0144, amended): a login can be renamed
-- and then taken by another account. Rows from 0088 carry only the login; the first read as the App records their id,
-- since a migration cannot ask GitHub. gone_at marks an account whose installation the App no longer lists.
CREATE TABLE github_installation_workspaces_new (
    account_id    INTEGER,
    account_login TEXT NOT NULL,
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_at   INTEGER NOT NULL,
    gone_at       INTEGER
);
INSERT INTO github_installation_workspaces_new (account_login, workspace_id, assigned_at)
SELECT account_login, workspace_id, assigned_at FROM github_installation_workspaces;
DROP TABLE github_installation_workspaces;
ALTER TABLE github_installation_workspaces_new RENAME TO github_installation_workspaces;

-- One row per account and workspace, whether the account is known by id or, until its id is recorded, by login.
CREATE UNIQUE INDEX IF NOT EXISTS idx_github_installation_workspaces_account
    ON github_installation_workspaces(account_id, workspace_id) WHERE account_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_github_installation_workspaces_login
    ON github_installation_workspaces(account_login, workspace_id) WHERE account_id IS NULL;
-- Serves ListGitHubAssignedAccountsIn and ListGitHubInstallationAssignmentsIn: what one workspace's lists read.
CREATE INDEX IF NOT EXISTS idx_github_installation_workspaces_workspace
    ON github_installation_workspaces(workspace_id, account_id);

-- An install link's state, stored only as a hash: it names the workspace and the person who asked for the link, and
-- the claim deletes it, so a link works once.
CREATE TABLE IF NOT EXISTS github_install_states (
    state_hash   TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL,
    expires_at   INTEGER NOT NULL
);

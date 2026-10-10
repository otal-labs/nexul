-- The private key the instance signs as its GitHub App with (ADR 0144), encrypted at rest like client_secret; empty
-- until the owner pastes one, and Nexul reads GitHub as the connected account meanwhile.
ALTER TABLE connector_app_config ADD COLUMN private_key TEXT NOT NULL DEFAULT '';

-- Which workspaces see an installation's repositories (ADR 0144). Keyed by the account the App is installed on,
-- lowercase: an App has one installation per account, and a project's repositories record that account as owner.
CREATE TABLE IF NOT EXISTS github_installation_workspaces (
    account_login TEXT NOT NULL,
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    assigned_at   INTEGER NOT NULL,
    PRIMARY KEY (account_login, workspace_id)
);
-- Serves ListGitHubInstallationAccountsIn: the accounts one workspace's repository list reads.
CREATE INDEX IF NOT EXISTS idx_github_installation_workspaces_workspace
    ON github_installation_workspaces(workspace_id, account_login);

-- Each workspace keeps seeing the accounts its projects already attach repositories from; an installation no
-- project uses stays unassigned until someone holding connectors:write assigns it.
INSERT OR IGNORE INTO github_installation_workspaces (account_login, workspace_id, assigned_at)
SELECT DISTINCT lower(r.owner), p.workspace_id, CAST(strftime('%s', 'now') AS INTEGER)
FROM project_repos r
JOIN projects p ON p.id = r.project_id
WHERE r.connector_id = 'github';

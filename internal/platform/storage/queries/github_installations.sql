-- name: ListGitHubInstallationWorkspaces :many
SELECT a.account_login, w.id AS workspace_id, w.name AS workspace_name
FROM github_installation_workspaces a
JOIN workspaces w ON w.id = a.workspace_id
ORDER BY a.account_login, w.name, w.id;

-- name: ListGitHubInstallationAccountsIn :many
SELECT DISTINCT account_login FROM github_installation_workspaces
WHERE workspace_id IN (SELECT value FROM json_each(?))
ORDER BY account_login;

-- name: AssignGitHubInstallation :exec
INSERT INTO github_installation_workspaces (account_login, workspace_id, assigned_at)
VALUES (?, ?, ?)
ON CONFLICT(account_login, workspace_id) DO NOTHING;

-- name: UnassignGitHubInstallation :exec
DELETE FROM github_installation_workspaces WHERE account_login = ? AND workspace_id = ?;

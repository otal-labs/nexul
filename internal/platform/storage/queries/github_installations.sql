-- name: ListGitHubInstallationAssignments :many
SELECT a.account_id, a.account_login, a.workspace_id, w.name AS workspace_name, a.gone_at
FROM github_installation_workspaces a
JOIN workspaces w ON w.id = a.workspace_id
ORDER BY a.account_login, w.name, w.id;

-- name: ListGitHubInstallationAssignmentsIn :many
SELECT a.account_id, a.account_login, a.workspace_id, w.name AS workspace_name, a.gone_at,
    EXISTS (
        SELECT 1 FROM project_repos r JOIN projects p ON p.id = r.project_id
        WHERE p.workspace_id = a.workspace_id AND r.connector_id = 'github' AND lower(r.owner) = a.account_login
    ) AS attached
FROM github_installation_workspaces a
JOIN workspaces w ON w.id = a.workspace_id
WHERE a.workspace_id IN (SELECT value FROM json_each(?))
ORDER BY a.account_login, w.name, w.id;

-- name: ListGitHubAssignedAccountsIn :many
SELECT DISTINCT account_id FROM github_installation_workspaces
WHERE workspace_id IN (SELECT value FROM json_each(?)) AND account_id IS NOT NULL AND gone_at IS NULL
ORDER BY account_id;

-- name: CountGitHubUnresolvedAssignments :one
SELECT COUNT(*) FROM github_installation_workspaces WHERE account_id IS NULL AND gone_at IS NULL;

-- name: AssignGitHubInstallation :execrows
INSERT INTO github_installation_workspaces (account_id, account_login, workspace_id, assigned_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (account_id, workspace_id) WHERE account_id IS NOT NULL
DO UPDATE SET gone_at = NULL, account_login = excluded.account_login, assigned_at = excluded.assigned_at
WHERE github_installation_workspaces.gone_at IS NOT NULL;

-- name: UnassignGitHubInstallation :execrows
DELETE FROM github_installation_workspaces
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (account_id = sqlc.arg(account_id) OR (account_id IS NULL AND account_login = sqlc.arg(account_login)));

-- name: ResolveGitHubInstallationAccount :exec
UPDATE OR IGNORE github_installation_workspaces SET account_id = ?
WHERE account_id IS NULL AND account_login = ?;

-- name: DropGitHubUnresolvedDuplicates :exec
DELETE FROM github_installation_workspaces WHERE account_id IS NULL AND account_login = ?;

-- name: RenameGitHubInstallationAccount :exec
UPDATE github_installation_workspaces SET account_login = ? WHERE account_id = ?;

-- name: MarkGitHubInstallationGone :exec
UPDATE github_installation_workspaces SET gone_at = sqlc.arg(gone_at)
WHERE gone_at IS NULL AND workspace_id = sqlc.arg(workspace_id)
  AND (account_id = sqlc.arg(account_id) OR (account_id IS NULL AND account_login = sqlc.arg(account_login)));

-- name: DropGoneGitHubInstallation :exec
DELETE FROM github_installation_workspaces WHERE account_id = ? AND gone_at IS NOT NULL;

-- name: ListGitHubAssignedAccounts :many
SELECT DISTINCT account_id FROM github_installation_workspaces
WHERE account_id IS NOT NULL AND gone_at IS NULL
ORDER BY account_id;

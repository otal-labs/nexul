package repository

import "context"

// InstallationStore records which workspaces see each installation, keyed by its account's lowercase login (ADR 0144).
type InstallationStore interface {
	// ListInstallationWorkspaces maps every assigned account to its workspaces.
	ListInstallationWorkspaces(ctx context.Context) (map[string][]InstallationWorkspace, error)
	// InstallationAccountsIn lists the accounts assigned to any of workspaceIDs.
	InstallationAccountsIn(ctx context.Context, workspaceIDs []string) ([]string, error)
	// AssignInstallation is a no-op when the account is already assigned there.
	AssignInstallation(ctx context.Context, account, workspaceID string) error
	// UnassignInstallation is a no-op when the account was not assigned there.
	UnassignInstallation(ctx context.Context, account, workspaceID string) error
}

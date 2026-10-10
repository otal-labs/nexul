package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/repository"
)

var _ repository.InstallationStore = (*GitHubInstallationsRepo)(nil)

// GitHubInstallationsRepo stores which workspaces see each GitHub App installation's repositories.
type GitHubInstallationsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// ListInstallationWorkspaces maps every assigned account to its workspaces, by name.
func (r *GitHubInstallationsRepo) ListInstallationWorkspaces(ctx context.Context) (map[string][]repository.InstallationWorkspace, error) {
	rows, err := r.q.ListGitHubInstallationWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation workspaces: %w", err)
	}
	out := make(map[string][]repository.InstallationWorkspace, len(rows))
	for _, row := range rows {
		out[row.AccountLogin] = append(out[row.AccountLogin], repository.InstallationWorkspace{ID: row.WorkspaceID, Name: row.WorkspaceName})
	}
	return out, nil
}

// InstallationAccountsIn lists the accounts assigned to any of workspaceIDs; none matches nothing.
func (r *GitHubInstallationsRepo) InstallationAccountsIn(ctx context.Context, workspaceIDs []string) ([]string, error) {
	accounts, err := r.q.ListGitHubInstallationAccountsIn(ctx, idsJSON(workspaceIDs))
	if err != nil {
		return nil, fmt.Errorf("list installation accounts: %w", err)
	}
	return accounts, nil
}

// AssignInstallation records that workspaceID sees account's repositories.
func (r *GitHubInstallationsRepo) AssignInstallation(ctx context.Context, account, workspaceID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AssignGitHubInstallation(ctx, sqlcgen.AssignGitHubInstallationParams{
			AccountLogin: strings.ToLower(account), WorkspaceID: workspaceID, AssignedAt: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("assign installation %s to %s: %w", account, workspaceID, classifyWriteErr(err))
		}
		return nil
	})
}

// UnassignInstallation stops workspaceID seeing account's repositories.
func (r *GitHubInstallationsRepo) UnassignInstallation(ctx context.Context, account, workspaceID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UnassignGitHubInstallation(ctx, sqlcgen.UnassignGitHubInstallationParams{
			AccountLogin: strings.ToLower(account), WorkspaceID: workspaceID,
		})
		if err != nil {
			return fmt.Errorf("unassign installation %s from %s: %w", account, workspaceID, err)
		}
		return nil
	})
}

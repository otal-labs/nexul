package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/repository"
)

var (
	_ repository.InstallationStore = (*GitHubInstallationsRepo)(nil)
	_ repository.InstallStateStore = (*GitHubInstallationsRepo)(nil)
)

// GitHubInstallationsRepo stores which workspaces see each GitHub App installation, and the install links' states.
type GitHubInstallationsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func toAssignment(accountID sql.NullInt64, login, workspaceID, workspaceName string, goneAt sql.NullInt64) repository.Assignment {
	return repository.Assignment{
		AccountID: accountID.Int64, AccountLogin: login, WorkspaceID: workspaceID, WorkspaceName: workspaceName, Gone: goneAt.Valid,
	}
}

// ListAssignments lists every assignment with its workspace's name.
func (r *GitHubInstallationsRepo) ListAssignments(ctx context.Context) ([]repository.Assignment, error) {
	rows, err := r.q.ListGitHubInstallationAssignments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation assignments: %w", err)
	}
	out := make([]repository.Assignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAssignment(row.AccountID, row.AccountLogin, row.WorkspaceID, row.WorkspaceName, row.GoneAt))
	}
	return out, nil
}

// AssignmentsIn lists the assignments to any of workspaceIDs; none matches nothing.
func (r *GitHubInstallationsRepo) AssignmentsIn(ctx context.Context, workspaceIDs []string) ([]repository.Assignment, error) {
	rows, err := r.q.ListGitHubInstallationAssignmentsIn(ctx, idsJSON(workspaceIDs))
	if err != nil {
		return nil, fmt.Errorf("list installation assignments: %w", err)
	}
	out := make([]repository.Assignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAssignment(row.AccountID, row.AccountLogin, row.WorkspaceID, row.WorkspaceName, row.GoneAt))
	}
	return out, nil
}

// AssignedAccountsIn lists the accounts assigned, and not gone, to any of workspaceIDs; none matches nothing.
func (r *GitHubInstallationsRepo) AssignedAccountsIn(ctx context.Context, workspaceIDs []string) ([]int64, error) {
	ids, err := r.q.ListGitHubAssignedAccountsIn(ctx, idsJSON(workspaceIDs))
	if err != nil {
		return nil, fmt.Errorf("list installation accounts: %w", err)
	}
	return accountIDs(ids), nil
}

// AssignedAccounts lists the accounts assigned, and not gone, to any workspace.
func (r *GitHubInstallationsRepo) AssignedAccounts(ctx context.Context) ([]int64, error) {
	ids, err := r.q.ListGitHubAssignedAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation accounts: %w", err)
	}
	return accountIDs(ids), nil
}

func accountIDs(ids []sql.NullInt64) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Int64)
	}
	return out
}

// HasUnresolvedAssignments reports an assignment from before account ids were kept.
func (r *GitHubInstallationsRepo) HasUnresolvedAssignments(ctx context.Context) (bool, error) {
	n, err := r.q.CountGitHubUnresolvedAssignments(ctx)
	if err != nil {
		return false, fmt.Errorf("count unresolved installation assignments: %w", err)
	}
	return n > 0, nil
}

// AssignInstallation records that a.WorkspaceID sees a.AccountID's repositories, reviving a gone assignment.
func (r *GitHubInstallationsRepo) AssignInstallation(ctx context.Context, a repository.Assignment, events ...eventbus.OutboxEvent) (bool, error) {
	var changed bool
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).AssignGitHubInstallation(ctx, sqlcgen.AssignGitHubInstallationParams{
			AccountID: sql.NullInt64{Int64: a.AccountID, Valid: true}, AccountLogin: strings.ToLower(a.AccountLogin),
			WorkspaceID: a.WorkspaceID, AssignedAt: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("assign installation %s to %s: %w", a.AccountLogin, a.WorkspaceID, classifyWriteErr(err))
		}
		changed = n > 0
		if !changed {
			return nil
		}
		return enqueueWorkspaceOutbox(ctx, tx, events)
	})
	return changed, err
}

// UnassignInstallation stops a.WorkspaceID seeing the account's repositories.
func (r *GitHubInstallationsRepo) UnassignInstallation(ctx context.Context, a repository.Assignment, events ...eventbus.OutboxEvent) (bool, error) {
	var changed bool
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UnassignGitHubInstallation(ctx, sqlcgen.UnassignGitHubInstallationParams{
			WorkspaceID: a.WorkspaceID, AccountID: sql.NullInt64{Int64: a.AccountID, Valid: true}, AccountLogin: strings.ToLower(a.AccountLogin),
		})
		if err != nil {
			return fmt.Errorf("unassign installation %s from %s: %w", a.AccountLogin, a.WorkspaceID, err)
		}
		changed = n > 0
		if !changed {
			return nil
		}
		return enqueueWorkspaceOutbox(ctx, tx, events)
	})
	return changed, err
}

// SyncAccounts records resolved ids and new logins, marks gone assignments, and drops those of a reinstalled account.
func (r *GitHubInstallationsRepo) SyncAccounts(ctx context.Context, s repository.AccountSync, events ...eventbus.OutboxEvent) error {
	now := time.Now().Unix()
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for login, id := range s.Resolved {
			if err := q.ResolveGitHubInstallationAccount(ctx, sqlcgen.ResolveGitHubInstallationAccountParams{AccountID: sql.NullInt64{Int64: id, Valid: true}, AccountLogin: login}); err != nil {
				return fmt.Errorf("record the account id of %s: %w", login, err)
			}
			// A workspace already holding the account by id keeps that row; the login-only one is a duplicate.
			if err := q.DropGitHubUnresolvedDuplicates(ctx, login); err != nil {
				return fmt.Errorf("drop duplicate assignments of %s: %w", login, err)
			}
		}
		for id, login := range s.Renamed {
			if err := q.RenameGitHubInstallationAccount(ctx, sqlcgen.RenameGitHubInstallationAccountParams{AccountLogin: login, AccountID: sql.NullInt64{Int64: id, Valid: true}}); err != nil {
				return fmt.Errorf("rename installation account %d: %w", id, err)
			}
		}
		for _, a := range s.Gone {
			err := q.MarkGitHubInstallationGone(ctx, sqlcgen.MarkGitHubInstallationGoneParams{
				GoneAt: sql.NullInt64{Int64: now, Valid: true}, WorkspaceID: a.WorkspaceID,
				AccountID: sql.NullInt64{Int64: a.AccountID, Valid: true}, AccountLogin: a.AccountLogin,
			})
			if err != nil {
				return fmt.Errorf("mark installation %s gone: %w", a.AccountLogin, err)
			}
		}
		for _, id := range s.Reinstalled {
			if err := q.DropGoneGitHubInstallation(ctx, sql.NullInt64{Int64: id, Valid: true}); err != nil {
				return fmt.Errorf("drop gone assignments of %d: %w", id, err)
			}
		}
		return enqueueWorkspaceOutbox(ctx, tx, events)
	})
}

// SaveInstallState stores an install link's state hash, clearing expired ones.
func (r *GitHubInstallationsRepo) SaveInstallState(ctx context.Context, stateHash string, st repository.InstallState, now time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeleteExpiredGitHubInstallStates(ctx, now.Unix()); err != nil {
			return fmt.Errorf("clear expired install states: %w", err)
		}
		err := q.SaveGitHubInstallState(ctx, sqlcgen.SaveGitHubInstallStateParams{
			StateHash: stateHash, WorkspaceID: st.WorkspaceID, UserID: st.UserID, ExpiresAt: st.ExpiresAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save install state: %w", classifyWriteErr(err))
		}
		return nil
	})
}

// ConsumeInstallState deletes the state and returns what it stood for, ErrNotFound when unknown or already used.
func (r *GitHubInstallationsRepo) ConsumeInstallState(ctx context.Context, stateHash string) (repository.InstallState, error) {
	var st repository.InstallState
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		row, err := r.q.WithTx(tx).ConsumeGitHubInstallState(ctx, stateHash)
		if errors.Is(err, sql.ErrNoRows) {
			return apperrs.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("consume install state: %w", err)
		}
		st = repository.InstallState{WorkspaceID: row.WorkspaceID, UserID: row.UserID, ExpiresAt: time.Unix(row.ExpiresAt, 0)}
		return nil
	})
	return st, err
}

package repository

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// AssignmentReader reads which workspaces see each installation, keyed by the account's GitHub id (ADR 0144).
type AssignmentReader interface {
	// ListAssignments lists every assignment, gone ones included.
	ListAssignments(ctx context.Context) ([]Assignment, error)
	// AssignmentsIn lists the assignments to any of workspaceIDs, gone ones included.
	AssignmentsIn(ctx context.Context, workspaceIDs []string) ([]Assignment, error)
	// AssignedAccountsIn lists the ids of the accounts assigned, and not gone, to any of workspaceIDs.
	AssignedAccountsIn(ctx context.Context, workspaceIDs []string) ([]int64, error)
	// AssignedAccounts lists the ids of the accounts assigned, and not gone, to any workspace.
	AssignedAccounts(ctx context.Context) ([]int64, error)
	// HasUnresolvedAssignments reports an assignment still waiting for its account's id.
	HasUnresolvedAssignments(ctx context.Context) (bool, error)
}

// AssignmentWriter changes the assignments, writing events in the same transaction only when something changed.
type AssignmentWriter interface {
	// AssignInstallation reports false, writing nothing, when the account is already assigned there.
	AssignInstallation(ctx context.Context, a Assignment, events ...eventbus.OutboxEvent) (bool, error)
	// UnassignInstallation reports false, writing nothing, when the account was not assigned there.
	UnassignInstallation(ctx context.Context, a Assignment, events ...eventbus.OutboxEvent) (bool, error)
	// SyncAccounts applies what reading the App's installations found.
	SyncAccounts(ctx context.Context, s AccountSync, events ...eventbus.OutboxEvent) error
}

// InstallationStore records which workspaces see each installation.
type InstallationStore interface {
	AssignmentReader
	AssignmentWriter
}

// InstallStateStore keeps each install link's state, by hash, until it is claimed or expires.
type InstallStateStore interface {
	SaveInstallState(ctx context.Context, stateHash string, st InstallState, now time.Time) error
	// ConsumeInstallState deletes and returns the state, ErrNotFound when unknown or already used.
	ConsumeInstallState(ctx context.Context, stateHash string) (InstallState, error)
}

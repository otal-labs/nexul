package runner

import (
	"context"
	"time"
)

// Repo persists runners together with their enrollment codes and credentials.
type Repo interface {
	RunnerStore
	CredentialStore
}

// RunnerStore persists the runner rows.
type RunnerStore interface {
	Create(ctx context.Context, r *Runner) error
	GetByID(ctx context.Context, id string) (*Runner, error)
	// GetByName returns the runner enrolled under name, or apperrs.ErrNotFound.
	GetByName(ctx context.Context, name string) (*Runner, error)
	// GetByComputer returns the personal runner that reaches computerID, or apperrs.ErrNotFound.
	GetByComputer(ctx context.Context, computerID string) (*Runner, error)
	// ListByOwner returns the personal runners userID enrolled.
	ListByOwner(ctx context.Context, userID string) ([]*Runner, error)
	// List returns the deploy runners; a personal runner is never listed (ADR 0146).
	List(ctx context.Context) ([]*Runner, error)
	// ListOn returns the deploy runners on the machines machineIDs names, in one statement however many there are.
	ListOn(ctx context.Context, machineIDs []string) ([]*Runner, error)
	Heartbeat(ctx context.Context, id string, at time.Time) error
	SetConnected(ctx context.Context, id string, connected bool) error
	// SetVersion records the runner's stamped build version, reported on each connect.
	SetVersion(ctx context.Context, id string, version string) error
}

// CredentialStore persists enrollment codes and runner credentials, both by hash only.
type CredentialStore interface {
	// CreateEnrollment stores a code and prunes every expired one.
	CreateEnrollment(ctx context.Context, e *EnrollmentCode) error
	// DeleteComputerEnrollments drops every unused code that would enroll computerID's runner.
	DeleteComputerEnrollments(ctx context.Context, computerID string) error
	// DeleteOwnerEnrollments drops every unused code that would enroll a personal runner for userID.
	DeleteOwnerEnrollments(ctx context.Context, userID string) error
	// ListEnrollments returns the unexpired codes userID owns as a personal runner's and every deploy runner's code
	// with a maker, oldest first.
	ListEnrollments(ctx context.Context, userID string, now time.Time) ([]*EnrollmentCode, error)
	// GetEnrollmentByID returns the unexpired code with id, or apperrs.ErrNotFound.
	GetEnrollmentByID(ctx context.Context, id string, now time.Time) (*EnrollmentCode, error)
	// RevokeEnrollment drops an unused code; one already used or revoked is apperrs.ErrNotFound.
	RevokeEnrollment(ctx context.Context, id string) error
	// GetEnrollment returns the unexpired code with codeHash, or apperrs.ErrNotFound.
	GetEnrollment(ctx context.Context, codeHash string, now time.Time) (*EnrollmentCode, error)
	// Enroll consumes the code, creates the runner and stores its credential in one transaction, and with created the
	// machine and its first rule too; a code already used or expired is apperrs.ErrNotFound, a runner or machine name
	// already taken is apperrs.ErrConflict.
	Enroll(ctx context.Context, codeHash string, r *Runner, credentialHash string, now time.Time, created *CreatedMachine) error
	// GetCredential returns the credential with hash, revoked ones included, or apperrs.ErrNotFound.
	GetCredential(ctx context.Context, hash string) (*Credential, error)
	// Remove revokes the runner's credentials and deletes its row; an unknown id is apperrs.ErrNotFound.
	Remove(ctx context.Context, id string, at time.Time) error
}

// UpgradeRepo persists instance_upgrades, a separate aggregate from Repo's runner rows.
type UpgradeRepo interface {
	Create(ctx context.Context, u *Upgrade) error
	GetByID(ctx context.Context, id string) (*Upgrade, error)
	// Latest returns the most recently created record, or apperrs.ErrNotFound when none exists.
	Latest(ctx context.Context) (*Upgrade, error)
	// ListUnresolved returns every pending/started record, oldest first, for boot resolution.
	ListUnresolved(ctx context.Context) ([]*Upgrade, error)
	// SetStatus updates status and error together and stamps updated_at; unknown id is apperrs.ErrNotFound.
	SetStatus(ctx context.Context, id, status, errMsg string) error
}

// TeardownRepo keeps the stack teardowns waiting for a machine's runner, so a server restart does not forget them.
type TeardownRepo interface {
	// Save keeps a teardown for machine; saving a slug already waiting replaces its strategy.
	Save(ctx context.Context, machine string, f Frame) error
	// List returns machine's waiting teardowns, oldest first.
	List(ctx context.Context, machine string) ([]Frame, error)
	// Delete drops the teardown for machine's slug; one that is not waiting is not an error.
	Delete(ctx context.Context, machine, slug string) error
}

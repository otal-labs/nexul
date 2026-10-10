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
	// GetEnrollment returns the unexpired code with codeHash, or apperrs.ErrNotFound.
	GetEnrollment(ctx context.Context, codeHash string, now time.Time) (*EnrollmentCode, error)
	// Enroll consumes the code, creates the runner and stores its credential in one transaction; a code already
	// used or expired is apperrs.ErrNotFound, a runner name already taken is apperrs.ErrConflict.
	Enroll(ctx context.Context, codeHash string, r *Runner, credentialHash string, now time.Time) error
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

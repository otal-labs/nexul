package runner

import (
	"context"
	"time"
)

type Repo interface {
	Create(ctx context.Context, r *Runner) error
	GetByID(ctx context.Context, id string) (*Runner, error)
	List(ctx context.Context) ([]*Runner, error)
	Heartbeat(ctx context.Context, id string, at time.Time) error
	SetConnected(ctx context.Context, id string, connected bool) error
	// SetVersion records the runner's stamped build version, reported on
	// each connect.
	SetVersion(ctx context.Context, id string, version string) error
	// SetMachine links a runner to the machine it reported on connect (issue 05).
	SetMachine(ctx context.Context, id string, machineID string) error
	Delete(ctx context.Context, id string) error
	// Secret returns the shared runner-connection secret, generated and persisted on first use.
	Secret(ctx context.Context) (string, error)
	// SetSecret overwrites the runner-connection secret, e.g. seeding it from NEXUL_RUNNER_SECRET at boot.
	SetSecret(ctx context.Context, secret string) error
}

// UpgradeRepo persists instance_upgrades (instance-upgrade spec), a separate aggregate from Repo's runner rows.
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

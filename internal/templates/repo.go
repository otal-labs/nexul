package templates

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Repo persists the edited instance templates; implemented in internal/platform/storage.
type Repo interface {
	// Get returns ErrNotFound for a template nobody has edited.
	Get(ctx context.Context, kind, key string) (*Record, error)
	List(ctx context.Context) ([]*Record, error)
	Save(ctx context.Context, r *Record, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, kind, key string, evts ...eventbus.OutboxEvent) error
}

// Gate checks an instance-level action against every workspace the caller belongs to (ADR 0088).
type Gate interface {
	RequireAnywhere(ctx context.Context, action permissions.Action) error
}

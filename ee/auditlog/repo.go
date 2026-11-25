package auditlog

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo stores the instance's one audit log setting; implemented in internal/platform/storage.
type Repo interface {
	// Get returns the zero Settings when the instance has none.
	Get(ctx context.Context) (Settings, error)
	Set(ctx context.Context, s Settings, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error
}

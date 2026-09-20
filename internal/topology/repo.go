package topology

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type Repo interface {
	Get(ctx context.Context, environment string) (*Canvas, error)
	// Save persists the canvas and enqueues its outbox events in the same transaction, so nothing is lost on a crash.
	Save(ctx context.Context, environment string, c *Canvas, evts ...eventbus.OutboxEvent) error
}

package license

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo stores the instance's one key as the text it was pasted as; implemented in internal/platform/storage.
type Repo interface {
	// Key returns the stored key text, empty when the instance holds none.
	Key(ctx context.Context) (string, error)
	SetKey(ctx context.Context, key, setBy string, at time.Time, evts ...eventbus.OutboxEvent) error
	DeleteKey(ctx context.Context, evts ...eventbus.OutboxEvent) error
}

package branding

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo stores the instance's one branding record; implemented in internal/platform/storage.
type Repo interface {
	// Load returns the saved branding, and false when the instance has none.
	Load(ctx context.Context) (Record, bool, error)
	// Save replaces the name, accent and themes, keeping the logo.
	Save(ctx context.Context, r Record, evts ...eventbus.OutboxEvent) error
	// Logo returns the stored image bytes, or ErrNotFound when the instance holds none.
	Logo(ctx context.Context) (contentType string, data []byte, err error)
	SetLogo(ctx context.Context, contentType string, data []byte, version, by string, at time.Time, evts ...eventbus.OutboxEvent) error
	ClearLogo(ctx context.Context, by string, at time.Time, evts ...eventbus.OutboxEvent) error
	// Delete removes everything saved, the logo included.
	Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error
}

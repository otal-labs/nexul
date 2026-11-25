package auditlog

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo stores the instance's one audit log setting; implemented in internal/platform/storage.
type Repo interface {
	// Get returns the zero Settings when the instance has none.
	Get(ctx context.Context) (Settings, error)
	Set(ctx context.Context, s Settings, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error
}

// Queue is the stream's deliveries, one per audit row written while the stream was on (ADR 0190); implemented in
// internal/platform/storage, where the audit row's own write inserts each one.
type Queue interface {
	// Due returns up to limit pending rows due by now, oldest audit row first.
	Due(ctx context.Context, limit int, now time.Time) ([]Pending, error)
	MarkDelivered(ctx context.Context, auditID string, at time.Time) error
	// MarkFailed keeps the row pending until next, with the failure's reason.
	MarkFailed(ctx context.Context, auditID string, attempts int, next time.Time, reason string) error
	MarkDead(ctx context.Context, auditID string, attempts int, reason string) error
	// RetryDead returns every dead row to pending, due at now, and says how many.
	RetryDead(ctx context.Context, now time.Time) (int, error)
	Stats(ctx context.Context) (Stats, error)
}

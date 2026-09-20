// Package outbox implements the transactional outbox.
package outbox

import (
	"context"
	"time"
)

// Entry is one row of the outbox table awaiting relay.
type Entry struct {
	ID        string
	Topic     string
	Payload   []byte
	CreatedAt time.Time
}

// Storer exposes the unpublished rows and the publish mark.
type Storer interface {
	// Unpublished returns up to limit rows with published = 0, oldest first.
	Unpublished(ctx context.Context, limit int) ([]Entry, error)
	// MarkPublished flags the row as relayed. Idempotent.
	MarkPublished(ctx context.Context, id string) error
}

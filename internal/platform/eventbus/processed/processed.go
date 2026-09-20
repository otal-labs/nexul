// Package processed holds the consumer-side idempotency store.
package processed

import "context"

// Storer records event IDs that have been successfully processed.
type Storer interface {
	// Seen reports whether eventID has already been processed.
	Seen(ctx context.Context, eventID string) (bool, error)
	// Record marks eventID as processed. It is idempotent.
	Record(ctx context.Context, eventID string) error
}

// Scope namespaces a Storer under consumer so different subscribers of a topic never share dedupe records.
func Scope(store Storer, consumer string) Storer {
	return &scoped{store: store, prefix: consumer + ":"}
}

type scoped struct {
	store  Storer
	prefix string
}

func (s *scoped) Seen(ctx context.Context, eventID string) (bool, error) {
	return s.store.Seen(ctx, s.prefix+eventID)
}

func (s *scoped) Record(ctx context.Context, eventID string) error {
	return s.store.Record(ctx, s.prefix+eventID)
}

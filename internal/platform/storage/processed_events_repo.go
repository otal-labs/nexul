package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

type ProcessedEventsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Record satisfies processed.Storer directly (no adapter layer): it marks eventID as processed, idempotently.
func (r *ProcessedEventsRepo) Record(ctx context.Context, eventID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).RecordProcessedEvent(ctx, sqlcgen.RecordProcessedEventParams{
			EventID: eventID, ProcessedAt: time.Now().UTC().Unix(),
		})
		if err != nil {
			return fmt.Errorf("mark event %s processed: %w", eventID, err)
		}
		return nil
	})
}

// Seen satisfies processed.Storer directly.
func (r *ProcessedEventsRepo) Seen(ctx context.Context, eventID string) (bool, error) {
	n, err := r.q.CountProcessedEvent(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("check event %s processed: %w", eventID, err)
	}
	return n > 0, nil
}

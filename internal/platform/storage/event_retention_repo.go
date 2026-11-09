package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ integrations.EventRetentionStore = (*EventRetentionRepo)(nil)

// EventRetentionRepo deletes the dedupe records, published outbox rows and deploy log lines past their retention.
type EventRetentionRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// DeleteProcessedBefore deletes at most limit dedupe records processed before before, in one serialized write.
func (r *EventRetentionRepo) DeleteProcessedBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	return r.delete(ctx, "processed events", before, func(q *sqlcgen.Queries) (int64, error) {
		return q.DeleteProcessedEventsBefore(ctx, sqlcgen.DeleteProcessedEventsBeforeParams{Before: before.Unix(), MaxRows: int64(limit)})
	})
}

// DeleteOutboxBefore deletes at most limit published rows older than before that no enabled automation still has to read.
func (r *EventRetentionRepo) DeleteOutboxBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	return r.delete(ctx, "outbox rows", before, func(q *sqlcgen.Queries) (int64, error) {
		return q.DeleteOutboxBefore(ctx, sqlcgen.DeleteOutboxBeforeParams{Before: before.Unix(), MaxRows: int64(limit)})
	})
}

// DeleteDeployLogBefore deletes at most limit deploy log lines (stamped in milliseconds) before before, in one serialized write.
func (r *EventRetentionRepo) DeleteDeployLogBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	return r.delete(ctx, "deploy log lines", before, func(q *sqlcgen.Queries) (int64, error) {
		return q.DeleteDeployLogLinesBefore(ctx, sqlcgen.DeleteDeployLogLinesBeforeParams{Before: before.UnixMilli(), MaxRows: int64(limit)})
	})
}

func (r *EventRetentionRepo) delete(ctx context.Context, what string, before time.Time, run func(*sqlcgen.Queries) (int64, error)) (int64, error) {
	var n int64
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var err error
		n, err = run(r.q.WithTx(tx))
		if err != nil {
			return fmt.Errorf("delete %s before %s: %w", what, before, err)
		}
		return nil
	})
	return n, err
}

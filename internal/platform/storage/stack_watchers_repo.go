package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.StackWatchRepo = (*StackWatchersRepo)(nil)

// StackWatchersRepo backs deploy.StackWatchRepo: the stack_unwatchers table, one row per person who opted out.
type StackWatchersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *StackWatchersRepo) Unwatching(ctx context.Context, stackID, userID string) (bool, error) {
	n, err := r.q.CountStackUnwatcher(ctx, sqlcgen.CountStackUnwatcherParams{StackID: stackID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("read opt-out of %s on stack %s: %w", userID, stackID, err)
	}
	return n > 0, nil
}

func (r *StackWatchersRepo) ListUnwatcherIDs(ctx context.Context, stackID string) ([]string, error) {
	ids, err := r.q.ListStackUnwatcherIDs(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("list opt-outs of stack %s: %w", stackID, err)
	}
	return ids, nil
}

func (r *StackWatchersRepo) SetWatching(ctx context.Context, stackID, userID string, watching bool, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if watching {
			return q.DeleteStackUnwatcher(ctx, sqlcgen.DeleteStackUnwatcherParams{StackID: stackID, UserID: userID})
		}
		return q.AddStackUnwatcher(ctx, sqlcgen.AddStackUnwatcherParams{StackID: stackID, UserID: userID, CreatedAt: at.Unix()})
	})
}

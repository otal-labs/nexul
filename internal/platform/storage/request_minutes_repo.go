package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.RequestRepo = (*RequestMinutesRepo)(nil)

// RequestMinutesRepo backs deploy.RequestRepo: the stack_request_minutes table.
type RequestMinutesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *RequestMinutesRepo) Add(ctx context.Context, adds []deploy.RequestAdd) error {
	if len(adds) == 0 {
		return nil
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, a := range adds {
			err := q.AddStackRequestMinute(ctx, sqlcgen.AddStackRequestMinuteParams{
				StackID: a.StackID, Minute: a.Minute.Unix(),
				Requests: sql.NullInt64{Int64: a.Requests, Valid: true}, Errors5xx: sql.NullInt64{Int64: a.Errors, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("add request minute of stack %s: %w", a.StackID, classifyWriteErr(err))
			}
		}
		return nil
	})
}

func (r *RequestMinutesRepo) MarkMissing(ctx context.Context, stackIDs []string, minute time.Time) error {
	if len(stackIDs) == 0 {
		return nil
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, id := range stackIDs {
			err := q.MarkStackRequestMinuteMissing(ctx, sqlcgen.MarkStackRequestMinuteMissingParams{StackID: id, Minute: minute.Unix()})
			if err != nil {
				return fmt.Errorf("mark request minute of stack %s missing: %w", id, classifyWriteErr(err))
			}
		}
		return nil
	})
}

func (r *RequestMinutesRepo) Since(ctx context.Context, stackID string, from time.Time) ([]deploy.RequestMinute, error) {
	rows, err := r.q.ListStackRequestMinutes(ctx, sqlcgen.ListStackRequestMinutesParams{StackID: stackID, Minute: from.Unix()})
	if err != nil {
		return nil, fmt.Errorf("list request minutes of stack %s: %w", stackID, err)
	}
	out := make([]deploy.RequestMinute, 0, len(rows))
	for _, row := range rows {
		m := deploy.RequestMinute{Minute: time.Unix(row.Minute, 0).UTC()}
		if row.Requests.Valid {
			m.Requests, m.Errors = &row.Requests.Int64, &row.Errors5xx.Int64
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *RequestMinutesRepo) Prune(ctx context.Context, before time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).PruneStackRequestMinutes(ctx, before.Unix()); err != nil {
			return fmt.Errorf("prune request minutes: %w", err)
		}
		return nil
	})
}

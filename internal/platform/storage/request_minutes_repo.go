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
	return r.write(ctx, "add", adds, func(q *sqlcgen.Queries, a deploy.RequestAdd) error {
		return q.AddStackRequestMinute(ctx, sqlcgen.AddStackRequestMinuteParams{
			StackID: a.StackID, Source: string(deploy.SourceProxy), Minute: a.Minute.Unix(),
			Requests: sql.NullInt64{Int64: a.Requests, Valid: true}, Errors5xx: sql.NullInt64{Int64: a.Errors, Valid: true},
		})
	})
}

func (r *RequestMinutesRepo) Set(ctx context.Context, sets []deploy.RequestAdd) error {
	return r.write(ctx, "set", sets, func(q *sqlcgen.Queries, a deploy.RequestAdd) error {
		return q.SetStackRequestMinute(ctx, sqlcgen.SetStackRequestMinuteParams{
			StackID: a.StackID, Source: string(deploy.SourceTunnel), Minute: a.Minute.Unix(),
			Requests: sql.NullInt64{Int64: a.Requests, Valid: true}, Errors5xx: sql.NullInt64{Int64: a.Errors, Valid: true},
		})
	})
}

func (r *RequestMinutesRepo) write(ctx context.Context, verb string, entries []deploy.RequestAdd, one func(*sqlcgen.Queries, deploy.RequestAdd) error) error {
	if len(entries) == 0 {
		return nil
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, a := range entries {
			if err := one(q, a); err != nil {
				return fmt.Errorf("%s request minute of stack %s: %w", verb, a.StackID, classifyWriteErr(err))
			}
		}
		return nil
	})
}

func (r *RequestMinutesRepo) MarkMissing(ctx context.Context, source deploy.RequestSource, stackIDs []string, minute time.Time) error {
	if len(stackIDs) == 0 {
		return nil
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, id := range stackIDs {
			err := q.MarkStackRequestMinuteMissing(ctx, sqlcgen.MarkStackRequestMinuteMissingParams{StackID: id, Source: string(source), Minute: minute.Unix()})
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
		if row.CountedSources > 0 {
			m.Requests, m.Errors = &row.Requests, &row.Errors5xx
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

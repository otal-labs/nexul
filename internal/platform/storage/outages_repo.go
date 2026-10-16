package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.OutageRepo = (*OutagesRepo)(nil)

// OutagesRepo backs deploy.OutageRepo: the service_outages table.
type OutagesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *OutagesRepo) Open(ctx context.Context, o *deploy.Outage, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateOutage(ctx, sqlcgen.CreateOutageParams{
			ID: o.ID, StackID: o.StackID, Service: o.Service, Container: o.Container, StartedAt: o.StartedAt.Unix(),
			ExitCode: int64(o.ExitCode), Oom: boolToInt(o.OOM), RestartCount: int64(o.RestartCount),
		})
		if err != nil {
			return fmt.Errorf("insert outage %s: %w", o.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *OutagesRepo) Update(ctx context.Context, o *deploy.Outage) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpdateOutage(ctx, sqlcgen.UpdateOutageParams{
			Container: o.Container, ExitCode: int64(o.ExitCode), Oom: boolToInt(o.OOM), RestartCount: int64(o.RestartCount), ID: o.ID,
		})
		if err != nil {
			return fmt.Errorf("update outage %s: %w", o.ID, err)
		}
		return nil
	})
}

func (r *OutagesRepo) Close(ctx context.Context, id string, endedAt time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).CloseOutage(ctx, sqlcgen.CloseOutageParams{EndedAt: sql.NullInt64{Int64: endedAt.Unix(), Valid: true}, ID: id})
		if err != nil {
			return fmt.Errorf("close outage %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("close outage %s: %w", id, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *OutagesRepo) GetOpen(ctx context.Context, stackID, service string) (*deploy.Outage, error) {
	row, err := r.q.GetOpenOutage(ctx, sqlcgen.GetOpenOutageParams{StackID: stackID, Service: service})
	if err != nil {
		return nil, fmt.Errorf("get open outage of %s: %w", service, notFoundIfNoRows(err))
	}
	return toOutage(row), nil
}

func (r *OutagesRepo) ListOpenByStack(ctx context.Context, stackID string) ([]*deploy.Outage, error) {
	rows, err := r.q.ListOpenOutagesByStack(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("list open outages of stack %s: %w", stackID, err)
	}
	out := make([]*deploy.Outage, 0, len(rows))
	for _, row := range rows {
		out = append(out, toOutage(row))
	}
	return out, nil
}

func (r *OutagesRepo) LatestEnd(ctx context.Context, stackID, service string) (time.Time, error) {
	end, err := r.q.LatestOutageEnd(ctx, sqlcgen.LatestOutageEndParams{StackID: stackID, Service: service})
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get last outage end of %s: %w", service, err)
	}
	return time.Unix(end.Int64, 0).UTC(), nil
}

func toOutage(row sqlcgen.ServiceOutage) *deploy.Outage {
	o := &deploy.Outage{
		ID: row.ID, StackID: row.StackID, Service: row.Service, Container: row.Container,
		StartedAt: time.Unix(row.StartedAt, 0).UTC(), ExitCode: int(row.ExitCode), OOM: row.Oom != 0, RestartCount: int(row.RestartCount),
	}
	if row.EndedAt.Valid {
		end := time.Unix(row.EndedAt.Int64, 0).UTC()
		o.EndedAt = &end
	}
	return o
}

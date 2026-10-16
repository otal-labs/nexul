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

var _ deploy.ErrorRateRepo = (*ErrorRateRepo)(nil)

// ErrorRateRepo backs deploy.ErrorRateRepo: the error_rate_episodes table.
type ErrorRateRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *ErrorRateRepo) Open(ctx context.Context, e *deploy.ErrorRateEpisode, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateErrorRateEpisode(ctx, sqlcgen.CreateErrorRateEpisodeParams{
			ID: e.ID, StackID: e.StackID, StartedAt: e.StartedAt.Unix(), PeakPercent: e.PeakPercent,
			Requests: e.Requests, Errors: e.Errors, LimitPercent: e.LimitPercent,
			WindowMinutes: int64(e.WindowMinutes), MinRequests: int64(e.MinRequests),
		})
		if err != nil {
			return fmt.Errorf("insert error-rate episode %s: %w", e.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *ErrorRateRepo) UpdatePeak(ctx context.Context, e *deploy.ErrorRateEpisode) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpdateErrorRateEpisodePeak(ctx, sqlcgen.UpdateErrorRateEpisodePeakParams{
			PeakPercent: e.PeakPercent, Requests: e.Requests, Errors: e.Errors, ID: e.ID,
		})
		if err != nil {
			return fmt.Errorf("update error-rate episode %s: %w", e.ID, err)
		}
		return nil
	})
}

func (r *ErrorRateRepo) Close(ctx context.Context, id string, endedAt time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).CloseErrorRateEpisode(ctx, sqlcgen.CloseErrorRateEpisodeParams{EndedAt: sql.NullInt64{Int64: endedAt.Unix(), Valid: true}, ID: id})
		if err != nil {
			return fmt.Errorf("close error-rate episode %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("close error-rate episode %s: %w", id, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *ErrorRateRepo) GetOpen(ctx context.Context, stackID string) (*deploy.ErrorRateEpisode, error) {
	row, err := r.q.GetOpenErrorRateEpisode(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("get open error-rate episode of stack %s: %w", stackID, notFoundIfNoRows(err))
	}
	return toErrorRateEpisode(row), nil
}

func (r *ErrorRateRepo) LatestEnd(ctx context.Context, stackID string) (time.Time, error) {
	end, err := r.q.LatestErrorRateEnd(ctx, stackID)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get last error-rate end of stack %s: %w", stackID, err)
	}
	return time.Unix(end.Int64, 0).UTC(), nil
}

func toErrorRateEpisode(row sqlcgen.ErrorRateEpisode) *deploy.ErrorRateEpisode {
	e := &deploy.ErrorRateEpisode{
		ID: row.ID, StackID: row.StackID, StartedAt: time.Unix(row.StartedAt, 0).UTC(), PeakPercent: row.PeakPercent,
		Requests: row.Requests, Errors: row.Errors, LimitPercent: row.LimitPercent,
		WindowMinutes: int(row.WindowMinutes), MinRequests: int(row.MinRequests),
	}
	if row.EndedAt.Valid {
		end := time.Unix(row.EndedAt.Int64, 0).UTC()
		e.EndedAt = &end
	}
	return e
}

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/plays"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func (r *PlayQueueRepo) CreateSweep(ctx context.Context, s *plays.Sweep, evts ...eventbus.OutboxEvent) (bool, error) {
	inserted := false
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).InsertAutoPlaySweep(ctx, sqlcgen.InsertAutoPlaySweepParams{
			ID: s.ID, WorkspaceID: s.WorkspaceID, AutoPlayID: s.AutoPlayID, PlayID: s.PlayID, PlayLabel: s.PlayLabel,
			FiredAt: s.FiredAt.Unix(), StartBy: nullUnixPtr(s.StartBy), Matched: int64(s.Matched), OverCap: int64(s.OverCap),
			FinishedAt: nullUnixPtr(s.FinishedAt),
		})
		if err != nil {
			return fmt.Errorf("insert sweep %s: %w", s.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return nil
		}
		inserted = true
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
	return inserted, err
}

func (r *PlayQueueRepo) GetSweep(ctx context.Context, id string) (*plays.Sweep, error) {
	row, err := r.q.GetAutoPlaySweep(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sweep %s: %w", id, notFoundIfNoRows(err))
	}
	return toSweep(row), nil
}

func (r *PlaysRepo) LastSweeps(ctx context.Context, autoPlayIDs []string) (map[string]*plays.Sweep, error) {
	out := map[string]*plays.Sweep{}
	if len(autoPlayIDs) == 0 {
		return out, nil
	}
	rows, err := r.q.ListLastAutoPlaySweeps(ctx, idsJSON(autoPlayIDs))
	if err != nil {
		return nil, fmt.Errorf("list the last sweeps: %w", err)
	}
	for _, row := range rows {
		out[row.AutoPlayID] = toSweep(row)
	}
	return out, nil
}

func (r *PlayQueueRepo) ListOpenSweeps(ctx context.Context) ([]*plays.Sweep, error) {
	rows, err := r.q.ListOpenAutoPlaySweeps(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the open sweeps: %w", err)
	}
	out := make([]*plays.Sweep, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSweep(row))
	}
	return out, nil
}

func (r *PlayQueueRepo) FinishSweep(ctx context.Context, s *plays.Sweep, evts ...eventbus.OutboxEvent) (bool, error) {
	counts, err := jsonx.Marshal(s.Counts)
	if err != nil {
		return false, fmt.Errorf("encode the counts of sweep %s: %w", s.ID, err)
	}
	finished := false
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).FinishAutoPlaySweep(ctx, sqlcgen.FinishAutoPlaySweepParams{FinishedAt: nullUnixPtr(s.FinishedAt), Counts: string(counts), ID: s.ID})
		if err != nil {
			return fmt.Errorf("finish sweep %s: %w", s.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return nil
		}
		finished = true
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
	return finished, err
}

func (r *PlayQueueRepo) ListQueueBySweep(ctx context.Context, sweepID string) ([]*plays.QueueItem, error) {
	rows, err := r.q.ListPlayQueueBySweep(ctx, sweepID)
	if err != nil {
		return nil, fmt.Errorf("list the queue of sweep %s: %w", sweepID, err)
	}
	return toQueueItems(rows), nil
}

func (r *PlayQueueRepo) SweepBusy(ctx context.Context, sweepID string) (bool, error) {
	busy, err := r.q.PlayQueueSweepBusy(ctx, sweepID)
	if err != nil {
		return false, fmt.Errorf("check sweep %s: %w", sweepID, err)
	}
	return busy, nil
}

func (r *PlayQueueRepo) BusyTargets(ctx context.Context, targetType plays.TargetType, targetIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(targetIDs) == 0 {
		return out, nil
	}
	rows, err := r.q.ListBusyPlayQueueTargets(ctx, sqlcgen.ListBusyPlayQueueTargetsParams{TargetType: string(targetType), TargetIds: idsJSON(targetIDs)})
	if err != nil {
		return nil, fmt.Errorf("list the busy %ss: %w", targetType, err)
	}
	for _, id := range rows {
		out[id] = true
	}
	return out, nil
}

func (r *PlayQueueRepo) SweepWorkingOn(ctx context.Context, targetType plays.TargetType, targetID string) (bool, error) {
	working, err := r.q.PlayQueueSweepActiveOnTarget(ctx, sqlcgen.PlayQueueSweepActiveOnTargetParams{TargetType: string(targetType), TargetID: targetID})
	if err != nil {
		return false, fmt.Errorf("check for a sweep on %s %s: %w", targetType, targetID, err)
	}
	return working, nil
}

func (r *PlayQueueRepo) LockHeld(ctx context.Context, lockKey, exceptID string) (bool, error) {
	held, err := r.q.PlayQueueLockHeld(ctx, sqlcgen.PlayQueueLockHeldParams{LockKey: lockKey, ID: exceptID})
	if err != nil {
		return false, fmt.Errorf("check lock %s: %w", lockKey, err)
	}
	return held, nil
}

func (r *PlayQueueRepo) NextStartBy(ctx context.Context, after time.Time) (time.Time, bool, error) {
	v, err := r.q.NextPlayQueueStartBy(ctx, sql.NullInt64{Int64: after.Unix(), Valid: true})
	if err != nil {
		return time.Time{}, false, err
	}
	n, ok, err := optionalInt(v)
	if err != nil || !ok {
		return time.Time{}, false, err
	}
	return time.Unix(n, 0).UTC(), true, nil
}

func toSweep(row sqlcgen.AutoPlaySweep) *plays.Sweep {
	return &plays.Sweep{
		ID: row.ID, WorkspaceID: row.WorkspaceID, AutoPlayID: row.AutoPlayID, PlayID: row.PlayID, PlayLabel: row.PlayLabel,
		FiredAt: time.Unix(row.FiredAt, 0).UTC(), StartBy: unixPtrFromNull(row.StartBy), Matched: int(row.Matched),
		OverCap: int(row.OverCap), Counts: decodeJSON(row.Counts, plays.SweepCounts{}), FinishedAt: unixPtrFromNull(row.FinishedAt),
	}
}

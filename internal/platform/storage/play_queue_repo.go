package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ plays.QueueRepo = (*PlayQueueRepo)(nil)

// PlayQueueRepo persists the run queue of auto plays (ADR 0132).
type PlayQueueRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

var levelPriority = map[plays.Level]int64{plays.LevelLow: 0, plays.LevelNormal: 1, plays.LevelHigh: 2}

func (r *PlayQueueRepo) EnqueueRun(ctx context.Context, it *plays.QueueItem, evts ...eventbus.OutboxEvent) (bool, error) {
	inserted := false
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).InsertPlayQueueItem(ctx, sqlcgen.InsertPlayQueueItemParams{
			ID: it.ID, WorkspaceID: it.WorkspaceID, ProjectID: it.ProjectID, TargetType: string(it.TargetType), TargetID: it.TargetID,
			PlayID: it.PlayID, PlayLabel: it.PlayLabel, AutoPlayID: it.AutoPlayID, AutomationID: it.AutomationID, PersonID: it.PersonID, RunOn: string(it.RunOn),
			Moment: string(it.Moment), Via: string(it.Via), Priority: levelPriority[it.Priority], Status: string(it.Status),
			Reason: it.Reason, TrailID: it.TrailID, QueuedAt: it.QueuedAt.Unix(), DecidedAt: nullUnixPtr(it.DecidedAt),
			NotBefore: it.NotBefore.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert queue item %s: %w", it.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return nil
		}
		inserted = true
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
	return inserted, err
}

func (r *PlayQueueRepo) GetQueueItem(ctx context.Context, id string) (*plays.QueueItem, error) {
	row, err := r.q.GetPlayQueueItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get queue item %s: %w", id, notFoundIfNoRows(err))
	}
	return toQueueItem(row), nil
}

func (r *PlayQueueRepo) MoveQueueItem(ctx context.Context, it *plays.QueueItem, from plays.QueueStatus, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdatePlayQueueItem(ctx, sqlcgen.UpdatePlayQueueItemParams{
			Status: string(it.Status), Reason: it.Reason, TrailID: it.TrailID, DecidedAt: nullUnixPtr(it.DecidedAt),
			NotBefore: it.NotBefore.Unix(), ID: it.ID, FromStatus: string(from),
		})
		if err != nil {
			return fmt.Errorf("update queue item %s: %w", it.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update queue item %s: no longer %s: %w", it.ID, from, apperrs.ErrConflict)
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlayQueueRepo) ListQueueByTarget(ctx context.Context, targetType plays.TargetType, targetID string) ([]*plays.QueueItem, error) {
	rows, err := r.q.ListPlayQueueByTarget(ctx, sqlcgen.ListPlayQueueByTargetParams{TargetType: string(targetType), TargetID: targetID})
	if err != nil {
		return nil, fmt.Errorf("list queue of %s %s: %w", targetType, targetID, err)
	}
	return toQueueItems(rows), nil
}

func (r *PlayQueueRepo) ListQueuedByPlay(ctx context.Context, playID string, scope plays.ProjectScope) ([]*plays.QueueItem, error) {
	rows, err := r.q.ListQueuedPlayQueueByPlay(ctx, sqlcgen.ListQueuedPlayQueueByPlayParams{
		PlayID: playID, AllProjects: scope.All, ProjectIds: idsJSON(scope.ProjectIDs),
	})
	if err != nil {
		return nil, fmt.Errorf("list the queued runs of play %s: %w", playID, err)
	}
	return toQueueItems(rows), nil
}

func (r *PlayQueueRepo) Resume(ctx context.Context, targetType plays.TargetType, targetID, by string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertPlayQueueResume(ctx, sqlcgen.UpsertPlayQueueResumeParams{
			TargetType: string(targetType), TargetID: targetID, ResumedBy: by, ResumedAt: at.Unix(),
		})
		if err != nil {
			return fmt.Errorf("resume %s %s: %w", targetType, targetID, classifyWriteErr(err))
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlayQueueRepo) QueuedPeople(ctx context.Context, now time.Time) ([]string, error) {
	return r.q.ListPlayQueuePeople(ctx, now.Unix())
}

func (r *PlayQueueRepo) ListDue(ctx context.Context, personID string, now time.Time) ([]*plays.QueueItem, error) {
	rows, err := r.q.ListDuePlayQueue(ctx, sqlcgen.ListDuePlayQueueParams{PersonID: personID, NotBefore: now.Unix()})
	if err != nil {
		return nil, fmt.Errorf("list the due queue of %s: %w", personID, err)
	}
	return toQueueItems(rows), nil
}

func (r *PlayQueueRepo) NextNotBefore(ctx context.Context, after time.Time) (time.Time, bool, error) {
	v, err := r.q.NextPlayQueueNotBefore(ctx, after.Unix())
	if err != nil {
		return time.Time{}, false, err
	}
	n, ok, err := optionalInt(v)
	if err != nil || !ok {
		return time.Time{}, false, err
	}
	return time.Unix(n, 0).UTC(), true, nil
}

func (r *PlayQueueRepo) CountActiveRuns(ctx context.Context, personID string) (int, error) {
	n, err := r.q.CountActivePlayTrailsByStarter(ctx, personID)
	return int(n), err
}

func (r *PlayQueueRepo) CountAutoRuns(ctx context.Context, targetType plays.TargetType, targetID string, since time.Time) (int, time.Time, error) {
	row, err := r.q.CountPlayQueueStarted(ctx, sqlcgen.CountPlayQueueStartedParams{TargetType: string(targetType), TargetID: targetID, Since: since.Unix()})
	if err != nil {
		return 0, time.Time{}, err
	}
	oldest, ok, err := optionalInt(row.Oldest)
	if err != nil || !ok {
		return int(row.Started), time.Time{}, err
	}
	return int(row.Started), time.Unix(oldest, 0).UTC(), nil
}

func (r *PlayQueueRepo) QueuedSince(ctx context.Context, autoPlayID string, targetType plays.TargetType, targetID string, since time.Time) (bool, error) {
	return r.q.PlayQueueQueuedSince(ctx, sqlcgen.PlayQueueQueuedSinceParams{
		TargetType: string(targetType), TargetID: targetID, QueuedAt: since.Unix(), AutoPlayID: autoPlayID,
	})
}

func (r *PlayQueueRepo) RecoverDispatching(ctx context.Context) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return r.q.WithTx(tx).RecoverDispatchingPlayQueue(ctx)
	})
}

func toQueueItems(rows []sqlcgen.PlayQueue) []*plays.QueueItem {
	out := make([]*plays.QueueItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toQueueItem(row))
	}
	return out
}

func toQueueItem(row sqlcgen.PlayQueue) *plays.QueueItem {
	level := plays.LevelNormal
	for l, p := range levelPriority {
		if p == row.Priority {
			level = l
		}
	}
	return &plays.QueueItem{
		ID: row.ID, WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, TargetType: plays.TargetType(row.TargetType),
		TargetID: row.TargetID, PlayID: row.PlayID, PlayLabel: row.PlayLabel, AutoPlayID: row.AutoPlayID, AutomationID: row.AutomationID, PersonID: row.PersonID,
		RunOn: plays.RunOn(row.RunOn), Moment: plays.Moment(row.Moment), Priority: level, Status: plays.QueueStatus(row.Status),
		Reason: row.Reason, TrailID: row.TrailID, Via: plays.Via(row.Via), QueuedAt: time.Unix(row.QueuedAt, 0).UTC(),
		DecidedAt: unixPtrFromNull(row.DecidedAt), NotBefore: time.Unix(row.NotBefore, 0).UTC(),
	}
}

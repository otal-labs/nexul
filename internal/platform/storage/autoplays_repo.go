package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func (r *PlaysRepo) CreateAutoPlay(ctx context.Context, a *plays.AutoPlay, evts ...eventbus.OutboxEvent) error {
	conditions, priority, err := encodeAutoPlayTrees(a)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateAutoPlay(ctx, sqlcgen.CreateAutoPlayParams{
			ID: a.ID, PlayID: a.PlayID, WorkspaceID: a.WorkspaceID, Enabled: int64(boolInt(a.Enabled)),
			Moment: string(a.Moment), MomentStage: nullStage(a.MomentStage), Conditions: conditions, Priority: priority,
			OnceWithinMinutes: int64(a.OnceWithinMinutes), RunOn: string(a.RunOn), CreatedBy: a.CreatedBy,
			CreatedAt: a.CreatedAt.Unix(), UpdatedAt: a.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert auto play %s: %w", a.ID, classifyWriteErr(err))
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlaysRepo) GetAutoPlay(ctx context.Context, id string) (*plays.AutoPlay, error) {
	row, err := r.q.GetAutoPlay(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get auto play %s: %w", id, notFoundIfNoRows(err))
	}
	return toAutoPlay(row)
}

func (r *PlaysRepo) ListAutoPlays(ctx context.Context, playIDs []string) ([]*plays.AutoPlay, error) {
	if len(playIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListAutoPlaysByPlays(ctx, idsJSON(playIDs))
	if err != nil {
		return nil, fmt.Errorf("list auto plays: %w", err)
	}
	out := make([]*plays.AutoPlay, 0, len(rows))
	for _, row := range rows {
		a, err := toAutoPlay(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *PlaysRepo) UpdateAutoPlay(ctx context.Context, a *plays.AutoPlay, evts ...eventbus.OutboxEvent) error {
	conditions, priority, err := encodeAutoPlayTrees(a)
	if err != nil {
		return err
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateAutoPlay(ctx, sqlcgen.UpdateAutoPlayParams{
			Enabled: int64(boolInt(a.Enabled)), Moment: string(a.Moment), MomentStage: nullStage(a.MomentStage),
			Conditions: conditions, Priority: priority, OnceWithinMinutes: int64(a.OnceWithinMinutes),
			RunOn: string(a.RunOn), UpdatedAt: a.UpdatedAt.Unix(), ID: a.ID,
		})
		if err != nil {
			return fmt.Errorf("update auto play %s: %w", a.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update auto play %s: %w", a.ID, apperrs.ErrNotFound)
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func (r *PlaysRepo) DeleteAutoPlay(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteAutoPlay(ctx, id)
		if err != nil {
			return fmt.Errorf("delete auto play %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("delete auto play %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

// ListEnabledAutoPlays reads a workspace's switched-on auto plays waiting for moment, through idx_auto_plays_moment.
func (r *PlaysRepo) ListEnabledAutoPlays(ctx context.Context, workspaceID string, moment plays.Moment) ([]*plays.AutoPlay, error) {
	rows, err := r.q.ListEnabledAutoPlaysByMoment(ctx, sqlcgen.ListEnabledAutoPlaysByMomentParams{WorkspaceID: workspaceID, Moment: string(moment)})
	if err != nil {
		return nil, fmt.Errorf("list auto plays for %s in workspace %s: %w", moment, workspaceID, err)
	}
	out := make([]*plays.AutoPlay, 0, len(rows))
	for _, row := range rows {
		a, err := toAutoPlay(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// AutoPlayDailyCap reads workspaceID's cap on automatic runs per ticket per day.
func (r *PlaysRepo) AutoPlayDailyCap(ctx context.Context, workspaceID string) (int, error) {
	limit, err := r.q.GetAutoPlayDailyCap(ctx, workspaceID)
	if err != nil {
		return 0, fmt.Errorf("get auto play daily cap for workspace %s: %w", workspaceID, notFoundIfNoRows(err))
	}
	return int(limit), nil
}

// SetAutoPlayDailyCap changes workspaceID's cap on automatic runs per ticket per day.
func (r *PlaysRepo) SetAutoPlayDailyCap(ctx context.Context, workspaceID string, limit int, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetAutoPlayDailyCap(ctx, sqlcgen.SetAutoPlayDailyCapParams{AutoPlayDailyCap: int64(limit), ID: workspaceID})
		if err != nil {
			return classifyWriteErr(err)
		}
		if n == 0 {
			return apperrs.ErrNotFound
		}
		return enqueuePlaysOutbox(ctx, tx, evts)
	})
}

func encodeAutoPlayTrees(a *plays.AutoPlay) (string, string, error) {
	conditions, err := json.Marshal(a.Conditions)
	if err != nil {
		return "", "", fmt.Errorf("encode conditions of auto play %s: %w", a.ID, err)
	}
	priority, err := json.Marshal(a.Priority)
	if err != nil {
		return "", "", fmt.Errorf("encode priority of auto play %s: %w", a.ID, err)
	}
	return string(conditions), string(priority), nil
}

func toAutoPlay(row sqlcgen.AutoPlay) (*plays.AutoPlay, error) {
	a := &plays.AutoPlay{
		ID: row.ID, PlayID: row.PlayID, WorkspaceID: row.WorkspaceID, Enabled: row.Enabled != 0,
		Moment: plays.Moment(row.Moment), OnceWithinMinutes: int(row.OnceWithinMinutes), RunOn: plays.RunOn(row.RunOn),
		CreatedBy: row.CreatedBy, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if row.MomentStage.Valid {
		s := plays.Stage(row.MomentStage.String)
		a.MomentStage = &s
	}
	if err := json.Unmarshal([]byte(row.Conditions), &a.Conditions); err != nil {
		return nil, fmt.Errorf("decode conditions of auto play %s: %w", row.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Priority), &a.Priority); err != nil {
		return nil, fmt.Errorf("decode priority of auto play %s: %w", row.ID, err)
	}
	return a, nil
}

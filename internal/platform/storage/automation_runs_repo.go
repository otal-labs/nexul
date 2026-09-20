package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ automations.RunsRepo = (*AutomationRunsRepo)(nil)

// AutomationRunsRepo persists automation run reports (ticket 09).
type AutomationRunsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AutomationRunsRepo) Create(ctx context.Context, run *automations.Run) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateAutomationRun(ctx, sqlcgen.CreateAutomationRunParams{
			ID: run.ID, AutomationID: run.AutomationID, EventTopic: run.EventTopic, EventID: run.EventID,
			Outcome: string(run.Outcome), Error: run.Error,
			StartedAt: run.StartedAt.Unix(), FinishedAt: run.FinishedAt.Unix(), DurationMs: int64(run.DurationMS),
			Logs: run.Logs, CreatedAt: run.CreatedAt.Unix(),
		})
		if err != nil {
			return classifyWriteErr(err)
		}
		return nil
	})
}

func (r *AutomationRunsRepo) Get(ctx context.Context, id string) (*automations.Run, error) {
	row, err := r.q.GetAutomationRun(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get automation run %s: %w", id, notFoundIfNoRows(err))
	}
	run := toAutomationRun(row)
	return &run, nil
}

func (r *AutomationRunsRepo) ListByAutomation(ctx context.Context, automationID string, limit int) ([]automations.Run, error) {
	rows, err := r.q.ListAutomationRunsByAutomation(ctx, sqlcgen.ListAutomationRunsByAutomationParams{
		AutomationID: automationID, Limit: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list automation runs for %s: %w", automationID, err)
	}
	var out []automations.Run
	for _, row := range rows {
		out = append(out, toAutomationRun(row))
	}
	return out, nil
}

func (r *AutomationRunsRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	var n int64
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var err error
		n, err = r.q.WithTx(tx).DeleteAutomationRunsOlderThan(ctx, before.Unix())
		if err != nil {
			return fmt.Errorf("delete old automation runs: %w", err)
		}
		return nil
	})
	return n, err
}

func toAutomationRun(row sqlcgen.AutomationRun) automations.Run {
	return automations.Run{
		ID: row.ID, AutomationID: row.AutomationID, EventTopic: row.EventTopic, EventID: row.EventID,
		Outcome: automations.RunOutcome(row.Outcome), Error: row.Error,
		StartedAt: time.Unix(row.StartedAt, 0).UTC(), FinishedAt: time.Unix(row.FinishedAt, 0).UTC(),
		DurationMS: int64(row.DurationMs), Logs: row.Logs, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

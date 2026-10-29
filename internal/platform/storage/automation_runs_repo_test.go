package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestRun(id, automationID string, createdAt time.Time) *automations.Run {
	return &automations.Run{
		ID:           id,
		AutomationID: automationID,
		EventTopic:   "ticket.created",
		EventID:      "ev-" + id,
		Outcome:      automations.RunOutcomeSuccess,
		StartedAt:    createdAt,
		FinishedAt:   createdAt.Add(time.Second),
		DurationMS:   1000,
		Logs:         "did the thing\n",
		CreatedAt:    createdAt,
	}
}

func TestAutomationRunsRepo_Create_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Now().UTC()
	want := newTestRun("r1", "a1", now)
	want.Error = "boom"
	want.Outcome = automations.RunOutcomeCrash
	require.NoError(t, s.AutomationRuns.Create(context.Background(), want))

	got, err := s.AutomationRuns.Get(context.Background(), "r1")
	require.NoError(t, err)
	assert.Equal(t, "a1", got.AutomationID)
	assert.Equal(t, automations.RunOutcomeCrash, got.Outcome)
	assert.Equal(t, "boom", got.Error)
	assert.Equal(t, "did the thing\n", got.Logs)
	assert.Equal(t, int64(1000), got.DurationMS)
	assert.WithinDuration(t, now, got.StartedAt, time.Second)
}

func TestAutomationRunsRepo_ListByAutomation_FiltersAndOrdersNewestFirst(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("r1", "a1", base)))
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("r2", "a1", base.Add(time.Minute))))
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("r3", "a2", base.Add(2*time.Minute))))

	got, err := s.AutomationRuns.ListByAutomation(context.Background(), "a1", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "r2", got[0].ID, "newest first")
	assert.Equal(t, "r1", got[1].ID)

	capped, err := s.AutomationRuns.ListByAutomation(context.Background(), "a1", 1)
	require.NoError(t, err)
	require.Len(t, capped, 1, "the limit caps the page")
	assert.Equal(t, "r2", capped[0].ID)
}

func TestAutomationRunsRepo_ListByAutomation_UnknownAutomation_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.AutomationRuns.ListByAutomation(context.Background(), "nope", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAutomationRunsRepo_DeleteOlderThan_RemovesOnlyOldRows(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	old := time.Now().UTC().Add(-40 * 24 * time.Hour)
	recent := time.Now().UTC()
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("old", "a1", old)))
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("recent", "a1", recent)))

	n, err := s.AutomationRuns.DeleteOlderThan(context.Background(), time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	_, err = s.AutomationRuns.Get(context.Background(), "old")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.AutomationRuns.Get(context.Background(), "recent")
	require.NoError(t, err)
}

func TestAutomationRunsRepo_DeleteOlderThan_NothingToDelete_ReturnsZero(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("recent", "a1", time.Now().UTC())))

	n, err := s.AutomationRuns.DeleteOlderThan(context.Background(), time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestAutomationRunsRepo_ListFailedByAutomation_KeepsFailuresAndCrashesNewestFirst(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC().Add(-time.Hour)
	for i, outcome := range []automations.RunOutcome{automations.RunOutcomeFailure, automations.RunOutcomeSuccess, automations.RunOutcomeCrash, automations.RunOutcomeSuccess} {
		run := newTestRun(fmt.Sprintf("r%d", i), "a1", base.Add(time.Duration(i)*time.Minute))
		run.Outcome = outcome
		require.NoError(t, s.AutomationRuns.Create(context.Background(), run))
	}
	other := newTestRun("other", "a2", base)
	other.Outcome = automations.RunOutcomeFailure
	require.NoError(t, s.AutomationRuns.Create(context.Background(), other))

	got, err := s.AutomationRuns.ListFailedByAutomation(context.Background(), "a1", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "r2", got[0].ID, "the crash is newest")
	assert.Equal(t, "r0", got[1].ID)

	capped, err := s.AutomationRuns.ListFailedByAutomation(context.Background(), "a1", 1)
	require.NoError(t, err)
	require.Len(t, capped, 1, "the limit counts failures, not every run")
	assert.Equal(t, "r2", capped[0].ID)
}

func TestAutomationRunsRepo_ListFailedByAutomation_ReadsThePartialIndexWithoutASort(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	rows, err := s.db.QueryContext(t.Context(), `EXPLAIN QUERY PLAN SELECT * FROM automation_runs WHERE automation_id = 'a1' AND outcome <> 'success' ORDER BY started_at DESC, id DESC LIMIT 50`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var plan string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		plan += detail + "\n"
	}
	require.NoError(t, rows.Err())
	assert.Contains(t, plan, "idx_automation_runs_failed", "the failures view walks only the automation's failed rows")
	assert.NotContains(t, plan, "TEMP B-TREE", "the index already holds the newest-first order")
}

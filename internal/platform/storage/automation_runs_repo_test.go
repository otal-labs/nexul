package storage

import (
	"context"
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

func TestAutomationRunsRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.AutomationRuns.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestAutomationRunsRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Now().UTC()
	require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun("r1", "a1", now)))
	err := s.AutomationRuns.Create(context.Background(), newTestRun("r1", "a1", now))
	require.ErrorIs(t, err, apperrs.ErrConflict)
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
}

func TestAutomationRunsRepo_ListByAutomation_RespectsLimit(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	base := time.Now().UTC()
	for i, id := range []string{"r1", "r2", "r3"} {
		require.NoError(t, s.AutomationRuns.Create(context.Background(), newTestRun(id, "a1", base.Add(time.Duration(i)*time.Minute))))
	}

	got, err := s.AutomationRuns.ListByAutomation(context.Background(), "a1", 2)
	require.NoError(t, err)
	assert.Len(t, got, 2)
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

package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"
)

func newTestQueueItem(id, autoPlayID, targetID, person string, level plays.Level, at time.Time) *plays.QueueItem {
	return &plays.QueueItem{
		ID: id, WorkspaceID: "ws-1", ProjectID: "p-1", TargetType: plays.TargetTicket, TargetID: targetID, PlayID: "play-1",
		PlayLabel: "Fix", AutoPlayID: autoPlayID, PersonID: person, RunOn: plays.RunOnDeveloper, Moment: plays.MomentTicketUnblocked,
		Priority: level, Status: plays.QueueQueued, Via: plays.ViaWeb, QueuedAt: at, NotBefore: at, ComputerIDs: []string{},
	}
}

func TestPlayQueueRepo_WaitingMatchIsANoOpAndPublishesNothing(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	evt := func(id string) eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: id, Topic: plays.TopicQueued, Payload: map[string]string{"id": id}}
	}

	inserted, err := s.PlayQueue.EnqueueRun(ctx, newTestQueueItem("q-1", "ap-1", "t-1", "u-1", plays.LevelHigh, at), evt("e-1"))
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = s.PlayQueue.EnqueueRun(ctx, newTestQueueItem("q-2", "ap-1", "t-1", "u-1", plays.LevelHigh, at), evt("e-2"))
	require.NoError(t, err)
	assert.False(t, inserted, "the same auto play already waits on the ticket")
	var n int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE topic = ?`, plays.TopicQueued).Scan(&n))
	assert.Equal(t, 1, n)

	got, err := s.PlayQueue.GetQueueItem(ctx, "q-1")
	require.NoError(t, err)
	assert.Equal(t, newTestQueueItem("q-1", "ap-1", "t-1", "u-1", plays.LevelHigh, at), got)
	_, err = s.PlayQueue.GetQueueItem(ctx, "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	got.Status, got.DecidedAt = plays.QueueStarted, &at
	require.NoError(t, s.PlayQueue.MoveQueueItem(ctx, got, plays.QueueQueued))
	require.ErrorIs(t, s.PlayQueue.MoveQueueItem(ctx, got, plays.QueueQueued), apperrs.ErrConflict, "it already moved on")
	inserted, err = s.PlayQueue.EnqueueRun(ctx, newTestQueueItem("q-3", "ap-1", "t-1", "u-1", plays.LevelHigh, at))
	require.NoError(t, err)
	assert.True(t, inserted, "once started, the next match waits again")
}

func TestPlayQueueRepo_KeepsTheStackServiceAndOutageARunWasQueuedFor(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	at := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	it := newTestQueueItem("q-1", "ap-1", "st-1", "u-alice", plays.LevelNormal, at)
	it.TargetType, it.RunOn, it.Moment = plays.TargetStack, plays.RunOnComputer, plays.MomentStackServiceCrashed
	it.ComputerIDs, it.ComputerID, it.Service, it.OutageID = []string{"c-2", "c-1"}, "c-2", "api", "o-1"
	_, err := s.PlayQueue.EnqueueRun(t.Context(), it)
	require.NoError(t, err)

	got, err := s.PlayQueue.GetQueueItem(t.Context(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, it, got)
	listed, err := s.PlayQueue.ListQueueByTarget(t.Context(), plays.TargetStack, "st-1")
	require.NoError(t, err)
	assert.Equal(t, []*plays.QueueItem{it}, listed)
}

func TestPlayQueueRepo_Move_KeepsWhatTheRunWaitsOnOnlyWhileItWaits(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	_, err := s.PlayQueue.EnqueueRun(ctx, newTestQueueItem("q-1", "ap-1", "t-1", "u-1", plays.LevelNormal, at))
	require.NoError(t, err)
	it, err := s.PlayQueue.GetQueueItem(ctx, "q-1")
	require.NoError(t, err)
	stale := *it

	it.Status, it.Reason = plays.QueueDispatching, ""
	require.NoError(t, s.PlayQueue.MoveQueueItem(ctx, it, plays.QueueQueued))
	waiting := []plays.PassedOver{{ComputerID: "c-laptop", Name: "laptop", Why: "full", Text: "laptop is full (1 of 1 running)"}}
	it.Status, it.Reason, it.WaitingOn = plays.QueueQueued, "no_computer_ready", waiting
	require.NoError(t, s.PlayQueue.MoveQueueItem(ctx, it, plays.QueueDispatching))

	got, err := s.PlayQueue.GetQueueItem(ctx, "q-1")
	require.NoError(t, err)
	assert.Equal(t, waiting, got.WaitingOn)
	stale.Status = plays.QueueCancelled
	require.ErrorIs(t, s.PlayQueue.MoveQueueItem(ctx, &stale, plays.QueueDispatching), apperrs.ErrConflict, "a move from a status the row left is refused")
}

func TestPlayQueueRepo_DueByPriorityThenAge(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	later := newTestQueueItem("q-later", "ap-1", "t-later", "u-1", plays.LevelLow, at.Add(time.Hour))
	later.NotBefore = at.Add(2 * time.Hour)
	for _, it := range []*plays.QueueItem{
		newTestQueueItem("q-low", "ap-1", "t-1", "u-1", plays.LevelLow, at),
		newTestQueueItem("q-high-new", "ap-1", "t-2", "u-1", plays.LevelHigh, at.Add(time.Minute)),
		newTestQueueItem("q-high-old", "ap-1", "t-3", "u-1", plays.LevelHigh, at),
		newTestQueueItem("q-other", "ap-1", "t-4", "u-2", plays.LevelNormal, at),
		later,
	} {
		_, err := s.PlayQueue.EnqueueRun(ctx, it)
		require.NoError(t, err)
	}
	now := at.Add(time.Hour)

	due, err := s.PlayQueue.ListDue(ctx, "u-1", now)
	require.NoError(t, err)
	ids := make([]string, 0, len(due))
	for _, it := range due {
		ids = append(ids, it.ID)
	}
	assert.Equal(t, []string{"q-high-old", "q-high-new", "q-low"}, ids)
	people, err := s.PlayQueue.QueuedPeople(ctx, now)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"u-1", "u-2"}, people)
	next, ok, err := s.PlayQueue.NextNotBefore(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, []any{true, at.Add(2 * time.Hour)}, []any{ok, next})
	_, ok, err = s.PlayQueue.NextNotBefore(ctx, at.Add(3*time.Hour))
	require.NoError(t, err)
	assert.False(t, ok)

	onTargets, err := s.PlayQueue.ListQueuedByTargets(ctx, plays.TargetTicket, []string{"t-1", "t-3", "t-9"})
	require.NoError(t, err)
	require.Len(t, onTargets, 2)
	assert.Equal(t, []string{"q-high-old", "q-low"}, []string{onTargets[0].ID, onTargets[1].ID})
	onTargets, err = s.PlayQueue.ListQueuedByTargets(ctx, plays.TargetTicket, nil)
	require.NoError(t, err)
	assert.Empty(t, onTargets)
}

func TestPlayQueueRepo_CountsStartedRunsSinceTheLaterOfTheDayAndAResume(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	for i, id := range []string{"q-1", "q-2", "q-3"} {
		it := newTestQueueItem(id, "ap-"+id, "t-1", "u-1", plays.LevelNormal, at.Add(time.Duration(i)*time.Hour))
		decided := it.QueuedAt
		it.Status, it.DecidedAt = plays.QueueStarted, &decided
		_, err := s.PlayQueue.EnqueueRun(ctx, it)
		require.NoError(t, err)
	}

	count, oldest, err := s.PlayQueue.CountAutoRuns(ctx, plays.TargetTicket, "t-1", at.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []any{2, at.Add(time.Hour)}, []any{count, oldest})
	ran, err := s.PlayQueue.QueuedSince(ctx, "ap-q-3", plays.TargetTicket, "t-1", at.Add(time.Hour))
	require.NoError(t, err)
	assert.True(t, ran)

	require.NoError(t, s.PlayQueue.Resume(ctx, plays.TargetTicket, "t-1", "u-1", at.Add(90*time.Minute)))
	count, _, err = s.PlayQueue.CountAutoRuns(ctx, plays.TargetTicket, "t-1", at)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only the run after the resume counts")
	count, oldest, err = s.PlayQueue.CountAutoRuns(ctx, plays.TargetTicket, "t-2", at)
	require.NoError(t, err)
	assert.Equal(t, []any{0, time.Time{}}, []any{count, oldest})
}

func TestPlayQueueRepo_RecoverDispatching_StartedOnlyWhenTheTrailExists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	require.NoError(t, s.PlayTrails.CreateTrail(ctx, &plays.Trail{
		ID: "tr-1", WorkspaceID: "ws-1", PlayID: "play-1", PlayLabel: "Fix", TargetType: plays.TargetTicket, TargetID: "t-1",
		ProjectID: "p-1", StarterID: "u-1", Via: plays.ViaWeb, SelectedMemoryIDs: []string{}, State: plays.TrailStarting, StartedAt: at,
	}))
	for _, it := range []struct{ id, target, trail string }{{"q-1", "t-1", "tr-1"}, {"q-2", "t-2", "tr-never-written"}} {
		item := newTestQueueItem(it.id, "ap-1", it.target, "u-1", plays.LevelNormal, at)
		item.Status, item.TrailID, item.DecidedAt = plays.QueueDispatching, it.trail, &at
		_, err := s.PlayQueue.EnqueueRun(ctx, item)
		require.NoError(t, err)
	}
	require.NoError(t, s.PlayQueue.RecoverDispatching(ctx))

	started, err := s.PlayQueue.GetQueueItem(ctx, "q-1")
	require.NoError(t, err)
	assert.Equal(t, []any{plays.QueueStarted, "tr-1", true}, []any{started.Status, started.TrailID, started.DecidedAt != nil})
	requeued, err := s.PlayQueue.GetQueueItem(ctx, "q-2")
	require.NoError(t, err)
	assert.Equal(t, []any{plays.QueueQueued, "", (*time.Time)(nil)}, []any{requeued.Status, requeued.TrailID, requeued.DecidedAt})
	listed, err := s.PlayQueue.ListQueueByTarget(ctx, plays.TargetTicket, "t-2")
	require.NoError(t, err)
	assert.Len(t, listed, 1)
}

func TestPlaysRepo_ListEnabledAutoPlays_OnlySwitchedOnForTheMoment(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	on := newTestAutoPlay("ap-on", "play-1", at)
	on.Enabled = true
	off := newTestAutoPlay("ap-off", "play-1", at)
	other := newTestAutoPlay("ap-other", "play-1", at)
	other.Enabled, other.Moment = true, plays.MomentDocCreated
	for _, a := range []*plays.AutoPlay{on, off, other} {
		require.NoError(t, s.Plays.CreateAutoPlay(ctx, a))
	}

	got, err := s.Plays.ListEnabledAutoPlays(ctx, "ws-1", plays.MomentDocChanged)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "ap-on", got[0].ID)
}

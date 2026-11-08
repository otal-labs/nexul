package plays_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// nightly is a schedule auto play on the developer, every day at 23:00 in London, stopping at 07:00, with edits applied.
func (r *rig) nightly(t *testing.T, playID string, edit ...func(*plays.AutoPlay)) *plays.AutoPlay {
	t.Helper()
	return r.autoPlay(t, playID, plays.MomentSchedule, append([]func(*plays.AutoPlay){func(a *plays.AutoPlay) {
		a.Schedule = &plays.Schedule{Cron: "0 23 * * *", Timezone: "Europe/London", StopAt: "07:00", MaxRuns: 10}
	}}, edit...)...)
}

func (r *rig) sweep(t *testing.T, a *plays.AutoPlay) *plays.Sweep {
	t.Helper()
	require.NoError(t, r.Runner.Sweep(t.Context(), a, r.clock))
	last, err := r.store.Plays.LastSweeps(t.Context(), []string{a.ID})
	require.NoError(t, err)
	require.Contains(t, last, a.ID)
	return last[a.ID]
}

func (r *rig) sweepItems(t *testing.T, s *plays.Sweep) []*plays.QueueItem {
	t.Helper()
	items, err := r.store.PlayQueue.ListQueueBySweep(t.Context(), s.ID)
	require.NoError(t, err)
	return items
}

func targets(items []*plays.QueueItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.TargetID)
	}
	return out
}

func TestSweep_QueuesWhatMatches_HighestPriorityFirst_UpToItsCap_Once(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	a := r.nightly(t, fix.ID, func(a *plays.AutoPlay) {
		a.Schedule.MaxRuns = 2
		a.Conditions = plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{{Match: plays.MatchAll, Rules: []plays.Rule{{Field: plays.FieldStage, Op: plays.OpIs, Values: []string{"backlog"}}}}}}
		a.Priority = plays.Priority{Otherwise: plays.LevelNormal, Rules: []plays.PriorityRule{{Level: plays.LevelHigh, When: plays.Group{Match: plays.MatchAll, Rules: []plays.Rule{{Field: plays.FieldType, Op: plays.OpIs, Values: []string{"bug"}}}}}}}
	})
	r.ticket("t-1")
	r.ticket("t-2", func(f *plays.Facts) { f.Type = "bug" })
	r.ticket("t-3")
	r.ticket("t-4", func(f *plays.Facts) { f.Stage = plays.StageDone })

	s := r.sweep(t, a)

	items := r.sweepItems(t, s)
	assert.Equal(t, []string{"t-2", "t-1"}, targets(items), "the bug goes first, then board order, and the cap leaves t-3")
	assert.Equal(t, plays.LevelHigh, items[0].Priority)
	assert.Equal(t, 3, s.Matched)
	assert.Equal(t, 1, s.OverCap)
	assert.Equal(t, time.Date(2026, 10, 11, 6, 0, 0, 0, time.UTC), *s.StartBy, "07:00 in London the next morning")

	require.NoError(t, r.Runner.Sweep(t.Context(), a, r.clock))
	assert.Len(t, r.sweepItems(t, s), 2, "a firing sweeps once")
}

func TestSweep_SkipsATicketAlreadyQueuedOrRunning(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	other := r.play(t, "Review it")
	r.autoPlay(t, other.ID, plays.MomentTicketUnblocked)
	a := r.nightly(t, fix.ID)
	r.ticket("t-1")
	r.ticket("t-2")
	r.ticket("t-3")
	r.unblock(t, "t-1")
	r.press(t, other.ID, "t-2", tester)

	items := r.sweepItems(t, r.sweep(t, a))

	require.Equal(t, []string{"t-1", "t-2", "t-3"}, targets(items))
	assert.Equal(t, []plays.QueueStatus{plays.QueueSkipped, plays.QueueSkipped, plays.QueueQueued}, statuses(items))
	assert.Equal(t, "it already had a run going or waiting", items[0].Reason)
}

func TestDispatch_OneAtATime_ASharedResourceLetsOneRunGo(t *testing.T) {
	r := newRig(t, openStore(t))
	r.runsAtOnce(5)
	fix := r.play(t, "Fix it")
	check := r.play(t, "Check on Windows")
	shared := func(a *plays.AutoPlay) { a.OneAtATime, a.Resource = true, "Windows-VM" }
	night := r.nightly(t, fix.ID, shared)
	r.autoPlay(t, check.ID, plays.MomentTicketUnblocked, shared, func(a *plays.AutoPlay) { a.RunOn = plays.RunOnTester })
	r.ticket("t-1")
	r.ticket("t-2")
	r.ticket("t-3")
	s := r.sweep(t, night)
	r.unblock(t, "t-3")

	r.dispatch(t)

	items := r.sweepItems(t, s)
	assert.Equal(t, []plays.QueueStatus{plays.QueueStarted, plays.QueueQueued, plays.QueueQueued}, statuses(items))
	assert.Equal(t, plays.ReasonOneAtATime, items[1].Reason)
	unblocked := r.queued(t, "t-3")
	assert.Equal(t, plays.ReasonOneAtATime, unblocked[0].Reason, "another auto play naming the resource waits too")

	r.endRuns(t, "t-1")
	r.dispatch(t)
	assert.Equal(t, []plays.QueueStatus{plays.QueueStarted, plays.QueueStarted, plays.QueueQueued}, statuses(r.sweepItems(t, s)))
}

func TestDispatch_SweepWindowClosed_SkipsWhatIsLeft(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	a := r.nightly(t, fix.ID)
	r.ticket("t-1")
	r.ticket("t-2")
	s := r.sweep(t, a)
	r.dispatch(t)

	r.clock = *s.StartBy
	r.endRuns(t, "t-1")
	r.dispatch(t)

	items := r.sweepItems(t, s)
	assert.Equal(t, []plays.QueueStatus{plays.QueueStarted, plays.QueueSkipped}, statuses(items))
	assert.Equal(t, "its sweep's window closed before it could start", items[1].Reason)
}

func TestQueue_AnotherAutoPlay_SkipsATicketASweepIsWorkingOn(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	other := r.play(t, "In progress")
	r.autoPlay(t, other.ID, plays.MomentTicketUnblocked)
	a := r.nightly(t, fix.ID)
	r.ticket("t-1")
	r.sweep(t, a)
	r.dispatch(t)

	r.unblock(t, "t-1")

	for _, it := range r.queued(t, "t-1") {
		if it.AutoPlayID != a.ID {
			assert.Equal(t, plays.QueueSkipped, it.Status)
			assert.Equal(t, "a scheduled sweep was working on it", it.Reason)
		}
	}
}

func TestSweep_FinishesOnceNothingWaitsOrWorks_WithItsCounts(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	a := r.nightly(t, fix.ID)
	r.ticket("t-1")
	r.ticket("t-2", func(f *plays.Facts) { f.Developer = "" })
	s := r.sweep(t, a)
	r.dispatch(t)

	open, err := r.store.PlayQueue.GetSweep(t.Context(), s.ID)
	require.NoError(t, err)
	assert.Nil(t, open.FinishedAt, "a run still works")

	r.endRuns(t, "t-1")
	r.dispatch(t)

	done, err := r.store.PlayQueue.GetSweep(t.Context(), s.ID)
	require.NoError(t, err)
	require.NotNil(t, done.FinishedAt)
	assert.Equal(t, plays.SweepCounts{Finished: 1, DidntRun: 1}, done.Counts)

	d, err := r.Runner.GetSweepDigest(as(t.Context(), dev), s.ID)
	require.NoError(t, err)
	require.Len(t, d.Lines, 2)
	assert.Equal(t, "finished", d.Lines[0].Outcome)
	assert.Equal(t, plays.QueueDidntRun, d.Lines[1].Status)

	r.Hide(tester, fix.ID)
	_, err = r.Runner.GetSweepDigest(as(t.Context(), tester), s.ID)
	assert.Error(t, err, "someone who cannot see the play cannot read its sweep")
}

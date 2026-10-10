package plays_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage"
	storagetest "github.com/otal-labs/nexul/internal/platform/storage/testutil"
	"github.com/otal-labs/nexul/internal/plays"
)

const (
	ws      = "workspace-default"
	project = storagetest.GeneralProjectID
	dev     = "u-dev"
	tester  = "u-tester"
)

type rig struct {
	*plays.QueueRig
	store *storage.Store
	clock time.Time
}

// migrated is one migrated database's bytes per test binary; replaying every migration per test costs seconds under -race.
var migrated struct {
	once sync.Once
	data []byte
	err  error
}

func migratedTemplate() ([]byte, error) {
	migrated.once.Do(func() {
		dir, err := os.MkdirTemp("", "nexul-plays-template-")
		if err != nil {
			migrated.err = err
			return
		}
		defer func() { migrated.err = errors.Join(migrated.err, os.RemoveAll(dir)) }()
		path := filepath.Join(dir, "template.db")
		db, err := storage.OpenDB(path)
		if err != nil {
			migrated.err = err
			return
		}
		migrated.err = errors.Join(storage.Migrate(db), storagetest.SeedGeneralProject(db))
		// A truncating checkpoint folds the WAL into the main file, so the bytes are the whole database.
		_, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		migrated.err = errors.Join(migrated.err, err, db.Close())
		if migrated.err == nil {
			migrated.data, migrated.err = os.ReadFile(path)
		}
	})
	return migrated.data, migrated.err
}

func openStore(t *testing.T) *storage.Store {
	t.Helper()
	data, err := migratedTemplate()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	db, err := storage.OpenDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
}

// newRig runs on a clock the test moves; newRigOn takes another, such as synctest's.
func newRig(t *testing.T, store *storage.Store) *rig {
	r := &rig{clock: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)}
	return newRigOn(r, store, func() time.Time { return r.clock })
}

func newRigOn(r *rig, store *storage.Store, now func() time.Time) *rig {
	r.store = store
	r.QueueRig = plays.NewQueueRig(store.Plays, store.PlayTrails, store.PlayQueue, ws, project, now)
	r.Grant(dev)
	r.Grant(tester)
	return r
}

func (r *rig) play(t *testing.T, label string) *plays.Play {
	t.Helper()
	stage := plays.StageProgress
	p := &plays.Play{
		ID: ids.New(), WorkspaceID: ws, Label: label, Type: plays.TypeTicket, Instructions: "Do it.", Enabled: true,
		ShowWhenStage: &stage, ExcludedProjectIDs: []string{}, CreatedAt: r.clock, UpdatedAt: r.clock,
	}
	require.NoError(t, r.store.Plays.Create(t.Context(), p))
	return p
}

func (r *rig) autoPlay(t *testing.T, playID string, moment plays.Moment, edit ...func(*plays.AutoPlay)) *plays.AutoPlay {
	t.Helper()
	a := &plays.AutoPlay{
		ID: ids.New(), PlayID: playID, WorkspaceID: ws, Enabled: true, Moment: moment,
		Conditions: plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{}},
		Priority:   plays.Priority{Rules: []plays.PriorityRule{}, Otherwise: plays.LevelNormal},
		RunOn:      plays.RunOnDeveloper, CreatedAt: r.clock, UpdatedAt: r.clock,
	}
	for _, e := range edit {
		e(a)
	}
	require.NoError(t, r.store.Plays.CreateAutoPlay(t.Context(), a))
	return a
}

func (r *rig) ticket(id string, edit ...func(*plays.Facts)) {
	f := plays.Facts{ProjectID: project, Stage: plays.StageBacklog, Status: "open", Type: "task", Developer: dev, Tester: tester}
	for _, e := range edit {
		e(&f)
	}
	r.SetTicket(id, f)
}

func (r *rig) moment(t *testing.T, topic string, payload map[string]any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, r.Runner.HandleAutoPlayMoment(t.Context(), eventbus.Event{ID: ids.New(), Topic: topic, Payload: raw}))
}

func (r *rig) unblock(t *testing.T, ticketID string) {
	t.Helper()
	r.moment(t, "ticket.unblocked", map[string]any{"ticket_id": ticketID, "project_id": project, "actor": map[string]any{"kind": "user", "user_id": tester}})
}

func (r *rig) queued(t *testing.T, ticketID string) []*plays.QueueItem {
	t.Helper()
	items, err := r.store.PlayQueue.ListQueueByTarget(t.Context(), plays.TargetTicket, ticketID)
	require.NoError(t, err)
	return items
}

func (r *rig) only(t *testing.T, ticketID string) *plays.QueueItem {
	t.Helper()
	items := r.queued(t, ticketID)
	require.Len(t, items, 1)
	return items[0]
}

func (r *rig) dispatch(t *testing.T) time.Duration {
	t.Helper()
	wait, err := r.Dispatch(t.Context())
	require.NoError(t, err)
	return wait
}

// endRuns finishes every active trail on a ticket, as a run reaching done would.
func (r *rig) endRuns(t *testing.T, ticketID string) {
	t.Helper()
	trails, err := r.store.PlayTrails.ListTrailsByTarget(t.Context(), plays.TargetTicket, ticketID)
	require.NoError(t, err)
	for _, tr := range trails {
		if tr.State.Active() {
			tr.State = plays.TrailDone
			require.NoError(t, r.store.PlayTrails.UpdateTrail(t.Context(), tr))
		}
	}
}

func (r *rig) press(t *testing.T, playID, ticketID, starterID string) string {
	t.Helper()
	id := ids.New()
	require.NoError(t, r.store.PlayTrails.CreateTrail(t.Context(), &plays.Trail{
		ID: id, WorkspaceID: ws, PlayID: playID, PlayLabel: "pressed", TargetType: plays.TargetTicket, TargetID: ticketID,
		ProjectID: project, StarterID: starterID, Via: plays.ViaWeb, SelectedMemoryIDs: []string{}, State: plays.TrailRunning, StartedAt: r.clock,
	}))
	return id
}

func as(ctx context.Context, userID string) context.Context {
	return identity.WithActor(ctx, identity.Actor{ID: userID})
}

func statuses(items []*plays.QueueItem) []plays.QueueStatus {
	out := make([]plays.QueueStatus, 0, len(items))
	for _, it := range items {
		out = append(out, it.Status)
	}
	return out
}

func TestQueue_MatchWhileOneWaits_IsANoOp(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")

	r.unblock(t, "t-1")
	r.unblock(t, "t-1")

	it := r.only(t, "t-1")
	assert.Equal(t, []any{plays.QueueQueued, dev, plays.LevelNormal, fix.Label}, []any{it.Status, it.PersonID, it.Priority, it.PlayLabel})
	assert.True(t, r.Kicked(), "a queued run wakes the dispatcher")
}

func TestQueue_MatchOnNobody_DidntRunWithAFailedTrail(t *testing.T) {
	tests := []struct {
		name   string
		facts  func(*plays.Facts)
		setup  func(r *rig, playID string)
		reason string
	}{
		{"no developer", func(f *plays.Facts) { f.Developer = "" }, func(*rig, string) {}, "nobody to run it on: the ticket has no developer"},
		{"developer excluded from the play", func(*plays.Facts) {}, func(r *rig, playID string) { r.Exclude(dev, playID) }, "may not run Fix it here"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			fix := r.play(t, "Fix it")
			r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
			r.ticket("t-1", tt.facts)
			tt.setup(r, fix.ID)

			r.unblock(t, "t-1")

			it := r.only(t, "t-1")
			assert.Equal(t, plays.QueueDidntRun, it.Status)
			assert.Contains(t, it.Reason, tt.reason)
			trail, err := r.store.PlayTrails.GetTrail(t.Context(), it.TrailID)
			require.NoError(t, err)
			assert.Equal(t, plays.TrailFailed, trail.State)
		})
	}
}

func TestQueue_Moments_MatchTheirAutoPlays(t *testing.T) {
	stage := plays.StageReview
	mover := map[string]any{"kind": "user", "user_id": tester}
	tests := []struct {
		name    string
		moment  plays.Moment
		topic   string
		payload map[string]any
		queued  bool
	}{
		{"entered the auto play's stage", plays.MomentTicketEnteredStage, "ticket.status_changed",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project}, "from": "st-progress", "to": "st-review", "actor": mover}, true},
		{"moved within the stage", plays.MomentTicketEnteredStage, "ticket.status_changed",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project}, "from": "st-review-2", "to": "st-review", "actor": mover}, false},
		{"tester set", plays.MomentTicketTesterSet, "ticket.tester_changed",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project}, "to": "login-u-tester", "actor": mover}, true},
		{"tester cleared", plays.MomentTicketTesterSet, "ticket.tester_changed",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project}, "to": "", "actor": mover}, false},
		{"created", plays.MomentTicketCreated, "ticket.created",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project, "reporter": map[string]any{"kind": "user", "login": "login-u-tester"}}}, true},
		{"test failed", plays.MomentTicketTestFailed, "ticket.test_failed",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project}, "tester": "login-u-tester"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			r.SetStatus("st-progress", plays.StageProgress)
			r.SetStatus("st-review", plays.StageReview)
			r.SetStatus("st-review-2", plays.StageReview)
			fix := r.play(t, "Fix it")
			r.autoPlay(t, fix.ID, tt.moment, func(a *plays.AutoPlay) { a.MomentStage, a.RunOn = &stage, plays.RunOnCauser })
			r.ticket("t-1")

			r.moment(t, tt.topic, tt.payload)

			items := r.queued(t, "t-1")
			assert.Equal(t, tt.queued, len(items) == 1 && items[0].Status == plays.QueueQueued && items[0].PersonID == tester)
		})
	}
}

func TestQueue_OnceWithin_LetsOneRunPerWindow(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked, func(a *plays.AutoPlay) { a.OnceWithinMinutes = 60 })
	r.ticket("t-1")
	r.unblock(t, "t-1")
	r.dispatch(t)
	r.endRuns(t, "t-1")

	r.clock = r.clock.Add(30 * time.Minute)
	r.unblock(t, "t-1")
	assert.Len(t, r.queued(t, "t-1"), 1, "inside the hour the started run is the one")

	r.clock = r.clock.Add(31 * time.Minute)
	r.unblock(t, "t-1")
	assert.Len(t, r.queued(t, "t-1"), 2)
}

func TestDispatch_HighestPriorityFirst_OneSlotPerPerson(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked, func(a *plays.AutoPlay) {
		a.Priority = plays.Priority{Otherwise: plays.LevelLow, Rules: []plays.PriorityRule{{
			Level: plays.LevelHigh, When: plays.Group{Match: plays.MatchAll, Rules: []plays.Rule{{Field: plays.FieldType, Op: plays.OpIs, Values: []string{"Bug"}}}},
		}}}
	})
	r.ticket("t-task")
	r.ticket("t-bug", func(f *plays.Facts) { f.Type = "bug" })
	r.unblock(t, "t-task")
	r.clock = r.clock.Add(time.Minute)
	r.unblock(t, "t-bug")

	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-bug").Status, "high goes before an older low")
	assert.Equal(t, plays.QueueQueued, r.only(t, "t-task").Status, "the developer's one slot is taken")

	r.endRuns(t, "t-bug")
	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-task").Status)
}

func TestDispatch_PressedRunTakesTheSlot(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.press(t, fix.ID, "t-other", dev)
	r.unblock(t, "t-1")

	r.dispatch(t)
	assert.Equal(t, plays.QueueQueued, r.only(t, "t-1").Status)
	assert.Zero(t, r.Resolves(), "nothing was tried while the slot is taken")
}

func TestDispatch_OneRunPerTicket(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	review := r.play(t, "Review it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.autoPlay(t, review.ID, plays.MomentTicketUnblocked, func(a *plays.AutoPlay) { a.RunOn = plays.RunOnTester })
	r.ticket("t-1")
	r.unblock(t, "t-1")

	r.dispatch(t)
	items := r.queued(t, "t-1")
	assert.ElementsMatch(t, []plays.QueueStatus{plays.QueueStarted, plays.QueueQueued}, statuses(items))
	for _, it := range items {
		if it.Status == plays.QueueQueued {
			assert.Equal(t, plays.ReasonTicketBusy, it.Reason)
		}
	}

	r.endRuns(t, "t-1")
	r.dispatch(t)
	assert.Equal(t, []plays.QueueStatus{plays.QueueStarted, plays.QueueStarted}, statuses(r.queued(t, "t-1")))
}

func TestDispatch_RecheckAtTheFront(t *testing.T) {
	tests := []struct {
		name   string
		change func(r *rig, t *testing.T, a *plays.AutoPlay, p *plays.Play)
		status plays.QueueStatus
		reason string
	}{
		{"still matches, outside the play's show-when stage", func(*rig, *testing.T, *plays.AutoPlay, *plays.Play) {}, plays.QueueStarted, ""},
		{"blocked again", func(r *rig, _ *testing.T, _ *plays.AutoPlay, _ *plays.Play) {
			r.ticket("t-1", func(f *plays.Facts) { f.Blocked = true })
		}, plays.QueueSkipped, "no longer unblocked"},
		{"developer changed", func(r *rig, _ *testing.T, _ *plays.AutoPlay, _ *plays.Play) {
			r.ticket("t-1", func(f *plays.Facts) { f.Developer = tester })
		}, plays.QueueSkipped, "the developer changed"},
		{"auto play switched off", func(r *rig, t *testing.T, a *plays.AutoPlay, _ *plays.Play) {
			a.Enabled = false
			require.NoError(t, r.store.Plays.UpdateAutoPlay(t.Context(), a))
		}, plays.QueueSkipped, "its auto play was switched off"},
		{"conditions no longer hold", func(r *rig, t *testing.T, a *plays.AutoPlay, _ *plays.Play) {
			a.Conditions = plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{{Match: plays.MatchAll, Rules: []plays.Rule{{Field: plays.FieldLabel, Op: plays.OpSet}}}}}
			require.NoError(t, r.store.Plays.UpdateAutoPlay(t.Context(), a))
		}, plays.QueueSkipped, "its conditions no longer hold"},
		{"play disabled", func(r *rig, t *testing.T, _ *plays.AutoPlay, p *plays.Play) {
			p.Enabled = false
			require.NoError(t, r.store.Plays.Update(t.Context(), p))
		}, plays.QueueSkipped, "is disabled"},
		{"pressed while it waited", func(r *rig, t *testing.T, _ *plays.AutoPlay, p *plays.Play) {
			r.press(t, p.ID, "t-1", tester)
			r.endRuns(t, "t-1")
		}, plays.QueueSkipped, "already ran"},
		{"developer excluded since", func(r *rig, _ *testing.T, _ *plays.AutoPlay, p *plays.Play) {
			r.Exclude(dev, p.ID)
		}, plays.QueueDidntRun, "plays:run required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			fix := r.play(t, "Fix it")
			a := r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
			r.ticket("t-1")
			r.unblock(t, "t-1")
			tt.change(r, t, a, fix)

			r.dispatch(t)

			it := r.only(t, "t-1")
			assert.Equal(t, tt.status, it.Status)
			assert.Contains(t, it.Reason, tt.reason)
		})
	}
}

func TestDispatch_OfflineComputer_WaitsAMinuteWithoutFailing(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	r.Offline(true)

	wait := r.dispatch(t)

	it := r.only(t, "t-1")
	assert.Equal(t, []any{plays.QueueQueued, plays.ReasonOffline, ""}, []any{it.Status, it.Reason, it.TrailID})
	assert.Equal(t, time.Minute, wait)
	trails, err := r.store.PlayTrails.ListTrailsByTarget(t.Context(), plays.TargetTicket, "t-1")
	require.NoError(t, err)
	assert.Empty(t, trails, "an offline computer queues, it never fails")

	r.dispatch(t)
	assert.Equal(t, 1, r.Resolves(), "not tried again before the minute is up")

	r.clock = r.clock.Add(time.Minute)
	r.dispatch(t)
	assert.Equal(t, []any{2, 1}, []any{r.Resolves(), r.updates(t)}, "a retry that waits again tells nobody")

	r.Offline(false)
	r.clock = r.clock.Add(time.Minute)
	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-1").Status)
	assert.Equal(t, 2, r.updates(t))
}

// updates counts the play.queue_updated events written so far.
func (r *rig) updates(t *testing.T) int {
	t.Helper()
	pending, err := r.store.Outbox.Unpublished(t.Context(), 1000)
	require.NoError(t, err)
	n := 0
	for _, e := range pending {
		if e.Topic == plays.TopicQueueUpdated {
			n++
		}
	}
	return n
}

func TestDispatch_SetupRefusal_DidntRunWithTheFailedTrail(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	r.Refuse("setup_required")

	r.dispatch(t)

	it := r.only(t, "t-1")
	assert.Equal(t, plays.QueueDidntRun, it.Status, "only a person can fix their setup, so it never waits on it")
	trail, err := r.store.PlayTrails.GetTrail(t.Context(), it.TrailID)
	require.NoError(t, err)
	assert.Equal(t, []any{plays.TrailFailed, "setup_required"}, []any{trail.State, trail.FailureReason})
	require.True(t, r.Kicked(), "the match woke the dispatcher")
	require.False(t, r.Kicked())
	require.NoError(t, r.Runner.HandleRunFinished(t.Context(), eventbus.Event{}))
	assert.True(t, r.Kicked(), "a run ending wakes it too")
}

func TestDispatch_DailyCap_PausesUntilResumed(t *testing.T) {
	r := newRig(t, openStore(t))
	require.NoError(t, r.store.Plays.SetAutoPlayDailyCap(t.Context(), ws, 2))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	start := r.clock
	run := func() {
		r.unblock(t, "t-1")
		r.dispatch(t)
		r.endRuns(t, "t-1")
		r.clock = r.clock.Add(time.Hour)
	}
	run()
	_, err := r.Runner.ResumeAutoPlays(as(t.Context(), dev), plays.TargetTicket, "t-1")
	require.ErrorIs(t, err, apperrs.ErrConflict, "nothing to resume under the cap")
	run()

	r.unblock(t, "t-1")
	wait := r.dispatch(t)

	q, err := r.Runner.GetQueue(as(t.Context(), dev), plays.TargetTicket, "t-1")
	require.NoError(t, err)
	assert.Equal(t, []any{true, 2, 2}, []any{q.Paused, q.AutoRuns, q.DailyCap})
	assert.Equal(t, []any{plays.QueueQueued, plays.ReasonPaused}, []any{q.Items[0].Status, q.Items[0].Reason})
	assert.Equal(t, start.Add(24*time.Hour).Sub(r.clock), wait, "wakes when the oldest run leaves the day")
	require.NotNil(t, q.PausedUntil)
	assert.Equal(t, start.Add(24*time.Hour), q.PausedUntil.UTC(), "the page looks again when the oldest run leaves the day")

	_, err = r.Runner.ResumeAutoPlays(as(t.Context(), "u-stranger"), plays.TargetTicket, "t-1")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	q, err = r.Runner.ResumeAutoPlays(as(t.Context(), dev), plays.TargetTicket, "t-1")
	require.NoError(t, err)
	assert.Equal(t, []any{false, 0}, []any{q.Paused, q.AutoRuns})
	assert.Nil(t, q.PausedUntil)
	assert.True(t, r.Kicked())

	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.queued(t, "t-1")[0].Status)
}

func TestQueue_Cancel(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	id := r.only(t, "t-1").ID

	_, err := r.Runner.CancelQueued(as(t.Context(), tester), id)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "not the person it runs on, and no autoplays:write")
	r.GrantWrite(tester)
	it, err := r.Runner.CancelQueued(as(t.Context(), tester), id)
	require.NoError(t, err)
	assert.Equal(t, plays.QueueCancelled, it.Status)
	_, err = r.Runner.CancelQueued(as(t.Context(), dev), id)
	require.ErrorIs(t, err, apperrs.ErrConflict)

	r.dispatch(t)
	assert.Zero(t, r.Resolves())
}

func TestQueuedForPlay_ListsWhatWaitsWhereTheCallerMayLook(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	other := r.play(t, "Other")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.autoPlay(t, other.ID, plays.MomentTicketUnblocked)
	for _, id := range []string{"t-1", "t-2"} {
		r.ticket(id)
		r.unblock(t, id)
	}
	cancelled := r.queued(t, "t-1")
	for _, it := range cancelled {
		if it.PlayID == fix.ID {
			_, err := r.Runner.CancelQueued(as(t.Context(), dev), it.ID)
			require.NoError(t, err)
		}
	}
	ctx := as(t.Context(), dev)

	_, err := r.Runner.QueuedForPlay(ctx, fix.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "autoplays:read is required")
	r.GrantRead(dev)
	_, err = r.Runner.QueuedForPlay(ctx, " ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = r.Runner.QueuedForPlay(ctx, "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	items, err := r.Runner.QueuedForPlay(ctx, fix.ID)
	require.NoError(t, err)
	require.Len(t, items, 1, "the cancelled run and the other play's runs are left out")
	assert.Equal(t, []any{"t-2", plays.QueueQueued}, []any{items[0].TargetID, items[0].Status})

	r.HideProject(dev, project)
	items, err = r.Runner.QueuedForPlay(ctx, fix.ID)
	require.NoError(t, err)
	assert.Empty(t, items, "a project the caller may not open shows nothing")
}

// The loop recovers a start a crash cut short, starts what is due, and wakes on its own for an offline retry.
func TestRunQueue_RecoversAtBootAndRetriesOffline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := openStore(t)
		r := newRigOn(&rig{clock: time.Now().UTC()}, store, time.Now)
		fix := r.play(t, "Fix it")
		a := r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
		r.ticket("t-started")
		r.ticket("t-cut")
		trailID := r.press(t, fix.ID, "t-started", tester)
		for _, target := range []struct{ id, trail string }{{"t-started", trailID}, {"t-cut", ids.New()}} {
			_, err := store.PlayQueue.EnqueueRun(t.Context(), &plays.QueueItem{
				ID: ids.New(), WorkspaceID: ws, ProjectID: project, TargetType: plays.TargetTicket, TargetID: target.id,
				PlayID: fix.ID, PlayLabel: fix.Label, AutoPlayID: a.ID, PersonID: dev, RunOn: plays.RunOnDeveloper,
				Moment: plays.MomentTicketUnblocked, Priority: plays.LevelNormal, Status: plays.QueueDispatching, TrailID: target.trail,
				Via: plays.ViaWeb, QueuedAt: r.clock.Add(-time.Hour), NotBefore: r.clock.Add(-time.Hour),
			})
			require.NoError(t, err)
		}
		r.Offline(true)
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			defer close(done)
			r.Runner.RunQueue(ctx)
		}()

		synctest.Wait()
		assert.Equal(t, plays.QueueStarted, r.only(t, "t-started").Status, "its trail exists, so it had started")
		cut := r.only(t, "t-cut")
		assert.Equal(t, []any{plays.QueueQueued, plays.ReasonOffline}, []any{cut.Status, cut.Reason})

		r.Offline(false)
		time.Sleep(time.Minute)
		synctest.Wait()
		assert.Equal(t, plays.QueueStarted, r.only(t, "t-cut").Status, "tried again a minute later, with no kick")

		cancel()
		<-done
	})
}

func tool(t *testing.T, r *rig, ctx context.Context, name, args string) (any, error) {
	t.Helper()
	for _, tl := range plays.RunMCPTools(r.Runner) {
		if tl.Name == name {
			return tl.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func TestQueueSurfaces_MCP(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	ctx := as(t.Context(), dev)

	out, err := tool(t, r, ctx, "trail_list", `{"target_type":"ticket","target_id":"t-1","queue":true}`)
	require.NoError(t, err)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	var page struct {
		Items []plays.QueueItem `json:"items"`
		Total int               `json:"total"`
		Pause bool              `json:"paused"`
		Cap   int               `json:"daily_cap"`
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	require.Len(t, page.Items, 1)
	assert.Equal(t, []any{1, false, 5, plays.QueueQueued}, []any{page.Total, page.Pause, page.Cap, page.Items[0].Status})

	_, err = tool(t, r, ctx, "trail_update", `{"id":"`+page.Items[0].ID+`","cancel":true,"stop":true}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "cancel takes no other steer")
	out, err = tool(t, r, ctx, "trail_update", `{"id":"`+page.Items[0].ID+`","cancel":true}`)
	require.NoError(t, err)
	assert.Equal(t, plays.QueueCancelled, out.(*plays.QueueItem).Status)

	_, err = tool(t, r, ctx, "play_run", `{"resume_auto_plays":true,"play_id":"`+fix.ID+`","target_type":"ticket","target_id":"t-1"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "resume takes only the target")
	_, err = tool(t, r, ctx, "play_run", `{"resume_auto_plays":true,"target_type":"ticket","target_id":"t-1"}`)
	require.ErrorIs(t, err, apperrs.ErrConflict, "nothing is paused")
	_, err = tool(t, r, as(t.Context(), "u-stranger"), "trail_list", `{"target_type":"ticket","target_id":"t-1","queue":true}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestQueueSurfaces_HTTP(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	routes := plays.NewRunHandler(r.Runner, nil).Routes()
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(as(t.Context(), dev), method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := call(http.MethodGet, "/api/plays/queue?target_type=ticket&target_id=t-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var q plays.Queue
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &q))
	require.Len(t, q.Items, 1)

	assert.Equal(t, http.StatusBadRequest, call(http.MethodGet, "/api/plays/queue?target_type=interview&target_id=p-1", "").Code)
	assert.Equal(t, http.StatusForbidden, call(http.MethodGet, "/api/plays/queued?play_id="+fix.ID, "").Code)
	r.GrantRead(dev)
	rec = call(http.MethodGet, "/api/plays/queued?play_id="+fix.ID, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var waiting []plays.QueueItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &waiting))
	assert.Len(t, waiting, 1)
	assert.Equal(t, http.StatusConflict, call(http.MethodPost, "/api/plays/queue/resume", `{"target_type":"ticket","target_id":"t-1"}`).Code)
	assert.Equal(t, http.StatusOK, call(http.MethodPost, "/api/plays/queue/"+q.Items[0].ID+"/cancel", "").Code)
	assert.Equal(t, http.StatusNotFound, call(http.MethodPost, "/api/plays/queue/missing/cancel", "").Code)
}

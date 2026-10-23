package plays_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
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
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
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

func openStore(t *testing.T) *storage.Store { return migrated.Store(t) }

// newRig runs on a clock the test moves; newRigOn takes another, such as synctest's.
func newRig(t *testing.T, store *storage.Store) *rig {
	r := &rig{clock: time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)}
	return newRigOn(r, store, func() time.Time { return r.clock })
}

func newRigOn(r *rig, store *storage.Store, now func() time.Time) *rig {
	r.store = store
	r.QueueRig = plays.NewQueueRig(store.Plays, store.PlayTrails, store.PlayQueue, ws, project, now)
	r.runsAtOnce(1)
	r.Grant(dev)
	r.Grant(tester)
	return r
}

// runsAtOnce gives each person's computer that many runs at once, counting the trails going on it as pairing does.
func (r *rig) runsAtOnce(n int) {
	r.CountRunningWith(n, func(ctx context.Context, computerID string) (int, error) {
		running, err := r.store.Pairing.CountRunning(ctx, []string{computerID})
		return running[computerID], err
	})
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
		ComputerID: "c-" + starterID,
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

func TestQueue_MomentFromAnMCPMove_RecordsViaMCP(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")

	r.moment(t, "ticket.unblocked", map[string]any{"ticket_id": "t-1", "project_id": project, "actor": map[string]any{"kind": "user:mcp", "user_id": tester}})

	assert.Equal(t, plays.ViaMCP, r.only(t, "t-1").Via)
}

func TestQueue_MatchOnNobody_DidntRunWithAFailedTrail(t *testing.T) {
	tests := []struct {
		name    string
		facts   func(*plays.Facts)
		setup   func(r *rig, playID string)
		reason  string
		starter string
	}{
		{"no developer", func(f *plays.Facts) { f.Developer = "" }, func(*rig, string) {}, "nobody to run it on: the ticket has no developer", ""},
		{"developer excluded from the play", func(*plays.Facts) {}, func(r *rig, playID string) { r.Exclude(dev, playID) }, "may not run Fix it here", dev},
		// A starter always reads their run, so a developer who can't see the play starts none.
		{"developer can't see the play", func(*plays.Facts) {}, func(r *rig, playID string) { r.Hide(dev, playID) }, "may not run Fix it here", ""},
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
			assert.Equal(t, tt.starter, trail.StarterID)
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

func TestQueue_PersonSetAfterEnteringTheStage_RunsTheStagesAutoPlay(t *testing.T) {
	progress := plays.StageProgress
	onlyDev := func(a *plays.AutoPlay) {
		a.MomentStage = &progress
		a.Conditions.Groups = []plays.Group{{Match: plays.MatchAll, Rules: []plays.Rule{{Field: plays.FieldDeveloper, Op: plays.OpIs, Values: []string{dev}}}}}
	}
	anyDev := func(a *plays.AutoPlay) { a.MomentStage = &progress }
	tests := []struct {
		name     string
		edit     func(*plays.AutoPlay)
		stage    plays.Stage
		from, to string
		want     []plays.QueueStatus
	}{
		{"the condition holds only once set", onlyDev, plays.StageProgress, "", dev, []plays.QueueStatus{plays.QueueQueued}},
		{"changed to the one the condition names", onlyDev, plays.StageProgress, tester, dev, []plays.QueueStatus{plays.QueueQueued}},
		{"no developer to run on until set", anyDev, plays.StageProgress, "", dev, []plays.QueueStatus{plays.QueueDidntRun, plays.QueueQueued}},
		{"it could already run before", anyDev, plays.StageProgress, tester, dev, []plays.QueueStatus{plays.QueueQueued}},
		{"the ticket is in another stage", onlyDev, plays.StageBacklog, "", dev, []plays.QueueStatus{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			r.SetStatus("st-backlog", plays.StageBacklog)
			r.SetStatus("st-progress", plays.StageProgress)
			fix := r.play(t, "Fix it")
			r.autoPlay(t, fix.ID, plays.MomentTicketEnteredStage, tt.edit)
			r.ticket("t-1", func(f *plays.Facts) { f.Stage, f.Developer = tt.stage, tt.from })
			ticket := map[string]any{"id": "t-1", "project_id": project}
			mover := map[string]any{"kind": "user", "user_id": tester}
			if tt.stage == plays.StageProgress {
				r.moment(t, "ticket.status_changed", map[string]any{"ticket": ticket, "from": "st-backlog", "to": "st-progress", "actor": mover})
			}

			r.ticket("t-1", func(f *plays.Facts) { f.Stage, f.Developer = tt.stage, tt.to })
			from := ""
			if tt.from != "" {
				from = "login-" + tt.from
			}
			r.moment(t, "ticket.developer_changed", map[string]any{"ticket": ticket, "from": from, "to": "login-" + tt.to, "actor": mover})

			items := r.queued(t, "t-1")
			assert.Len(t, items, len(tt.want))
			assert.ElementsMatch(t, tt.want, statuses(items))
			for _, it := range items {
				if it.Status == plays.QueueQueued && tt.from != tester {
					assert.Equal(t, dev, it.PersonID, "the run lands on the developer just set")
				}
			}
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
	assert.Equal(t, plays.QueueQueued, r.only(t, "t-task").Status, "the developer's computer takes one run at once")

	r.endRuns(t, "t-bug")
	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-task").Status)
}

func TestDispatch_RunsAtOnce_LetsThatManyGoSideBySide(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.runsAtOnce(2)
	r.press(t, fix.ID, "t-pressed", dev)
	for _, id := range []string{"t-1", "t-2", "t-3"} {
		r.ticket(id)
		r.unblock(t, id)
		r.clock = r.clock.Add(time.Second)
	}

	r.dispatch(t)
	assert.Equal(t, []any{plays.QueueStarted, plays.QueueQueued, "no_computer_ready"},
		[]any{r.only(t, "t-1").Status, r.only(t, "t-2").Status, r.only(t, "t-3").Reason},
		"the pressed run takes one of the computer's two runs at once, the oldest auto run the other")
	waiting := r.only(t, "t-3").WaitingOn
	require.Len(t, waiting, 1)
	assert.Equal(t, []any{"c-u-dev", "full", "c-u-dev is full (2 of 2 running)"}, []any{waiting[0].ComputerID, waiting[0].Why, waiting[0].Text},
		"a waiting run names the computer and why")

	r.endRuns(t, "t-pressed")
	r.dispatch(t)
	assert.Equal(t, []any{plays.QueueStarted, plays.QueueQueued}, []any{r.only(t, "t-2").Status, r.only(t, "t-3").Status})
	assert.Empty(t, r.only(t, "t-2").WaitingOn, "a started run waits on nothing")
}

func TestDispatch_AutoPlayMemories_TheRunReadsThoseInItsProject(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked, func(a *plays.AutoPlay) {
		a.MemoryIDs = []string{"m-elsewhere", "m-here"}
	})
	r.SetMemories(project, "m-here", "m-unpicked")
	r.SetMemories("other-project", "m-elsewhere")
	r.ticket("t-1")
	r.unblock(t, "t-1")

	r.dispatch(t)
	require.Equal(t, plays.QueueStarted, r.only(t, "t-1").Status, "a pick from another project never stops the run")
	trails, err := r.store.PlayTrails.ListTrailsByTarget(t.Context(), plays.TargetTicket, "t-1")
	require.NoError(t, err)
	require.Len(t, trails, 1)
	assert.Equal(t, []string{"m-here"}, trails[0].SelectedMemoryIDs)
}

func TestDispatch_PressedRunTakesTheSlot(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.press(t, fix.ID, "t-other", dev)
	r.unblock(t, "t-1")

	r.dispatch(t)
	it := r.only(t, "t-1")
	assert.Equal(t, []any{plays.QueueQueued, "no_computer_ready"}, []any{it.Status, it.Reason}, "the run says why it waits")
	assert.Zero(t, r.Resolves(), "nothing was started while the computer is full")
	assert.Equal(t, 1, r.updates(t))

	r.dispatch(t)
	assert.Equal(t, 1, r.updates(t), "a pass over a person still busy writes nothing")

	r.endRuns(t, "t-other")
	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-1").Status)
}

func (r *rig) setArchived(t *testing.T, archived bool) {
	t.Helper()
	var at *time.Time
	if archived {
		at = &r.clock
	}
	require.NoError(t, r.store.Workspaces.SetArchived(t.Context(), ws, at, r.clock))
}

// A run queued before the archive waits rather than ending, and starts after a restore.
func TestArchivedWorkspace_AutoPlaysMatchAndStartNothingUntilRestored(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	for _, id := range []string{"t-waiting", "t-archived", "t-restored"} {
		r.ticket(id)
	}
	r.unblock(t, "t-waiting")

	r.setArchived(t, true)
	r.unblock(t, "t-archived")
	assert.Empty(t, r.queued(t, "t-archived"), "its auto plays match nothing")
	assert.Negative(t, r.dispatch(t), "the queue has nothing to start or wake for")
	assert.Equal(t, plays.QueueQueued, r.only(t, "t-waiting").Status, "a run queued earlier waits")
	assert.Zero(t, r.Resolves(), "and no computer was asked to start it")

	r.setArchived(t, false)
	r.clock = r.clock.Add(time.Minute)
	r.unblock(t, "t-restored")
	assert.Len(t, r.queued(t, "t-restored"), 1, "auto plays match again")
	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-waiting").Status, "and the waiting run starts")
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

func TestQueuedTargets_NamesTicketsWithARunWaiting_ToThoseWhoSeeIt(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	review := r.play(t, "Review it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.autoPlay(t, review.ID, plays.MomentTicketUnblocked, func(a *plays.AutoPlay) { a.RunOn = plays.RunOnTester })
	r.ticket("t-1")
	r.ticket("t-2")
	r.unblock(t, "t-1")
	r.dispatch(t)
	items := r.queued(t, "t-1")
	waiting := items[slices.IndexFunc(items, func(it *plays.QueueItem) bool { return it.Status == plays.QueueQueued })]

	got, err := r.Runner.QueuedTargets(as(t.Context(), dev), plays.TargetTicket, []string{"t-1", "t-2"})
	require.NoError(t, err)
	require.Len(t, got, 1, "a started run is not waiting, and t-2 has none")
	assert.Equal(t, waiting.ID, got["t-1"].ID)

	got, err = r.Runner.QueuedTargets(as(t.Context(), "stranger"), plays.TargetTicket, []string{"t-1"})
	require.NoError(t, err)
	assert.Empty(t, got, "a caller who cannot see the play gets no clock")

	r.HideProject(tester, project)
	got, err = r.Runner.QueuedTargets(as(t.Context(), tester), plays.TargetTicket, []string{"t-1"})
	require.NoError(t, err)
	assert.Empty(t, got, "the run's own person loses it with the project")
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

func TestDispatch_SetupGap_WaitsWithoutAFailedTrailUntilFixed(t *testing.T) {
	for _, reason := range []string{"unpaired", "expired_token", "setup_required", "offline", "no_computer_ready"} {
		t.Run(reason, func(t *testing.T) {
			r := newRig(t, openStore(t))
			fix := r.play(t, "Fix it")
			r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
			r.ticket("t-1")
			r.unblock(t, "t-1")
			r.Refuse(reason)

			wait := r.dispatch(t)

			it := r.only(t, "t-1")
			assert.Equal(t, []any{plays.QueueQueued, reason, ""}, []any{it.Status, it.Reason, it.TrailID})
			assert.Equal(t, time.Minute, wait, "tried again a minute later")
			trails, err := r.store.PlayTrails.ListTrailsByTarget(t.Context(), plays.TargetTicket, "t-1")
			require.NoError(t, err)
			assert.Empty(t, trails, "a setup gap waits, it never fails")

			r.Refuse("")
			r.clock = r.clock.Add(time.Minute)
			r.dispatch(t)
			assert.Equal(t, plays.QueueStarted, r.only(t, "t-1").Status, "starts once the gap is fixed")
		})
	}
}

func TestHandleComputerChanged_RetriesTheHeldRunsAtOnce(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	r.Refuse("unpaired")
	r.dispatch(t)
	r.Refuse("")
	r.Kicked()

	require.NoError(t, r.Runner.HandleComputerChanged(t.Context(), eventbus.Event{ID: ids.New(), Topic: "computer.paired", Payload: []byte(`{"user_id":"u-tester"}`)}))
	r.dispatch(t)
	assert.Equal(t, plays.QueueQueued, r.only(t, "t-1").Status, "another person's computer wakes nothing of theirs")

	require.NoError(t, r.Runner.HandleComputerChanged(t.Context(), eventbus.Event{ID: ids.New(), Topic: "computer.paired", Payload: []byte(`{"user_id":"u-dev"}`)}))
	assert.True(t, r.Kicked())
	r.dispatch(t)
	assert.Equal(t, plays.QueueStarted, r.only(t, "t-1").Status, "started without waiting out the minute")

	err := r.Runner.HandleComputerChanged(t.Context(), eventbus.Event{ID: ids.New(), Topic: "computer.paired", Payload: []byte(`not json`)})
	require.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestDispatch_DailyCap_PausesUntilResumed(t *testing.T) {
	r := newRig(t, openStore(t))
	require.NoError(t, r.store.Plays.SetAutoPlayLimits(t.Context(), ws, plays.AutoPlayLimits{DailyCapPerTicket: 2}))
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
	assert.EqualError(t, err, "forbidden: only the person it runs on or an autoplays:write holder can cancel a queued run")
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
	out, err = tool(t, r, as(t.Context(), "u-stranger"), "trail_list", `{"target_type":"ticket","target_id":"t-1","queue":true}`)
	require.NoError(t, err)
	raw, err = json.Marshal(out)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"items":[]`, "auto runs of a play the caller can't see are left out")
}

// TestQueue_WhyEachComputerWaits_IsItsPersonsAlone guards ADR 0165's privacy: a computer is its owner's, so the
// reasons a run waits on each one reach the person it runs on, never a teammate reading the ticket or an event.
func TestQueue_WhyEachComputerWaits_IsItsPersonsAlone(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.press(t, fix.ID, "t-pressed", dev)
	r.ticket("t-1")
	r.unblock(t, "t-1")
	r.dispatch(t)

	read := func(userID string) []plays.PassedOver {
		q, err := r.Runner.GetQueue(as(t.Context(), userID), plays.TargetTicket, "t-1")
		require.NoError(t, err)
		require.Len(t, q.Items, 1)
		return q.Items[0].WaitingOn
	}
	assert.Len(t, read(dev), 1, "the developer the run waits on reads why")
	assert.Empty(t, read(tester), "a teammate sees the run wait, not their computers")
	pending, err := r.store.Outbox.Unpublished(t.Context(), 1000)
	require.NoError(t, err)
	for _, e := range pending {
		if e.Topic == plays.TopicQueueUpdated {
			assert.NotContains(t, string(mustJSON(t, e.Payload)), "waiting_on", "an event never names a computer")
		}
	}
}

func TestTrails_TheComputersARunPassedOver_AreItsStartersAlone(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.ticket("t-1")
	passed := []plays.PassedOver{{ComputerID: "c-laptop", Name: "laptop", Why: "offline", Text: "laptop is offline"}}
	require.NoError(t, r.store.PlayTrails.CreateTrail(t.Context(), &plays.Trail{
		ID: "tr-1", WorkspaceID: ws, PlayID: fix.ID, PlayLabel: "Fix it", TargetType: plays.TargetTicket, TargetID: "t-1", ProjectID: project,
		StarterID: dev, Via: plays.ViaWeb, SelectedMemoryIDs: []string{}, State: plays.TrailRunning, StartedAt: r.clock, PassedOver: passed,
	}))

	mine, err := r.Runner.GetTrail(as(t.Context(), dev), "tr-1")
	require.NoError(t, err)
	assert.Equal(t, passed, mine.PassedOver)
	theirs, err := r.Runner.ListTrails(as(t.Context(), tester), plays.TargetTicket, "t-1", "")
	require.NoError(t, err)
	require.Len(t, theirs, 1)
	assert.Empty(t, theirs[0].PassedOver, "a teammate reads the run, not the starter's computers")
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
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

// decisionsCheck is the migrated workspace's seeded decisions check, its auto play switched on.
func (r *rig) decisionsCheck(t *testing.T) *plays.Play {
	t.Helper()
	list, err := r.store.Plays.List(t.Context(), ws)
	require.NoError(t, err)
	var check *plays.Play
	for _, p := range list {
		if p.BuiltinKey == plays.DecisionsCheckKey {
			check = p
		}
	}
	require.NotNil(t, check)
	autoPlays, err := r.store.Plays.ListAutoPlays(t.Context(), []string{check.ID})
	require.NoError(t, err)
	require.Len(t, autoPlays, 1)
	autoPlays[0].Enabled = true
	require.NoError(t, r.store.Plays.UpdateAutoPlay(t.Context(), autoPlays[0]))
	return check
}

func TestDecisionsCheck_TicketEnteringDone_QueuesOneCheckForTheCauser(t *testing.T) {
	tests := []struct {
		name  string
		actor map[string]any
		want  string
		via   plays.Via
	}{
		{"a person moves it", map[string]any{"kind": "user", "user_id": tester}, tester, plays.ViaWeb},
		{"a person moves it over MCP", map[string]any{"kind": "user:mcp", "user_id": tester}, tester, plays.ViaMCP},
		{"an automation moves it, so its developer runs it", map[string]any{"kind": "automation"}, dev, plays.ViaWeb},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			r.SetStatus("st-progress", plays.StageProgress)
			r.SetStatus("st-done", plays.StageDone)
			r.SetStatus("st-shipped", plays.StageDone)
			check := r.decisionsCheck(t)
			r.ticket("t-1", func(f *plays.Facts) { f.Stage = plays.StageDone })
			move := func(from, to string) eventbus.Event {
				raw, err := json.Marshal(map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": project}, "from": from, "to": to, "actor": tt.actor})
				require.NoError(t, err)
				return eventbus.Event{ID: ids.New(), Topic: "ticket.status_changed", Payload: raw}
			}
			entered := move("st-progress", "st-done")

			require.NoError(t, r.Runner.HandleAutoPlayMoment(t.Context(), entered))
			require.NoError(t, r.Runner.HandleAutoPlayMoment(t.Context(), entered), "a redelivery while it waits")
			r.dispatch(t)
			require.NoError(t, r.Runner.HandleAutoPlayMoment(t.Context(), entered), "a redelivery after it started")
			require.NoError(t, r.Runner.HandleAutoPlayMoment(t.Context(), move("st-done", "st-shipped")), "a move between two done columns")

			it := r.only(t, "t-1")
			assert.Equal(t, []any{check.ID, tt.want, plays.QueueStarted, tt.via}, []any{it.PlayID, it.PersonID, it.Status, it.Via})
		})
	}
}

func TestDecisionsCheck_AutomationMoveWithoutADeveloper_DidntRunWithAFailedTrail(t *testing.T) {
	r := newRig(t, openStore(t))
	r.SetStatus("st-progress", plays.StageProgress)
	r.SetStatus("st-done", plays.StageDone)
	check := r.decisionsCheck(t)
	r.ticket("t-1", func(f *plays.Facts) { f.Stage, f.Developer = plays.StageDone, "" })

	r.moment(t, "ticket.status_changed", map[string]any{
		"ticket": map[string]any{"id": "t-1", "project_id": project}, "from": "st-progress", "to": "st-done", "actor": map[string]any{"kind": "automation"},
	})

	it := r.only(t, "t-1")
	assert.Equal(t, []any{plays.QueueDidntRun, "nobody to run it on: no person caused it and the ticket has no developer"}, []any{it.Status, it.Reason})
	trail, err := r.store.PlayTrails.GetTrail(t.Context(), it.TrailID)
	require.NoError(t, err)
	assert.Equal(t, []any{check.ID, plays.TrailFailed}, []any{trail.PlayID, trail.State}, "the ticket's notice offers to run it again")
}

func TestHandleAutoPlayOff_SkipsWhatItStillHasQueued(t *testing.T) {
	tests := []struct {
		name   string
		topic  string
		event  func(a *plays.AutoPlay) map[string]any
		reason string
	}{
		{"switched off", plays.TopicAutoPlayUpdated, func(a *plays.AutoPlay) map[string]any {
			off := *a
			off.Enabled = false
			return map[string]any{"auto_play": off}
		}, "its auto play was switched off"},
		{"deleted", plays.TopicAutoPlayDeleted, func(a *plays.AutoPlay) map[string]any {
			return map[string]any{"id": a.ID, "play_id": a.PlayID, "workspace_id": ws}
		}, "its auto play was removed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			fix := r.play(t, "Fix it")
			a := r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
			other := r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked, func(o *plays.AutoPlay) { o.RunOn = plays.RunOnTester })
			r.ticket("t-1")
			r.Offline(true)
			r.unblock(t, "t-1")
			r.dispatch(t)
			require.Equal(t, []plays.QueueStatus{plays.QueueQueued, plays.QueueQueued}, statuses(r.queued(t, "t-1")))

			raw, err := json.Marshal(tt.event(a))
			require.NoError(t, err)
			require.NoError(t, r.Runner.HandleAutoPlayOff(t.Context(), eventbus.Event{ID: ids.New(), Topic: tt.topic, Payload: raw}))

			for _, it := range r.queued(t, "t-1") {
				if it.AutoPlayID == a.ID {
					assert.Equal(t, []any{plays.QueueSkipped, tt.reason}, []any{it.Status, it.Reason})
					continue
				}
				assert.Equal(t, other.ID, it.AutoPlayID)
				assert.Equal(t, plays.QueueQueued, it.Status, "another auto play's run keeps waiting")
			}
		})
	}
}

func TestHandleAutoPlayOff_StaysOutOfTheWayOfAnAutoPlayStillOn(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	a := r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
	r.ticket("t-1")
	r.Offline(true)
	r.unblock(t, "t-1")
	r.dispatch(t)

	raw, err := json.Marshal(map[string]any{"auto_play": a})
	require.NoError(t, err)
	require.NoError(t, r.Runner.HandleAutoPlayOff(t.Context(), eventbus.Event{ID: ids.New(), Topic: plays.TopicAutoPlayUpdated, Payload: raw}))

	assert.Equal(t, plays.QueueQueued, r.only(t, "t-1").Status)
}

// computerRig has a ticket play whose auto play runs on alice's c-1, with t-1 started on it and t-2 held because it is full.
func computerRig(t *testing.T) (*rig, *plays.AutoPlay) {
	t.Helper()
	r := newRig(t, openStore(t))
	r.Grant(alice)
	r.Grant("u-bob")
	r.OwnComputer("c-1", alice)
	r.OwnComputer("c-2", alice)
	r.OwnComputer("c-3", "u-bob")
	a := r.autoPlay(t, r.play(t, "Fix it").ID, plays.MomentTicketUnblocked, onComputer("c-1"))
	for _, id := range []string{"t-1", "t-2"} {
		r.ticket(id)
		r.unblock(t, id)
		r.clock = r.clock.Add(time.Second)
		r.dispatch(t)
	}
	require.Equal(t, []any{plays.QueueStarted, plays.QueueQueued, "no_computer_ready"},
		[]any{r.only(t, "t-1").Status, r.only(t, "t-2").Status, r.only(t, "t-2").Reason})
	return r, a
}

func (r *rig) editAutoPlay(t *testing.T, a *plays.AutoPlay) {
	t.Helper()
	require.NoError(t, r.store.Plays.UpdateAutoPlay(t.Context(), a))
	raw, err := json.Marshal(plays.AutoPlayEvent{AutoPlay: *a})
	require.NoError(t, err)
	require.NoError(t, r.Runner.HandleAutoPlayComputersChanged(t.Context(), eventbus.Event{ID: ids.New(), Topic: plays.TopicAutoPlayUpdated, Payload: raw}))
}

func TestHandleAutoPlayComputersChanged_MovesAHeldRunOntoTheNewList(t *testing.T) {
	tests := []struct {
		name   string
		edit   func(a *plays.AutoPlay)
		person string
		starts bool
	}{
		{"a computer added", func(a *plays.AutoPlay) { onComputer("c-1", "c-2")(a) }, alice, false},
		{"picked again by someone else", func(a *plays.AutoPlay) { onComputer("c-3")(a); a.RunAs = "u-bob" }, "u-bob", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, a := computerRig(t)
			r.Kicked()
			r.clock = r.clock.Add(10 * time.Second)
			tt.edit(a)

			r.editAutoPlay(t, a)

			it := r.only(t, "t-2")
			assert.Equal(t, []any{a.ComputerIDs, a.ComputerID, tt.person, "", r.clock}, []any{it.ComputerIDs, it.ComputerID, it.PersonID, it.Reason, it.NotBefore},
				"the run carries the auto play's list and picker and is due now, not what it was held on")
			assert.Empty(t, it.WaitingOn)
			assert.True(t, r.Kicked(), "the dispatcher is woken to try it at once")

			r.dispatch(t)
			if tt.starts {
				assert.Equal(t, plays.QueueStarted, r.only(t, "t-2").Status, "the new computer had room")
			}
		})
	}
}

func TestHandleAutoPlayComputersChanged_LeavesTheQueueAloneWhenNothingMoves(t *testing.T) {
	tests := []struct {
		name string
		edit func(a *plays.AutoPlay)
	}{
		{"the list and picker are unchanged", func(a *plays.AutoPlay) { a.OnceWithinMinutes = 5 }},
		{"it is switched off", func(a *plays.AutoPlay) { onComputer("c-1", "c-2")(a); a.Enabled = false }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, a := computerRig(t)
			r.Kicked()
			before := r.only(t, "t-2")
			updates := r.updates(t)
			tt.edit(a)

			r.editAutoPlay(t, a)

			assert.Equal(t, before, r.only(t, "t-2"))
			assert.Equal(t, updates, r.updates(t), "nothing is announced")
			assert.False(t, r.Kicked())
		})
	}
}

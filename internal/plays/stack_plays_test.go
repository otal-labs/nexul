package plays_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
)

const alice = "u-alice"

func (r *rig) stackPlay(t *testing.T, label string) *plays.Play {
	t.Helper()
	p := &plays.Play{
		ID: ids.New(), WorkspaceID: ws, Label: label, Type: plays.TypeStack, Instructions: "Look.", Enabled: true,
		ExcludedProjectIDs: []string{}, CreatedAt: r.clock, UpdatedAt: r.clock,
	}
	require.NoError(t, r.store.Plays.Create(t.Context(), p))
	return p
}

// onComputer makes the auto play land on computer, as alice picked it.
func onComputer(computer string) func(*plays.AutoPlay) {
	return func(a *plays.AutoPlay) { a.RunOn, a.ComputerID, a.RunAs = plays.RunOnComputer, computer, alice }
}

func (r *rig) crash(t *testing.T, topic, stack, service string, projectID string) {
	t.Helper()
	r.moment(t, topic, map[string]any{
		"workspace_id": ws, "project_id": projectID, "stack_id": stack, "stack_name": "acme-web", "service": service, "outage_id": "o-1",
	})
}

func (r *rig) stackQueue(t *testing.T, stack string) []*plays.QueueItem {
	t.Helper()
	items, err := r.store.PlayQueue.ListQueueByTarget(t.Context(), plays.TargetStack, stack)
	require.NoError(t, err)
	return items
}

func newStackRig(t *testing.T, edit ...func(*plays.AutoPlay)) (*rig, *plays.AutoPlay) {
	t.Helper()
	r := newRig(t, openStore(t))
	r.Grant(alice)
	r.SetStack("st-1", project, "acme-web")
	r.OwnComputer("c-1", alice)
	play := r.stackPlay(t, "Investigate")
	edits := append([]func(*plays.AutoPlay){onComputer("c-1")}, edit...)
	return r, r.autoPlay(t, play.ID, plays.MomentStackServiceCrashed, edits...)
}

func TestStackMoment_QueuesOnTheComputerWhoeverPickedItRunsOn(t *testing.T) {
	r, _ := newStackRig(t)

	r.crash(t, "service.container_stopped", "st-1", "api", project)

	items := r.stackQueue(t, "st-1")
	require.Len(t, items, 1)
	it := items[0]
	assert.Equal(t, []any{plays.QueueQueued, alice, plays.RunOnComputer, "c-1", "api", "o-1", plays.MomentStackServiceCrashed, project},
		[]any{it.Status, it.PersonID, it.RunOn, it.ComputerID, it.Service, it.OutageID, it.Moment, it.ProjectID})
	assert.True(t, r.Kicked(), "a queued run wakes the dispatcher")

	r.dispatch(t)

	trails, err := r.store.PlayTrails.ListTrailsByTarget(t.Context(), plays.TargetStack, "st-1")
	require.NoError(t, err)
	require.Len(t, trails, 1, "the run is recorded on the stack under the person who picked the computer")
	assert.Equal(t, []any{alice, plays.TrailStarting, project}, []any{trails[0].StarterID, trails[0].State, trails[0].ProjectID})
	assert.Equal(t, "c-1", r.LastChoice().ComputerID, "it starts on the computer the auto play names")
	assert.Equal(t, plays.QueueStarted, r.stackQueue(t, "st-1")[0].Status)
	blocks := strings.Join(r.LastPlayBlocks(), "\n")
	assert.Contains(t, blocks, `Stack "acme-web" (stack id st-1)`)
	assert.Contains(t, blocks, "Service: api. Outage id o-1")
}

func TestStackMoment_ConditionsReadTheStackProjectAndService(t *testing.T) {
	rule := func(field plays.Field, op plays.Op, values ...string) func(*plays.AutoPlay) {
		return func(a *plays.AutoPlay) {
			a.Conditions = plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{{Match: plays.MatchAll, Rules: []plays.Rule{{Field: field, Op: op, Values: values}}}}}
		}
	}
	tests := []struct {
		name   string
		edit   func(*plays.AutoPlay)
		topic  string
		queued bool
	}{
		{"the service is one named", rule(plays.FieldService, plays.OpIs, "api"), "service.container_stopped", true},
		{"the service names are compared without case", rule(plays.FieldService, plays.OpIs, "API"), "service.container_stopped", true},
		{"the service is not one named", rule(plays.FieldService, plays.OpIs, "worker"), "service.container_stopped", false},
		{"the service is excluded", rule(plays.FieldService, plays.OpIsNot, "api"), "service.container_stopped", false},
		{"the stack is the one named", rule(plays.FieldStack, plays.OpIs, "st-1"), "service.container_stopped", true},
		{"the stack is another", rule(plays.FieldStack, plays.OpIs, "st-2"), "service.container_stopped", false},
		{"the project is the one named", rule(plays.FieldProject, plays.OpIs, project), "service.container_stopped", true},
		{"the project is another", rule(plays.FieldProject, plays.OpIs, "p-other"), "service.container_stopped", false},
		{"a recovery is not a crash", nil, "service.container_recovered", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var edits []func(*plays.AutoPlay)
			if tt.edit != nil {
				edits = append(edits, tt.edit)
			}
			r, _ := newStackRig(t, edits...)

			r.crash(t, tt.topic, "st-1", "api", project)

			assert.Equal(t, tt.queued, len(r.stackQueue(t, "st-1")) == 1)
		})
	}
}

func TestStackMoment_ARecoveryMatchesTheRecoveredMoment(t *testing.T) {
	r := newRig(t, openStore(t))
	r.Grant(alice)
	r.SetStack("st-1", project, "acme-web")
	r.OwnComputer("c-1", alice)
	play := r.stackPlay(t, "Note the recovery")
	r.autoPlay(t, play.ID, plays.MomentStackServiceRecovered, onComputer("c-1"))

	r.crash(t, "service.container_stopped", "st-1", "api", project)
	assert.Empty(t, r.stackQueue(t, "st-1"), "a crash is not what it waits for")
	r.crash(t, "service.container_recovered", "st-1", "api", project)

	items := r.stackQueue(t, "st-1")
	require.Len(t, items, 1)
	assert.Equal(t, plays.MomentStackServiceRecovered, items[0].Moment)
}

func TestStackMoment_NothingWaitsForAnInstanceStackOrADeletedOne(t *testing.T) {
	r, _ := newStackRig(t)

	r.crash(t, "service.container_stopped", "st-1", "api", "")
	r.crash(t, "service.container_stopped", "st-gone", "api", project)
	r.crash(t, "service.container_stopped", "", "api", project)

	assert.Empty(t, r.stackQueue(t, "st-1"), "an instance stack has no project, so no workspace's auto play waits for it")
	assert.Empty(t, r.stackQueue(t, "st-gone"))
}

func TestStackMoment_OffUntilSwitchedOn(t *testing.T) {
	r, _ := newStackRig(t, func(a *plays.AutoPlay) { a.Enabled = false })

	r.crash(t, "service.container_stopped", "st-1", "api", project)

	assert.Empty(t, r.stackQueue(t, "st-1"))
}

func TestRunOnComputer_NeedsTheRightWhoeverPickedItHolds(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(r *rig)
		reason string
	}{
		{"the owner runs on their own computer", func(*rig) {}, ""},
		{"a sharee with Run agents runs on it", func(r *rig) {
			r.OwnComputer("c-1", "u-owner")
			r.ShareComputer("c-1", alice, permissions.ComputersSee, permissions.ComputersAgents)
		}, ""},
		{"a sharee without Run agents is refused", func(r *rig) {
			r.OwnComputer("c-1", "u-owner")
			r.ShareComputer("c-1", alice, permissions.ComputersSee, permissions.ComputersLogs)
		}, "login-u-alice may not run agents on the computer picked for it"},
		{"a computer shared with nobody is refused", func(r *rig) { r.OwnComputer("c-1", "u-owner") },
			"login-u-alice may not run agents on the computer picked for it"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := newStackRig(t)
			tt.setup(r)

			r.crash(t, "service.container_stopped", "st-1", "api", project)

			it := r.stackQueue(t, "st-1")[0]
			if tt.reason == "" {
				assert.Equal(t, plays.QueueQueued, it.Status)
				return
			}
			assert.Equal(t, plays.QueueDidntRun, it.Status)
			assert.Equal(t, tt.reason, it.Reason)
			trail, err := r.store.PlayTrails.GetTrail(t.Context(), it.TrailID)
			require.NoError(t, err)
			assert.Equal(t, []any{plays.TrailFailed, tt.reason, alice}, []any{trail.State, trail.LastError, trail.StarterID},
				"the stack shows why, under the person who picked the computer")
		})
	}
}

func TestRunOnComputer_ACheckThatFailsDoesNotStartTheRun(t *testing.T) {
	r, _ := newStackRig(t)
	r.FailComputerCheck(errors.New("db down"))

	r.crash(t, "service.container_stopped", "st-1", "api", project)

	it := r.stackQueue(t, "st-1")[0]
	assert.Equal(t, []any{plays.QueueDidntRun, "could not check whether login-u-alice may run agents on the computer picked for it"}, []any{it.Status, it.Reason})
}

func TestRunOnComputer_ARightTakenAwayAfterQueuingStopsTheRun(t *testing.T) {
	r, _ := newStackRig(t)
	r.OwnComputer("c-1", "u-owner")
	r.ShareComputer("c-1", alice, permissions.ComputersSee, permissions.ComputersAgents)
	r.crash(t, "service.container_stopped", "st-1", "api", project)
	require.Equal(t, plays.QueueQueued, r.stackQueue(t, "st-1")[0].Status)

	r.ShareComputer("c-1", alice)
	r.dispatch(t)

	it := r.stackQueue(t, "st-1")[0]
	assert.Equal(t, plays.QueueDidntRun, it.Status)
	assert.Equal(t, "login-u-alice may not run agents on the computer picked for it", it.Reason)
	assert.Zero(t, r.Resolves(), "nothing was started on the computer")
	trail, err := r.store.PlayTrails.GetTrail(t.Context(), it.TrailID)
	require.NoError(t, err)
	assert.Equal(t, plays.TrailFailed, trail.State)
}

func TestRunOnComputer_AnAutoPlayPickedBySomeoneElseSinceSkipsTheQueuedRun(t *testing.T) {
	r, a := newStackRig(t)
	r.crash(t, "service.container_stopped", "st-1", "api", project)

	a.RunAs = "u-bob"
	require.NoError(t, r.store.Plays.UpdateAutoPlay(t.Context(), a))
	r.dispatch(t)

	it := r.stackQueue(t, "st-1")[0]
	assert.Equal(t, []any{plays.QueueSkipped, "its computer was picked by someone else"}, []any{it.Status, it.Reason})
}

func TestRunOnComputer_IsOfferedToEveryPlayType(t *testing.T) {
	r := newRig(t, openStore(t))
	r.Grant(alice)
	r.OwnComputer("c-1", alice)
	fix := r.play(t, "Fix it")
	r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked, onComputer("c-1"))
	r.ticket("t-1", func(f *plays.Facts) { f.Developer = "" })

	r.unblock(t, "t-1")

	it := r.only(t, "t-1")
	assert.Equal(t, []any{plays.QueueQueued, alice, "c-1"}, []any{it.Status, it.PersonID, it.ComputerID},
		"a ticket with no developer still runs, as the person who picked the computer")
}

func TestStackQueue_WaitsOnAGapAndIsReadThroughItsProject(t *testing.T) {
	r, _ := newStackRig(t)
	r.Refuse("no_default")
	r.crash(t, "service.container_stopped", "st-1", "api", project)
	r.dispatch(t)

	q, err := r.Runner.GetQueue(as(t.Context(), alice), plays.TargetStack, "st-1")
	require.NoError(t, err)
	require.Len(t, q.Items, 1)
	assert.Equal(t, []any{plays.QueueQueued, "no_default", 0}, []any{q.Items[0].Status, q.Items[0].Reason, q.AutoRuns},
		"a run whose picker has no T3 project on that computer waits on the gap instead of failing")

	r.Refuse("")
	r.clock = r.clock.Add(time.Minute)
	r.dispatch(t)
	q, err = r.Runner.GetQueue(as(t.Context(), alice), plays.TargetStack, "st-1")
	require.NoError(t, err)
	assert.Equal(t, []any{plays.QueueStarted, 1}, []any{q.Items[0].Status, q.AutoRuns})

	r.HideProject("u-stranger", project)
	_, err = r.Runner.GetQueue(as(t.Context(), "u-stranger"), plays.TargetStack, "st-1")
	assert.ErrorIs(t, err, apperrs.ErrNotFound, "someone who cannot open the project sees no queue")
}

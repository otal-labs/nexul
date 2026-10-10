package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// endedRun runs the fixture's ticket play to a failed end and returns its trail.
func endedRun(t *testing.T, f *runnerFixture) *Trail {
	t.Helper()
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("th-1")
	obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "Lost the connection to T3 Code"}, "")
	return trail
}

type continued struct {
	trail *Trail
	err   error
}

// continueAsync sends message to the trail's thread and returns the turn it opened, while Continue waits on how it opens.
func continueAsync(t *testing.T, f *runnerFixture, trailID, message string) (<-chan continued, agent.TurnRequest) {
	t.Helper()
	out := make(chan continued, 1)
	go func() {
		got, err := f.runner.Continue(ctxAs(starter), trailID, message, ViaWeb)
		out <- continued{got, err}
	}()
	<-f.turns.done
	return out, f.turns.last()
}

func TestContinue_EndedRun_SendsOnlyTheMessageToItsOwnThreadAsTheSameTrail(t *testing.T) {
	f := newRunnerFixture()
	trail := endedRun(t, f)

	done, req := continueAsync(t, f, trail.ID, "  Keep two threads.  ")
	assert.Equal(t, "Keep two threads.", req.RequestBody)
	assert.Nil(t, req.Play, "the play's instructions are not sent again")
	assert.True(t, req.KeepThread, "only the run's own thread will do")
	assert.Equal(t, trail.ConversationID, req.ConversationID)
	assert.Equal(t, starter, req.ViaUserID)
	req.Observer.OnStarted("th-1")
	got := <-done
	require.NoError(t, got.err)

	assert.Equal(t, trail.ID, got.trail.ID)
	assert.Equal(t, TrailRunning, got.trail.State)
	assert.Empty(t, got.trail.LastError)
	last := got.trail.Activity[len(got.trail.Activity)-1]
	assert.Equal(t, harness.ActivityUserMessage, last.Kind, "the transcript shows the message as the starter's")
	assert.Equal(t, "Keep two threads.", last.Detail)
	assert.Contains(t, f.threads.posts, fakePost{trail.ConversationID, starter, "Keep two threads."})
	assert.Len(t, f.trails.all(), 1, "same trail")
	assert.Empty(t, f.trails.eventsFor(TopicRunStarted)[1:], "a continued run announces no second start")
}

func TestContinue_ThreadGone_PutsTheTrailBackAndStartsThePlayAgainWithTheMessage(t *testing.T) {
	f := newRunnerFixture()
	trail := endedRun(t, f)
	f.trails.byID[trail.ID].CustomInstructions = "Be brief."

	done, req := continueAsync(t, f, trail.ID, "Keep two threads.")
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "the harness session is gone", SessionGone: true}, "")
	got := <-done
	require.NoError(t, got.err)

	assert.NotEqual(t, trail.ID, got.trail.ID, "a new run")
	assert.Equal(t, "Be brief.\n\nKeep two threads.", got.trail.CustomInstructions)
	<-f.turns.done
	assert.NotNil(t, f.turns.last().Play, "the new run hears the play")
	old, err := f.trails.GetTrail(t.Context(), trail.ID)
	require.NoError(t, err)
	assert.Equal(t, TrailFailed, old.State, "the old trail ends as it had")
	assert.Equal(t, "Lost the connection to T3 Code", old.LastError)
	assert.Contains(t, f.threads.noteBodies(), threadGoneNote)
}

func TestContinue_Refusals(t *testing.T) {
	f := newRunnerFixture()
	f.turns.hold = true
	t.Cleanup(f.turns.releaseAll)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("th-1")

	_, err := f.runner.Continue(ctxAs(starter), trail.ID, "more", ViaWeb)
	assert.ErrorIs(t, err, apperrs.ErrConflict, "a run still going is answered or stopped, not continued")
	_, err = f.runner.Continue(ctxAs("stranger"), trail.ID, "more", ViaWeb)
	assert.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.runner.Continue(ctxAs(starter), trail.ID, "  ", ViaWeb)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = f.runner.Continue(ctxAs(starter), " ", "more", ViaWeb)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = f.runner.Continue(ctxAs(starter), "missing", "more", ViaWeb)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)

	seedTrail(f, "tr-old", TrailDone, "th-0")
	_, err = f.runner.Continue(ctxAs(starter), "tr-old", "more", ViaWeb)
	assert.ErrorIs(t, err, apperrs.ErrConflict, "another run on the target is going")
	assert.Len(t, f.turns.reqs, 1, "nothing was sent")
}

func TestContinue_ThreadGone_StartingAgainNeedsALocation_SaysItDidNotStart(t *testing.T) {
	f := newRunnerFixture()
	trail := endedRun(t, f)
	f.harness.unlinked = true

	done, req := continueAsync(t, f, trail.ID, "Keep two threads.")
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "the harness session is gone", SessionGone: true}, "")
	got := <-done

	var refusal *HarnessRefusal
	require.ErrorAs(t, got.err, &refusal, "the caller hears it needs a location, to ask where")
	assert.Equal(t, RefusalNeedsLocation, refusal.Reason)
	assert.Len(t, f.trails.all(), 1, "no new run started")
	assert.NotContains(t, f.threads.noteBodies(), threadGoneNote, "nothing claims the play started again")
	old, err := f.trails.GetTrail(t.Context(), trail.ID)
	require.NoError(t, err)
	assert.Equal(t, threadGoneNotStartedNote, old.Activity[len(old.Activity)-1].Summary)
}

func TestContinue_ThreadGone_StartsAgainWhereTheRunWas_WithoutAskingOrTheLink(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.ComputerID, in.HarnessProjectID = "c-1", "t3-nexul"
	trail, obs := driveTurn(t, f, in)
	obs.OnStarted("th-1")
	obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "Lost the connection to T3 Code"}, "")
	f.harness.unlinked = true

	done, req := continueAsync(t, f, trail.ID, "Keep two threads.")
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "the harness session is gone", SessionGone: true}, "")
	got := <-done

	require.NoError(t, got.err, "the run knows where it ran, so nobody is asked")
	assert.NotEqual(t, trail.ID, got.trail.ID)
	assert.True(t, f.harness.recorded, "the old run's location is rechecked, not the starter's link")
	assert.Equal(t, "t3-nexul", f.harness.lastChoice.HarnessProjectID)
	assert.Contains(t, f.threads.noteBodies(), threadGoneNote)
}

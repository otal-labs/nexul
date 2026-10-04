package plays

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func clarifyRun() RunInput {
	return RunInput{PlayID: clarifyID, TargetType: TargetDoc, TargetID: docID, Via: ViaWeb}
}

func TestRun_ClarifyPlay_OpenRoundFails_FailsTheRunBeforeTheTurn(t *testing.T) {
	f := newRunnerFixture()
	f.rounds.openErr = apperrs.ErrConflict

	_, err := f.runner.Run(ctxAs(starter), clarifyRun())

	require.ErrorIs(t, err, apperrs.ErrConflict)
	trails := f.trails.all()
	require.Len(t, trails, 1)
	assert.Equal(t, TrailFailed, trails[0].State)
	assert.Contains(t, trails[0].LastError, "open the clarification round")
	assert.Empty(t, f.turns.reqs, "no turn starts without a round to post to")
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)
}

func TestRun_ClarifyPlay_EndRoundFails_TheTrailStillEnds(t *testing.T) {
	f := newRunnerFixture()
	f.rounds.endErr = errors.New("db down")
	trail, obs := driveTurn(t, f, clarifyRun())

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")

	stored, err := f.trails.GetTrail(t.Context(), trail.ID)
	require.NoError(t, err)
	assert.Equal(t, TrailDone, stored.State)
}

// Every end of a Clarify run ends its round with the trail's id, and the lock the run took comes off (ADR 0121).
func TestRun_ClarifyPlay_EveryEndEndsTheRoundAndUnlocks(t *testing.T) {
	tests := []struct {
		name  string
		held  bool
		end   func(t *testing.T, f *runnerFixture, trail *Trail, obs agent.Observer)
		state TrailState
	}{
		{"done", false, func(_ *testing.T, _ *runnerFixture, _ *Trail, obs agent.Observer) {
			obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")
		}, TrailDone},
		{"failed", false, func(_ *testing.T, _ *runnerFixture, _ *Trail, obs agent.Observer) {
			obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "boom"}, "")
		}, TrailFailed},
		{"silent", false, func(_ *testing.T, _ *runnerFixture, _ *Trail, obs agent.Observer) {
			obs.(*trailObserver).onSilence()
		}, TrailFailed},
		{"stopped", true, func(t *testing.T, f *runnerFixture, trail *Trail, obs agent.Observer) {
			_, err := f.runner.Stop(ctxAs(starter), trail.ID)
			require.NoError(t, err)
			obs.OnFinished(harness.TurnResult{State: harness.TurnInterrupted}, "")
		}, TrailInterrupted},
		{"stopped with no live turn", false, func(t *testing.T, f *runnerFixture, trail *Trail, _ agent.Observer) {
			_, err := f.runner.Stop(ctxAs(starter), trail.ID)
			require.NoError(t, err)
		}, TrailInterrupted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			if tt.held {
				f = heldFixture(t)
			}
			trail, obs := driveTurn(t, f, clarifyRun())

			round, ended := f.rounds.round(trail.ID)
			assert.Equal(t, fakeClarifyRound{docID: docID, starter: starter, tookLock: true, running: true}, round, "the round opens before the turn, recording the lock")
			assert.Empty(t, ended)
			assert.Equal(t, []string{clarifyLockedNote}, f.threads.noteBodies()[:1])

			tt.end(t, f, trail, obs)

			stored, err := f.trails.GetTrail(t.Context(), trail.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.state, stored.State)
			round, ended = f.rounds.round(trail.ID)
			assert.Equal(t, []string{trail.ID}, ended, "ended once, by its own trail")
			assert.False(t, round.running)
			locked, _ := f.locks.state(docID)
			assert.False(t, locked, "the lock this run took comes off")
		})
	}
}

func TestRun_ClarifyPlay_DocLockedBeforeTheRun_StaysLocked(t *testing.T) {
	f := newRunnerFixture()
	f.locks.locked[docID] = true
	trail, obs := driveTurn(t, f, clarifyRun())

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")

	round, ended := f.rounds.round(trail.ID)
	assert.False(t, round.tookLock)
	assert.Equal(t, []string{trail.ID}, ended)
	locked, _ := f.locks.state(docID)
	assert.True(t, locked, "a lock someone else set stays")
	assert.Empty(t, f.threads.noteBodies(), "no lock note when the run took no lock")
}

func TestRun_ClarifyPlay_WaitingOnAQuestion_KeepsTheRoundUntilStop(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, clarifyRun())
	obs.OnQuestion(askedQuestion())
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")
	f.runner.clearRun(trail.ID, f.runner.run(trail.ID))

	_, ended := f.rounds.round(trail.ID)
	assert.Empty(t, ended, "a run parked on a question still holds its round")

	_, err := f.runner.Stop(ctxAs(starter), trail.ID)
	require.NoError(t, err)
	_, ended = f.rounds.round(trail.ID)
	assert.Equal(t, []string{trail.ID}, ended, "Stop frees the doc")
}

func TestResumeRunsAfterRestart_ClarifyRunsEndTheirRounds(t *testing.T) {
	f := newRunnerFixture()
	for id, session := range map[string]string{"tr-running": "sess-1", "tr-starting": ""} {
		seedTrail(f, id, TrailRunning, session)
		f.trails.byID[id].PlayID, f.trails.byID[id].TargetType, f.trails.byID[id].TargetID = clarifyID, TargetDoc, docID
		f.rounds.byTrail[id] = &fakeClarifyRound{docID: docID, starter: starter, tookLock: true, running: true}
	}
	f.trails.byID["tr-starting"].State = TrailStarting

	require.NoError(t, f.runner.ResumeRunsAfterRestart(t.Context()))
	<-f.turns.done

	_, ended := f.rounds.round("tr-running")
	assert.Equal(t, []string{"tr-starting"}, ended, "a run cut off before it started ends its round at once")

	req := f.turns.last()
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")

	round, ended := f.rounds.round("tr-running")
	assert.Equal(t, []string{"tr-starting", "tr-running"}, ended, "the followed run ends its round as it ends")
	assert.False(t, round.running)
}

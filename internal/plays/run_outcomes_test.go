package plays

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// driveTurn starts a run over the fake turn runner and returns the observer the pipeline would drive.
func driveTurn(t *testing.T, f *runnerFixture, in RunInput) (*Trail, agent.Observer) {
	t.Helper()
	trail, err := f.runner.Run(ctxAs(starter), in)
	require.NoError(t, err)
	<-f.turns.done
	releaseUnheldTurn(f, trail.ID)
	return trail, f.turns.last().Observer
}

// releaseUnheldTurn drops the live entry of a turn the fake already returned from, as the runner's goroutine would
// on exit; done is sent before RunTurn returns, so without this a Stop races that exit.
func releaseUnheldTurn(f *runnerFixture, trailID string) {
	if !f.turns.hold {
		f.runner.clearRun(trailID, f.runner.run(trailID))
	}
}

func TestFinish_WritesRunStartedAndFinishedEvents(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.Via = ViaMCP
	trail, obs := driveTurn(t, f, in)

	obs.OnStarted("sess-1")
	obs.OnActivity(step("Read", "Read main.go"))
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")

	ref := RunRef{TrailID: trail.ID, PlayID: fixPlayID, PlayLabel: "Fix with AI", TargetType: TargetTicket, TargetID: ticketID, TargetTitle: "NEX-1", StarterID: starter, Via: ViaMCP, WorkspaceID: workspaceID}
	started := f.trails.eventsFor(TopicRunStarted)
	require.Len(t, started, 1)
	assert.NotEmpty(t, started[0].ID)
	assert.Equal(t, RunStartedEvent{RunRef: ref, HarnessSessionID: "sess-1"}, started[0].Payload)
	finished := f.trails.eventsFor(TopicRunFinished)
	require.Len(t, finished, 1)
	assert.Equal(t, RunFinishedEvent{RunRef: ref, Outcome: TrailDone, ReplyMessageID: "reply-1"}, finished[0].Payload)
}

func TestFinish_FailedOutcome_EventCarriesTheReason(t *testing.T) {
	f := newRunnerFixture()
	_, obs := driveTurn(t, f, RunInput{PlayID: docPlayID, TargetType: TargetDoc, TargetID: docID})

	obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "harness refused"}, "")

	finished := f.trails.eventsFor(TopicRunFinished)
	require.Len(t, finished, 1)
	e := finished[0].Payload.(RunFinishedEvent)
	assert.Equal(t, TrailFailed, e.Outcome)
	assert.Equal(t, "harness refused", e.LastError)
	assert.Equal(t, "Roadmap", e.TargetTitle, "a doc run names the doc")
	assert.Empty(t, f.trails.eventsFor(TopicRunStarted), "a run that never reached running has no started event")
}

func TestRun_HarnessNotReady_WritesFinishedEvent(t *testing.T) {
	f := newRunnerFixture()
	f.harness.err = errors.New("not paired")
	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.Error(t, err)

	finished := f.trails.eventsFor(TopicRunFinished)
	require.Len(t, finished, 1)
	assert.Equal(t, TrailFailed, finished[0].Payload.(RunFinishedEvent).Outcome)
	frames := f.live.snapshot()
	require.Len(t, frames, 1)
	assert.Equal(t, TrailFailed, frames[0].State)
}

func TestObserver_PublishesLiveFramesOnStateAndActivity(t *testing.T) {
	f := newRunnerFixture()
	trail, obs := driveTurn(t, f, ticketRun())

	obs.OnStarted("sess-1")
	obs.OnActivity(step("Read", "Read main.go"))
	obs.OnSnapshot()
	obs.OnActivity(step("Bash", "Bash go test"))
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")

	frames := f.live.snapshot()
	require.Len(t, frames, 4, "running, two activity lines, done; snapshots publish nothing")
	assert.Equal(t, RunFrame{TrailID: trail.ID, PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID, State: TrailRunning, Activity: nil, ProjectID: "proj-1", WorkspaceID: "workspace-1"}, frames[0], "the frame names the run's project and workspace")
	require.NotNil(t, frames[1].Activity)
	assert.Equal(t, ActivityEntry(step("Read", "Read main.go")), *frames[1].Activity, "the frame carries the whole latest step")
	assert.Equal(t, "Bash go test", frames[2].Activity.Summary)
	assert.Equal(t, TrailDone, frames[3].State)
	assert.Equal(t, "Bash go test", frames[3].Activity.Summary, "the latest step rides along with the terminal state")
	require.NotNil(t, frames[3].EndedAt)
}

func TestObserver_SecondTerminal_IsIgnored(t *testing.T) {
	f := newRunnerFixture()
	_, obs := driveTurn(t, f, ticketRun())

	obs.OnFinished(harness.TurnResult{State: harness.TurnError, LastError: "first"}, "")
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-late")
	obs.OnActivity(step("Bash", "late line"))

	final := f.trails.all()[0]
	assert.Equal(t, TrailFailed, final.State)
	assert.Equal(t, "first", final.LastError)
	assert.Empty(t, final.Activity)
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)
}

func TestRun_PostStartedMessageFails_NotesTheReason(t *testing.T) {
	f := newRunnerFixture()
	f.threads.postErr = errors.New("chat down")

	_, err := f.runner.Run(ctxAs(starter), ticketRun())

	require.Error(t, err)
	assert.Equal(t, []string{"Run failed: post started message: chat down"}, f.threads.noteBodies())
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)
}

// silentHarness accepts the turn and then sends nothing; the returned channel lets a test feed updates.
func silentHarness() (*harnesstest.Client, chan harness.Update) {
	ch := make(chan harness.Update, 16)
	return &harnesstest.Client{StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
		return harness.StartResult{SessionID: "sess-1", Updates: ch}, nil
	}}, ch
}

func TestSilence_NoHarnessUpdate_FailsTheRunOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, _ := silentHarness()
		interrupted := make(chan string, 1)
		client.InterruptFn = func(_ context.Context, target harness.Target) error {
			interrupted <- target.SessionID
			return nil
		}
		convs := newHarnessRunner(f, client)

		_, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)

		final := <-f.trails.terminal
		synctest.Wait()

		assert.Equal(t, TrailFailed, final.State)
		assert.Equal(t, "no harness update for 15m", final.LastError)
		assert.Equal(t, "sess-1", final.HarnessSessionID, "the harness had accepted before going quiet")
		assert.Equal(t, []string{"Run failed: no harness update for 15m"}, f.threads.noteBodies())
		_, pipelineNotes := convs.snapshot()
		assert.Empty(t, pipelineNotes, "the cancelled pipeline leaves the note to the runner")
		assert.Equal(t, []TrailState{TrailStarting, TrailRunning, TrailFailed}, f.trails.recordedStates(), "the pipeline's later terminal is ignored")
		assert.Equal(t, "sess-1", <-interrupted, "the harness stops the turn the trail gave up on")
		f.runner.mu.Lock()
		assert.Empty(t, f.runner.runs, "the cancelled turn released its live entry")
		f.runner.mu.Unlock()
	})
}

func TestSilence_InterruptFails_StillFailsTheRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, _ := silentHarness()
		client.InterruptFn = func(context.Context, harness.Target) error { return errors.New("computer offline") }
		newHarnessRunner(f, client)

		_, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)

		final := <-f.trails.terminal
		synctest.Wait()

		assert.Equal(t, TrailFailed, final.State)
		assert.Equal(t, "no harness update for 15m", final.LastError)
		f.runner.mu.Lock()
		assert.Empty(t, f.runner.runs, "the turn is released even when the harness could not be stopped")
		f.runner.mu.Unlock()
	})
}

func TestSilence_ContinuousActivity_KeepsTheRunAlive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, ch := silentHarness()
		newHarnessRunner(f, client)

		_, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)
		go func() {
			for i := 0; i < 4; i++ {
				time.Sleep(HarnessSilenceTimeout / 2)
				ch <- harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "working", Streaming: true}}
				time.Sleep(HarnessSilenceTimeout / 2)
				ch <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Bash", Summary: "step"}}
			}
			ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
			close(ch)
		}()

		final := <-f.trails.terminal
		synctest.Wait()

		assert.Equal(t, TrailDone, final.State, "four windows of steady updates never trip the timeout")
		assert.Len(t, final.Activity, 5, "the text streamed before the first step becomes a step of its own, then four tool steps")
		assert.Empty(t, f.threads.noteBodies())
	})
}

func TestStop_Starter_InterruptsTheHarnessAndKeepsTheActivity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, ch := silentHarness()
		client.InterruptFn = func(context.Context, harness.Target) error {
			ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnInterrupted}}
			close(ch)
			return nil
		}
		convs := newHarnessRunner(f, client)
		trail, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)
		ch <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Read", Summary: "Read main.go"}}
		synctest.Wait()

		stopped, err := f.runner.Stop(ctxAs(starter), trail.ID)
		require.NoError(t, err)
		assert.Equal(t, trail.ID, stopped.ID)

		final := <-f.trails.terminal
		synctest.Wait()
		assert.Equal(t, TrailInterrupted, final.State)
		assert.Equal(t, "stopped by login-u-1", final.LastError)
		assert.Equal(t, []string{"Read main.go", "Run stopped by login-u-1."}, summaries(final.Activity), "the stop is the trail's last line")
		assert.Equal(t, ActivityNote, final.Activity[1].Kind)
		assert.Equal(t, []string{"Run stopped by login-u-1."}, f.threads.noteBodies())
		_, pipelineNotes := convs.snapshot()
		assert.Equal(t, []string{"Agent turn interrupted."}, pipelineNotes)
	})
}

func TestStop_HarnessRacesWithDone_EndsInterruptedNotDone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, ch := silentHarness()
		client.InterruptFn = func(context.Context, harness.Target) error {
			// The harness had already finished the turn when the interrupt request landed.
			ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
			close(ch)
			return nil
		}
		newHarnessRunner(f, client)
		trail, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)
		synctest.Wait()

		_, err = f.runner.Stop(ctxAs(starter), trail.ID)
		require.NoError(t, err)

		final := <-f.trails.terminal
		synctest.Wait()
		assert.Equal(t, TrailInterrupted, final.State, "a stop request never reads as done, even if the harness raced to done")
		assert.Equal(t, "stopped by login-u-1", final.LastError)
	})
}

func TestStop_HarnessReportsErrorAfterStop_KeepsTheHarnessErrorText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, ch := silentHarness()
		client.InterruptFn = func(context.Context, harness.Target) error {
			ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "connection reset"}}
			close(ch)
			return nil
		}
		newHarnessRunner(f, client)
		trail, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)
		synctest.Wait()

		_, err = f.runner.Stop(ctxAs(starter), trail.ID)
		require.NoError(t, err)

		final := <-f.trails.terminal
		synctest.Wait()
		assert.Equal(t, TrailInterrupted, final.State)
		assert.Equal(t, "connection reset", final.LastError, "the harness's own error text wins over the stop reason")
	})
}

func TestStop_HarnessInterruptFails_ClosesTheTrailHere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, _ := silentHarness()
		client.InterruptFn = func(context.Context, harness.Target) error { return errors.New("socket gone") }
		convs := newHarnessRunner(f, client)
		trail, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)
		synctest.Wait()

		_, err = f.runner.Stop(ctxAs(starter), trail.ID)
		require.NoError(t, err)

		final := <-f.trails.terminal
		synctest.Wait()
		assert.Equal(t, TrailInterrupted, final.State)
		assert.Equal(t, "stopped by login-u-1", final.LastError)
		assert.Equal(t, []string{"Run stopped by login-u-1."}, f.threads.noteBodies())
		_, pipelineNotes := convs.snapshot()
		assert.Empty(t, pipelineNotes, "the cancelled pipeline posts nothing of its own")
		assert.Equal(t, []TrailState{TrailStarting, TrailRunning, TrailInterrupted}, f.trails.recordedStates())
	})
}

func TestStop_PlaysWriteHolder_ClosesAStuckTrail(t *testing.T) {
	f := newRunnerFixture()
	f.perm.grants["manager"] = []permissions.Action{permissions.PlaysWrite}
	trail, _ := driveTurn(t, f, ticketRun())

	stopped, err := f.runner.Stop(ctxAs("manager"), trail.ID)
	require.NoError(t, err)

	assert.Equal(t, TrailInterrupted, stopped.State)
	assert.Equal(t, "stopped by login-manager", stopped.LastError)
	assert.Equal(t, []string{"Run stopped by login-manager."}, f.threads.noteBodies())
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)

	_, err = f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err, "the target is free again")
}

func TestStop_Refusals(t *testing.T) {
	f := newRunnerFixture()
	f.perm.grants["reader"] = []permissions.Action{permissions.PlaysRead}
	trail, _ := driveTurn(t, f, ticketRun())

	_, err := f.runner.Stop(ctxAs("reader"), trail.ID)
	assert.ErrorIs(t, err, apperrs.ErrForbidden, "a third user without plays:write is refused")
	_, err = f.runner.Stop(ctxAs(starter), " ")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = f.runner.Stop(ctxAs(starter), "missing")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)

	_, err = f.runner.Stop(ctxAs(starter), trail.ID)
	require.NoError(t, err)
	_, err = f.runner.Stop(ctxAs(starter), trail.ID)
	assert.ErrorIs(t, err, apperrs.ErrConflict, "a finished run cannot be stopped again")
	assert.Len(t, f.threads.noteBodies(), 1)
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "15m", formatDuration(15*time.Minute))
	assert.Equal(t, "1m", formatDuration(time.Minute))
	assert.Equal(t, "30s", formatDuration(30*time.Second))
	assert.Equal(t, "1m30s", formatDuration(90*time.Second))
}

func seedTrail(f *runnerFixture, id string, state TrailState, sessionID string) {
	f.trails.byID[id] = &Trail{ID: id, WorkspaceID: workspaceID, PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID,
		ConversationID: "conv-" + id, StarterID: starter, State: state, StartedAt: fixedNow, HarnessSessionID: sessionID,
		ComputerID: "pc-1", Provider: "codex", Model: "gpt"}
}

func TestResumeRunsAfterRestart_FollowsRunningRunsAndEndsUnstartedOnes(t *testing.T) {
	f := newRunnerFixture()
	seedTrail(f, "tr-running", TrailRunning, "sess-1")
	seedTrail(f, "tr-starting", TrailStarting, "")
	seedTrail(f, "tr-waiting", TrailWaiting, "sess-2")
	seedTrail(f, "tr-done", TrailDone, "sess-3")

	require.NoError(t, f.runner.ResumeRunsAfterRestart(t.Context()))
	<-f.turns.done

	req := f.turns.last()
	assert.True(t, req.Watch, "the turn is followed, never started again")
	assert.Equal(t, "conv-tr-running", req.ConversationID)
	assert.Equal(t, starter, req.ViaUserID)
	assert.Equal(t, &agent.TargetOverride{ComputerID: "pc-1", Provider: "codex", Model: "gpt"}, req.Target)
	assert.Empty(t, req.RequestBody)

	for id, want := range map[string]TrailState{"tr-running": TrailRunning, "tr-starting": TrailInterrupted, "tr-waiting": TrailWaiting, "tr-done": TrailDone} {
		got, err := f.trails.GetTrail(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, want, got.State, id)
	}
	unstarted, err := f.trails.GetTrail(t.Context(), "tr-starting")
	require.NoError(t, err)
	assert.Equal(t, "Nexul restarted before the run started", unstarted.LastError)
	assert.ElementsMatch(t, []string{"Nexul restarted; following the run in T3 Code again.", "Nexul restarted before the run started."}, f.threads.noteBodies())

	req.Observer.OnStarted("sess-1")
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")
	ended, err := f.trails.GetTrail(t.Context(), "tr-running")
	require.NoError(t, err)
	assert.Equal(t, TrailDone, ended.State, "the followed run ends with the harness's own outcome")
	assert.Equal(t, "reply-1", ended.ReplyMessageID)
	assert.Empty(t, f.trails.eventsFor(TopicRunStarted), "a followed run announces no second start")
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 2)
}

func TestResumeRunsAfterRestart_NamesTheTargetAsTheStarter(t *testing.T) {
	f := newRunnerFixture()
	f.targets.needActor = true
	seedTrail(f, "tr-starting", TrailStarting, "")

	require.NoError(t, f.runner.ResumeRunsAfterRestart(t.Context()))

	finished := f.trails.eventsFor(TopicRunFinished)
	require.Len(t, finished, 1)
	assert.Equal(t, "NEX-1", finished[0].Payload.(RunFinishedEvent).TargetTitle, "boot has no signed-in actor, so the target is read as the run's starter")
}

func TestResumeRunsAfterRestart_ListFails_ReturnsTheError(t *testing.T) {
	f := newRunnerFixture()
	f.trails.listErr = errors.New("db down")

	assert.ErrorContains(t, f.runner.ResumeRunsAfterRestart(t.Context()), "db down")
}

func TestFollowThread_NewsOnAnEndedRunsThread_ReopensItsTrailAndCatchesUpFromSince(t *testing.T) {
	f := newRunnerFixture()
	seedTrail(f, "tr-failed", TrailFailed, "th-1")
	f.trails.byID["tr-failed"].LastError = "Lost the connection to T3 Code"

	done, ok := f.runner.FollowThread(t.Context(), "conv-tr-failed", "th-1", starter, "pc-1", "run-7")
	require.True(t, ok)
	<-f.turns.done

	req := f.turns.last()
	assert.True(t, req.Watch, "a catch-up sends nothing")
	assert.Equal(t, "run-7", req.Since)
	assert.Equal(t, "conv-tr-failed", req.ConversationID)
	assert.Equal(t, &agent.TargetOverride{ComputerID: "pc-1", Provider: "codex", Model: "gpt"}, req.Target)
	reopened, err := f.trails.GetTrail(t.Context(), "tr-failed")
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, reopened.State)
	assert.Nil(t, reopened.EndedAt)
	assert.Empty(t, reopened.LastError)
	assert.Contains(t, f.threads.noteBodies(), followedAgainNote, "the thread says why the run moved again")

	req.Observer.OnStarted("th-1")
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-2")
	<-done
	ended, err := f.trails.GetTrail(t.Context(), "tr-failed")
	require.NoError(t, err)
	assert.Equal(t, TrailDone, ended.State, "it ends with the harness's own outcome this time")
	assert.Equal(t, "reply-2", ended.ReplyMessageID)
	assert.Empty(t, f.trails.eventsFor(TopicRunStarted), "a reopened run announces no second start")
}

func TestFollowThread_RunsUnreadable_LeavesItToAPlainTurn(t *testing.T) {
	f := newRunnerFixture()
	f.trails.latestErr = errors.New("db down")

	_, ok := f.runner.FollowThread(t.Context(), "conv-1", "th-1", starter, "pc-1", "run-7")

	assert.False(t, ok)
}

func TestFollowThread_NotThisRunsToFollow_LeavesItToAPlainTurn(t *testing.T) {
	tests := []struct {
		name                         string
		state                        TrailState
		thread, user, computer, conv string
	}{
		{"a run still going", TrailRunning, "th-1", starter, "pc-1", "conv-tr-1"},
		{"a run on another thread", TrailDone, "th-2", starter, "pc-1", "conv-tr-1"},
		{"someone else's run", TrailDone, "th-1", "u-other", "pc-1", "conv-tr-1"},
		{"a run on another computer", TrailDone, "th-1", starter, "pc-2", "conv-tr-1"},
		{"a conversation no run used", TrailDone, "th-1", starter, "pc-1", "conv-none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			seedTrail(f, "tr-1", tt.state, "th-1")

			_, ok := f.runner.FollowThread(t.Context(), tt.conv, tt.thread, tt.user, tt.computer, "run-7")

			assert.False(t, ok)
			assert.Empty(t, f.turns.done)
			got, err := f.trails.GetTrail(t.Context(), "tr-1")
			require.NoError(t, err)
			assert.Equal(t, tt.state, got.State, "the trail is left as it was")
		})
	}
}

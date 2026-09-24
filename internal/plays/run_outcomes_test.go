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
	return trail, f.turns.last().Observer
}

func TestFinish_Done_MovesToTheChosenLaterColumn(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.MoveToStatusID = "st-review"
	trail, obs := driveTurn(t, f, in)

	obs.OnStarted("sess-1")
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")

	moves := f.mover.snapshot()
	require.Len(t, moves, 1)
	assert.Equal(t, fakeMove{ticketID, "st-review", PlayActor{PlayLabel: "Fix with AI", TrailID: trail.ID, StarterID: starter, Via: ViaWeb}}, moves[0])
	assert.Empty(t, f.threads.noteBodies(), "a clean move needs no note")
}

func TestFinish_Done_MCPRun_ActorCarriesMCPProvenance(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.MoveToStatusID = "st-review"
	in.Via = ViaMCP
	_, obs := driveTurn(t, f, in)

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	moves := f.mover.snapshot()
	require.Len(t, moves, 1)
	assert.Equal(t, ViaMCP, moves[0].actor.Via)
}

func TestFinish_Done_SameStage_StillMoves(t *testing.T) {
	f := newRunnerFixture()
	f.targets.statuses["st-review-2"] = StatusTarget{Name: "Reviewing", Stage: StageReview}
	in := ticketRun()
	in.MoveToStatusID = "st-review-2"
	_, obs := driveTurn(t, f, in)
	f.targets.setTicketStage(ticketID, StageReview)

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	assert.Len(t, f.mover.snapshot(), 1, "equal stage counts as later or equal")
}

func TestFinish_Done_TicketAlreadyFurther_SkipsWithNote(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.MoveToStatusID = "st-review"
	_, obs := driveTurn(t, f, in)
	f.targets.setTicketStage(ticketID, StageDone)

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	assert.Empty(t, f.mover.snapshot())
	assert.Equal(t, []string{"Ticket is already in done; not moving it back to In review"}, f.threads.noteBodies())
	final := f.trails.all()[0]
	require.NotEmpty(t, final.Activity)
	last := final.Activity[len(final.Activity)-1]
	assert.Equal(t, ActivityNote, last.Kind, "the skip is on the trail too, so the transcript shows it")
	assert.Equal(t, "Ticket is already in done; not moving it back to In review", last.Summary)
	assert.False(t, last.At.IsZero())
}

func TestFinish_Done_ColumnDeleted_SkipsWithNote(t *testing.T) {
	f := newRunnerFixture()
	in := ticketRun()
	in.MoveToStatusID = "st-gone"
	_, obs := driveTurn(t, f, in)

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	assert.Empty(t, f.mover.snapshot())
	assert.Equal(t, []string{"Chosen column no longer exists; ticket left where it is"}, f.threads.noteBodies())
}

func TestFinish_Done_MoveFails_NotesTheReason(t *testing.T) {
	f := newRunnerFixture()
	f.mover.err = errors.New("tickets down")
	in := ticketRun()
	in.MoveToStatusID = "st-review"
	_, obs := driveTurn(t, f, in)

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	assert.Equal(t, []string{"Could not move the ticket to In review: tickets down"}, f.threads.noteBodies())
}

func TestFinish_NoMoveWhenNotDoneOrNotATicketOrNothingChosen(t *testing.T) {
	tests := []struct {
		name string
		in   RunInput
		res  harness.TurnResult
	}{
		{"failed run", RunInput{PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID, MoveToStatusID: "st-review"}, harness.TurnResult{State: harness.TurnError, LastError: "boom"}},
		{"interrupted run", RunInput{PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID, MoveToStatusID: "st-review"}, harness.TurnResult{State: harness.TurnInterrupted}},
		{"no column chosen", ticketRun(), harness.TurnResult{State: harness.TurnDone}},
		{"doc play", RunInput{PlayID: docPlayID, TargetType: TargetDoc, TargetID: docID, MoveToStatusID: "st-review"}, harness.TurnResult{State: harness.TurnDone}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			_, obs := driveTurn(t, f, tt.in)
			obs.OnFinished(tt.res, "")
			assert.Empty(t, f.mover.snapshot())
			assert.Empty(t, f.threads.noteBodies(), "the pipeline owns the notes for its own outcomes")
		})
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

	ref := RunRef{TrailID: trail.ID, PlayID: fixPlayID, PlayLabel: "Fix with AI", TargetType: TargetTicket, TargetID: ticketID, TargetTitle: "NEX-1", StarterID: starter, Via: ViaMCP}
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
	assert.Equal(t, RunFrame{TrailID: trail.ID, PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID, State: TrailRunning, Activity: nil}, frames[0])
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
		f.runner.mu.Lock()
		assert.Empty(t, f.runner.runs, "the cancelled turn released its live entry")
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

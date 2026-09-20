package plays

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// heldFixture keeps the fake turn live until the test ends, as a real turn parked on a question would be.
func heldFixture(t *testing.T) *runnerFixture {
	t.Helper()
	f := newRunnerFixture()
	f.turns.hold = true
	t.Cleanup(f.turns.releaseAll)
	return f
}

func askedQuestion() harness.Question {
	return harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{
		ID: "q1", Text: "Proceed without Nexul tools?", Header: "Nexul MCP down",
		Options: []harness.QuestionOption{{Label: "Yes", Description: "skip the links"}, {Label: "No"}},
	}}}
}

func yesAnswer() harness.QuestionAnswer {
	return harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
}

func TestObserver_Question_ParksTheRunWithoutOutcomes(t *testing.T) {
	f := heldFixture(t)
	in := ticketRun()
	in.MoveToStatusID = "st-review"
	trail, obs := driveTurn(t, f, in)
	obs.OnStarted("sess-1")

	obs.OnQuestion(askedQuestion())

	got := f.trails.all()[0]
	assert.Equal(t, TrailWaiting, got.State)
	require.NotNil(t, got.Question)
	assert.Equal(t, "req-1", got.Question.RequestID)
	assert.Nil(t, got.Question.Answer)
	assert.Equal(t, fixedNow, got.Question.AskedAt)
	assert.Nil(t, got.EndedAt)
	assert.Len(t, f.trails.eventsFor(TopicRunWaiting), 1, "the starter is told the run needs them")
	assert.Equal(t, trail.ID, f.trails.eventsFor(TopicRunWaiting)[0].Payload.(RunWaitingEvent).TrailID)
	assert.Empty(t, f.trails.eventsFor(TopicRunFinished), "waiting is not an outcome")
	assert.Empty(t, f.mover.snapshot(), "the ticket does not move")
	frames := f.live.snapshot()
	last := frames[len(frames)-1]
	assert.Equal(t, TrailWaiting, last.State)
	require.NotNil(t, last.Question, "the live frame carries the question")

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")
	got = f.trails.all()[0]
	assert.Equal(t, TrailWaiting, got.State, "the harness closing the turn under the question keeps the run waiting")
	assert.Empty(t, f.trails.eventsFor(TopicRunFinished))
	assert.Empty(t, f.mover.snapshot())

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	assert.ErrorIs(t, err, apperrs.ErrConflict, "a waiting run still occupies its target")
}

func TestAnswer_LiveTurn_ContinuesTheSameTrail(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())

	got, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	require.NotNil(t, got.Question.Answer)
	assert.Equal(t, yesAnswer(), *got.Question.Answer)
	assert.Equal(t, []harness.QuestionAnswer{yesAnswer()}, f.turns.answered)
	posts := f.threads.snapshot()
	assert.Equal(t, fakePost{trail.ConversationID, starter, "Answered: Yes"}, posts[len(posts)-1])
	assert.Len(t, f.turns.reqs, 1, "the live turn continues; no fresh turn is started")

	obs.OnActivity(step("Bash", "go test"))
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")
	final := f.trails.all()[0]
	assert.Equal(t, TrailDone, final.State)
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)
	assert.Len(t, f.trails.eventsFor(TopicRunStarted), 1)
}

func TestAnswer_TurnGone_ResumesAFreshTurnOnTheSameTrail(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	got, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	<-f.turns.done
	req := f.turns.last()
	assert.Equal(t, "Answered: Yes", req.RequestBody, "the answer is the fresh turn's request")
	assert.Equal(t, trail.ConversationID, req.ConversationID)
	assert.Equal(t, starter, req.ViaUserID)
	assert.Empty(t, req.ExtraRequestBlocks)
	assert.Len(t, f.trails.all(), 1, "same trail")
	assert.Empty(t, f.turns.answered, "nothing to answer on a turn that is gone")

	req.Observer.OnStarted("sess-1")
	assert.Len(t, f.trails.eventsFor(TopicRunStarted), 1, "a resumed turn announces no second start")
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-2")
	final := f.trails.all()[0]
	assert.Equal(t, TrailDone, final.State)
	assert.Equal(t, "reply-2", final.ReplyMessageID)
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)
}

func TestAnswer_PipelineLostTheTurn_ResumesAFreshTurn(t *testing.T) {
	f := heldFixture(t)
	f.turns.answerErr = apperrs.ErrNotFound
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())

	_, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	<-f.turns.done
	assert.Len(t, f.turns.reqs, 2)
	assert.Equal(t, "Answered: Yes", f.turns.last().RequestBody)
}

func TestAnswer_Refusals(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")

	_, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	assert.ErrorIs(t, err, apperrs.ErrConflict, "a running turn has nothing to answer")

	obs.OnQuestion(askedQuestion())
	_, err = f.runner.Answer(ctxAs("stranger"), trail.ID, yesAnswer())
	assert.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.runner.Answer(ctxAs(starter), trail.ID, harness.QuestionAnswer{})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = f.runner.Answer(ctxAs(starter), "missing", yesAnswer())
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.runner.Answer(ctxAs(starter), " ", yesAnswer())
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, f.turns.answered)
	assert.Equal(t, TrailWaiting, f.trails.all()[0].State)
}

func TestAnswer_MultiQuestion_SummaryListsEachAnswer(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(harness.Question{RequestID: "req-2", Questions: []harness.QuestionItem{
		{ID: "a", Text: "Which direction?", Options: []harness.QuestionOption{{Label: "In"}, {Label: "Out"}}, MultiSelect: true},
		{ID: "b", Text: "What name?"},
	}})

	_, err := f.runner.Answer(ctxAs(starter), trail.ID, harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{
		"a": {Selected: []string{"In", "Out"}}, "b": {Text: "Bot"},
	}})
	require.NoError(t, err)
	posts := f.threads.snapshot()
	assert.Equal(t, "Answered:\n- Which direction?: In, Out\n- What name?: Bot", posts[len(posts)-1].body)
}

func TestStop_FromWaiting_EndsInterrupted(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())

	_, err := f.runner.Stop(ctxAs(starter), trail.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{trail.ConversationID}, f.turns.interrupted)
	// A T3 stop closes the turn as done; the requested stop still reads as interrupted.
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	final := f.trails.all()[0]
	assert.Equal(t, TrailInterrupted, final.State)
	require.NotNil(t, final.EndedAt)
	finished := f.trails.eventsFor(TopicRunFinished)
	require.Len(t, finished, 1)
	assert.Equal(t, TrailInterrupted, finished[0].Payload.(RunFinishedEvent).Outcome)
}

func TestStop_WaitingAfterTheTurnClosed_EndsInterrupted(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")
	f.runner.clearRun(trail.ID, f.runner.run(trail.ID))

	stopped, err := f.runner.Stop(ctxAs(starter), trail.ID)
	require.NoError(t, err)
	assert.Equal(t, TrailInterrupted, stopped.State)
	assert.Len(t, f.trails.eventsFor(TopicRunFinished), 1)
}

func TestSilence_PausedWhileWaiting_AnswerRestartsTheClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newRunnerFixture()
		client, ch := silentHarness()
		client.AnswerFn = func(_ context.Context, target harness.Target, requestID string, answer harness.QuestionAnswer) error {
			assert.Equal(t, "sess-1", target.SessionID)
			assert.Equal(t, "req-1", requestID)
			assert.Equal(t, yesAnswer(), answer)
			ch <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Bash", Summary: "go test"}}
			ch <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
			close(ch)
			return nil
		}
		convs := newHarnessRunner(f, client)
		trail, err := f.runner.Run(ctxAs(starter), ticketRun())
		require.NoError(t, err)

		q := askedQuestion()
		ch <- harness.Update{Question: &q}
		synctest.Wait()
		time.Sleep(HarnessSilenceTimeout * 3)
		synctest.Wait()
		parked := f.trails.all()[0]
		assert.Equal(t, TrailWaiting, parked.State, "the silence clock does not run while the user is the one being waited on")
		replies, _ := convs.snapshot()
		require.Len(t, replies, 1)
		assert.Contains(t, replies[0], "```nexul-question", "the question is the Agent's message in the thread")

		_, err = f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
		require.NoError(t, err)
		final := <-f.trails.terminal
		synctest.Wait()
		assert.Equal(t, TrailDone, final.State)
		assert.Equal(t, []string{"go test"}, summaries(final.Activity))
		assert.Equal(t, yesAnswer(), *final.Question.Answer)
		posts := f.threads.snapshot()
		assert.Equal(t, "Answered: Yes", posts[len(posts)-1].body)
		assert.Empty(t, f.threads.noteBodies(), "no silence failure")
	})
}

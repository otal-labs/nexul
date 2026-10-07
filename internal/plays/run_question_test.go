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
	frames := f.live.snapshot()
	last := frames[len(frames)-1]
	assert.Equal(t, TrailWaiting, last.State)
	require.NotNil(t, last.Question, "the live frame carries the question")

	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-1")
	got = f.trails.all()[0]
	assert.Equal(t, TrailWaiting, got.State, "the harness closing the turn under the question keeps the run waiting")
	assert.Empty(t, f.trails.eventsFor(TopicRunFinished))

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
	in := ticketRun()
	in.MemoryIDs = []string{pickedMem}
	trail, obs := driveTurn(t, f, in)
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")

	got, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	<-f.turns.done
	req := f.turns.last()
	assert.Equal(t, "Answered: Yes", req.RequestBody, "the answer is the fresh turn's request")
	assert.Equal(t, &harness.PendingAnswer{RequestID: "req-1", Answer: yesAnswer()}, req.Answer,
		"the harness resolves the question the ended turn left open")
	assert.Equal(t, trail.ConversationID, req.ConversationID)
	assert.Equal(t, starter, req.ViaUserID)
	assert.Equal(t, &agent.PlayContext{Label: "Fix with AI", Instructions: "Fix the ticket.",
		Memories: []agent.MemoryRef{{ID: alwaysMem, Name: "Working here"}, {ID: pickedMem, Name: "Deploy quirks"}}},
		req.Play, "a session the harness lost hears the play and the recorded memories again")
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

func TestAnswer_TurnGone_PlayDeletedSince_ResumesWithoutThePlay(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")
	delete(f.plays.byID, fixPlayID)

	_, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	<-f.turns.done
	play := f.turns.last().Play
	require.NotNil(t, play, "still a play turn, so its prompt says so")
	assert.Empty(t, play.Label)
	assert.Equal(t, []agent.MemoryRef{{ID: alwaysMem, Name: "Working here"}}, play.Memories)
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

func TestAnswer_HarnessAlreadyAnswered_RunCarriesOn(t *testing.T) {
	f := heldFixture(t)
	f.turns.answerErr = apperrs.ErrConflict
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())

	got, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	assert.Len(t, f.turns.reqs, 1, "the live turn already has its answer; no fresh turn is started")
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
		replies, pipelineNotes := convs.snapshot()
		assert.Empty(t, pipelineNotes, "the pipeline posts nothing while a play waits on its question")
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

func interviewRun() RunInput {
	return RunInput{PlayID: intPlayID, TargetType: TargetInterview, TargetID: projectID, Via: ViaWeb}
}

func followUpRound(requestID string) harness.Question {
	return harness.Question{RequestID: requestID, Questions: []harness.QuestionItem{
		{ID: "runner", Text: "Which test runner? ci.yml runs both bun test and vitest.", Header: "Tests",
			Options: []harness.QuestionOption{{Label: "Vitest (Recommended)", Description: "web/", Value: "vitest"}, {Label: "Bun"}}},
		{ID: "gates", Text: "Which gates block a merge?\nYou skipped this in the template.", Header: "Gates", MultiSelect: true,
			Options: []harness.QuestionOption{{Label: "Lint"}, {Label: "Coverage"}}},
		{ID: "floor", Text: "What coverage floor?", Header: "Coverage"},
	}}
}

func TestAnswer_InterviewRun_RecordsEachRoundInOrder(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, interviewRun())
	obs.OnStarted("sess-1")

	obs.OnQuestion(followUpRound("req-1"))
	_, err := f.runner.Answer(ctxAs(starter), trail.ID, harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{
		"runner": {Selected: []string{"vitest"}}, "gates": {Selected: []string{"Lint", "Coverage"}}, "floor": {},
	}})
	require.NoError(t, err)

	obs.OnQuestion(harness.Question{RequestID: "req-2", Questions: []harness.QuestionItem{{ID: "q", Text: "Keep the 80 floor?"}}})
	obs.OnFinished(harness.TurnResult{State: harness.TurnDone}, "")
	_, err = f.runner.Answer(ctxAs(starter), trail.ID, harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q": {Text: "Raise it to 85"}}})
	require.NoError(t, err)

	round1 := followUpRound("req-1").Questions
	assert.Equal(t, []fakeRound{
		{projectID: projectID, answeredBy: starter, followUps: []FollowUp{
			{Question: "Which test runner?", Why: "ci.yml runs both bun test and vitest.", Options: round1[0].Options, Selected: []string{"Vitest (Recommended)"}},
			{Question: "Which gates block a merge?", Why: "You skipped this in the template.", Options: round1[1].Options, MultiSelect: true, Selected: []string{"Lint", "Coverage"}},
			{Question: "What coverage floor?", Selected: []string{}},
		}},
		{projectID: projectID, answeredBy: starter, followUps: []FollowUp{{Question: "Keep the 80 floor?", Selected: []string{}, Text: "Raise it to 85"}}},
	}, f.answers.snapshot(), "a live answer and one that resumes the run each land as the next round, the why split off the text and never taken from the header, picks named by label, an empty answer left for the store to skip")
}

func TestAnswer_InterviewRun_RecordsTheSkipTextAsASkip(t *testing.T) {
	tests := []struct {
		name   string
		answer harness.AnswerValue
		want   FollowUp
	}{
		{"empty answer", harness.AnswerValue{}, FollowUp{Question: "What coverage floor?", Selected: []string{}}},
		{"the skip text", harness.AnswerValue{Text: "Skipped"}, FollowUp{Question: "What coverage floor?", Selected: []string{}}},
		{"the skip text padded", harness.AnswerValue{Text: " Skipped\n"}, FollowUp{Question: "What coverage floor?", Selected: []string{}}},
		{"the skip text inside an answer", harness.AnswerValue{Text: "Skipped for now"}, FollowUp{Question: "What coverage floor?", Selected: []string{}, Text: "Skipped for now"}},
		{"the skip text beside a pick", harness.AnswerValue{Selected: []string{"80"}, Text: "Skipped"}, FollowUp{Question: "What coverage floor?", Selected: []string{"80"}, Text: "Skipped"}},
		{"the skip text in lower case", harness.AnswerValue{Text: "skipped"}, FollowUp{Question: "What coverage floor?", Selected: []string{}, Text: "skipped"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := heldFixture(t)
			trail, obs := driveTurn(t, f, interviewRun())
			obs.OnStarted("sess-1")
			obs.OnQuestion(harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{ID: "floor", Text: "What coverage floor?"}}})

			_, err := f.runner.Answer(ctxAs(starter), trail.ID, harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"floor": tt.answer}})
			require.NoError(t, err)
			assert.Equal(t, []fakeRound{{projectID: projectID, answeredBy: starter, followUps: []FollowUp{tt.want}}}, f.answers.snapshot())
			assert.Equal(t, []harness.QuestionAnswer{{Answers: map[string]harness.AnswerValue{"floor": tt.answer}}}, f.turns.answered, "the harness gets the answer as sent")
		})
	}
}

func TestAnswer_TicketAndDocRuns_RecordNoFollowUps(t *testing.T) {
	for name, in := range map[string]RunInput{"ticket": ticketRun(), "doc": {PlayID: docPlayID, TargetType: TargetDoc, TargetID: docID, Via: ViaWeb}} {
		t.Run(name, func(t *testing.T) {
			f := heldFixture(t)
			trail, obs := driveTurn(t, f, in)
			obs.OnStarted("sess-1")
			obs.OnQuestion(askedQuestion())

			_, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
			require.NoError(t, err)
			assert.Empty(t, f.answers.snapshot())
		})
	}
}

func TestAnswer_InterviewRecordFails_TheRunStillGetsItsAnswer(t *testing.T) {
	f := heldFixture(t)
	f.answers.err = errors.New("database is locked")
	trail, obs := driveTurn(t, f, interviewRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())

	got, err := f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	assert.Equal(t, []harness.QuestionAnswer{yesAnswer()}, f.turns.answered)
}

func TestSplitWhy(t *testing.T) {
	tests := []struct {
		name, text, question, why string
	}{
		{"no question mark", "Pick a runner", "Pick a runner", ""},
		{"question mark at the end", "Which runner?", "Which runner?", ""},
		{"question then why", "When are tests written? You skipped this, and most commits add a test file.", "When are tests written?", "You skipped this, and most commits add a test file."},
		{"question mark inside a word", "Is it e.g.?x or y? Both appear in ci.yml.", "Is it e.g.?x or y?", "Both appear in ci.yml."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			question, why := splitWhy(tt.text)
			assert.Equal(t, tt.question, question)
			assert.Equal(t, tt.why, why)
		})
	}
}

func TestObserver_AnsweredInT3_TakesTheAnswerAndRunsAgain(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("sess-1")
	obs.OnQuestion(askedQuestion())
	typed := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Text: "Use the VM"}}}

	obs.OnAnswered(harness.AnsweredQuestion{RequestID: "req-other", Answer: typed})
	assert.Equal(t, TrailWaiting, f.trails.all()[0].State, "another question's answer is not this one's")

	obs.OnAnswered(harness.AnsweredQuestion{RequestID: "req-1", Answer: typed})
	got, err := f.trails.GetTrail(t.Context(), trail.ID)
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	require.NotNil(t, got.Question.Answer)
	assert.Equal(t, typed, *got.Question.Answer, "the card shows what was typed in T3")
	assert.Empty(t, f.turns.answered, "T3 already has it; nothing is sent back")
	_, err = f.runner.Answer(ctxAs(starter), trail.ID, yesAnswer())
	assert.ErrorIs(t, err, apperrs.ErrConflict, "the card no longer takes an answer")
}

func TestFollowThread_NewsOnAThreadParkedOnAQuestion_ReopensItForTheAnswerGivenInT3(t *testing.T) {
	f := newRunnerFixture()
	seedTrail(f, "tr-asked", TrailWaiting, "th-1")
	f.trails.byID["tr-asked"].Question = &TrailQuestion{Question: askedQuestion(), AskedAt: fixedNow}

	done, ok := f.runner.FollowThread(t.Context(), "conv-tr-asked", "th-1", starter, "pc-1", "run-7")
	require.True(t, ok, "the turn closed under the question, so nothing else is watching the thread")
	<-f.turns.done
	req := f.turns.last()
	assert.True(t, req.Watch)
	req.Observer.OnStarted("th-1")
	typed := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Text: "Use the VM"}}}
	req.Observer.OnAnswered(harness.AnsweredQuestion{RequestID: "req-1", Answer: typed})

	got, err := f.trails.GetTrail(t.Context(), "tr-asked")
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, got.State)
	require.NotNil(t, got.Question.Answer)
	assert.Equal(t, typed, *got.Question.Answer)
	req.Observer.OnFinished(harness.TurnResult{State: harness.TurnDone}, "reply-2")
	<-done
}

func TestFollowThread_QuestionStillOnALiveTurn_LeavesItToThatTurn(t *testing.T) {
	f := heldFixture(t)
	trail, obs := driveTurn(t, f, ticketRun())
	obs.OnStarted("th-1")
	obs.OnQuestion(askedQuestion())
	waiting := f.trails.all()[0]

	_, ok := f.runner.FollowThread(t.Context(), trail.ConversationID, "th-1", starter, waiting.ComputerID, "run-7")

	assert.False(t, ok, "the live turn hears the answer itself")
}

package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type fakeObserver struct {
	mu        sync.Mutex
	sessionID string
	started   bool
	activity  []harness.Activity
	snapshots int
	questions []harness.Question
	result    *harness.TurnResult
	replyID   string
}

func (f *fakeObserver) OnStarted(sessionID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started, f.sessionID = true, sessionID
}

func (f *fakeObserver) OnActivity(a harness.Activity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activity = append(f.activity, a)
}

func (f *fakeObserver) OnSnapshot() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshots++
}

func (f *fakeObserver) OnQuestion(q harness.Question) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.questions = append(f.questions, q)
}

func (f *fakeObserver) OnFinished(result harness.TurnResult, replyID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.result, f.replyID = &result, replyID
}

func TestRunTurn_Observer_SeesSessionActivityAndReply(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{
		SessionID: "sess-1",
		Updates: updatesChan(
			harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Read", Summary: `{"file_path":"main.go"}`, Detail: `{"input":{"file_path":"main.go"}}`}},
			harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolResult, CallID: "c-2", Tool: "Bash", Summary: "go test"}},
			harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "done!", Streaming: false}},
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
		),
	}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	assert.True(t, obs.started)
	assert.Equal(t, "sess-1", obs.sessionID)
	require.Len(t, obs.activity, 3, "two tool steps and the closed reply as a text step")
	assert.Equal(t, harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Read", Summary: `{"file_path":"main.go"}`, Detail: `{"input":{"file_path":"main.go"}}`}, obs.activity[0], "the observer sees the whole activity, not a label")
	assert.Equal(t, "Bash", obs.activity[1].Tool)
	assert.Equal(t, harness.ActivityText, obs.activity[2].Kind)
	assert.Equal(t, "done!", obs.activity[2].Summary)
	assert.Equal(t, "done!", obs.activity[2].Detail)
	assert.False(t, obs.activity[2].At.IsZero())
	assert.Equal(t, 1, obs.snapshots)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnDone, obs.result.State)
	assert.Equal(t, "reply-1", obs.replyID)
}

func TestRunTurn_Observer_InterleavedText_BecomesStepsBetweenTheToolCalls(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Reading the handler.", Streaming: true}},
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Read", Summary: "handler.go"}},
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolResult, CallID: "c-1", Tool: "Read", Summary: "handler.go"}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Reading the handler.\n\nNow the tests.", Streaming: true}},
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-2", Tool: "Bash", Summary: "go test"}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Reading the handler.\n\nNow the tests.\n\nAll green.", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	var kinds []harness.ActivityKind
	var texts []string
	for _, a := range obs.activity {
		kinds = append(kinds, a.Kind)
		if a.Kind == harness.ActivityText {
			texts = append(texts, a.Detail)
		}
	}
	assert.Equal(t, []harness.ActivityKind{harness.ActivityText, harness.ActivityToolCall, harness.ActivityToolResult, harness.ActivityText, harness.ActivityToolCall, harness.ActivityText}, kinds)
	assert.Equal(t, []string{"Reading the handler.", "Now the tests.", "All green."}, texts, "each piece once, the remainder at the close, never the whole reply again")
	replies, _ := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, "Reading the handler.\n\nNow the tests.\n\nAll green.", replies[0].body, "the chat reply stays the whole text")
}

func TestRunTurn_Observer_SegmentPerMessage_CloseMarkerEmitsOnce(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "First thought.", Streaming: true}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "First thought.", Streaming: false}},
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Bash", Summary: "ls"}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-2", Text: "Second thought.", Streaming: true}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-2", Text: "", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	var texts []string
	for _, a := range obs.activity {
		if a.Kind == harness.ActivityText {
			texts = append(texts, a.Summary)
		}
	}
	assert.Equal(t, []string{"First thought.", "Second thought."}, texts, "a closed message is one step; the tool call after it adds nothing; an empty close marker still flushes the message")
}

func TestRunTurn_Observer_ReusedSessionReportsTheThreadID(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "thread-old"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "thread-old", Updates: updatesChan(
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnInterrupted}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	assert.Equal(t, "thread-old", obs.sessionID)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnInterrupted, obs.result.State)
	assert.Empty(t, obs.replyID, "no reply text means no reply id")
}

func TestRunTurn_Observer_StartTurnError_FinishesFailedWithReason(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startErr: errors.New("harness refused")}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	assert.False(t, obs.started)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnError, obs.result.State)
	assert.Equal(t, "harness refused", obs.result.LastError)
}

func TestRunTurn_Observer_ResolveFailure_FinishesFailed(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{err: errors.New("no computer")}, Harnesses: registryOf(&fakeHarness{}), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnError, obs.result.State)
	assert.Equal(t, "no computer", obs.result.LastError)
}

func TestRunTurn_CancelledWithCause_LeavesTheNoteToTheCanceller(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: make(chan harness.Update)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("no harness update for 15m"))

	svc.RunTurn(ctx, TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go", Observer: obs})

	_, notes := conv.snapshot()
	assert.Empty(t, notes, "the canceller already told the thread why")
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnError, obs.result.State)
}

func TestRunTurn_PlainCancel_StillPostsTheFailureNote(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: make(chan harness.Update)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	svc.RunTurn(ctx, TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go"})

	_, notes := conv.snapshot()
	require.Len(t, notes, 1)
	assert.Contains(t, notes[0].body, "Agent turn failed")
}

func TestRunTurn_NoObserver_StillRuns(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "ok", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "go"})

	replies, _ := conv.snapshot()
	require.Len(t, replies, 1)
}

func question() harness.Question {
	return harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{ID: "q1", Text: "Proceed?", Options: []harness.QuestionOption{{Label: "Yes"}, {Label: "No"}}}}}
}

func TestRunTurn_Question_PostsTheCardAndTellsTheObserver(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	q := question()
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: updatesChan(
		harness.Update{Question: &q},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go", Observer: obs})

	assert.Equal(t, []harness.Question{q}, obs.questions)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnDone, obs.result.State, "a question is not terminal; the turn's own end still arrives")
	replies, _ := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, QuestionMessageBody(q), replies[0].body, "the question lands as the Agent's own message")
	assert.Contains(t, replies[0].body, "```nexul-question\n{\"request_id\":\"req-1\"")
}

func TestAnswer_NoActiveTurn_NotFound(t *testing.T) {
	svc := NewService(Config{Conversations: newFakeConversations(Conversation{ID: "conv-1"}), Live: &fakeLive{}})
	err := svc.Answer(t.Context(), "conv-1", "req-1", harness.QuestionAnswer{})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestAnswer_ActiveTurn_ForwardsToTheHarness(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	updates := make(chan harness.Update)
	var got struct {
		mu        sync.Mutex
		target    harness.Target
		requestID string
		answer    harness.QuestionAnswer
	}
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-1", Updates: updates}}
	client.AnswerFn = func(_ context.Context, target harness.Target, requestID string, answer harness.QuestionAnswer) error {
		got.mu.Lock()
		defer got.mu.Unlock()
		got.target, got.requestID, got.answer = target, requestID, answer
		return nil
	}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})
	}()
	waitFor(t, time.Second, func() bool { svc.mu.Lock(); defer svc.mu.Unlock(); return len(svc.active) == 1 })

	answer := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
	require.NoError(t, svc.Answer(t.Context(), "conv-1", "req-1", answer))
	got.mu.Lock()
	assert.Equal(t, "sess-1", got.target.SessionID)
	assert.Equal(t, "req-1", got.requestID)
	assert.Equal(t, answer, got.answer)
	got.mu.Unlock()

	updates <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	close(updates)
	<-done
}

func TestAnswerFromChat_TurnGone_PostsTheAnswerAndResumesAFreshTurn(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "sess-old"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "sess-old", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "carrying on", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})

	answer := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
	require.NoError(t, svc.AnswerFromChat(t.Context(), "conv-1", "u-2", "req-1", answer))

	conv.mu.Lock()
	require.Len(t, conv.userPosts, 1)
	assert.Equal(t, fakeNote{"conv-1", "u-2", "Answered: Yes"}, conv.userPosts[0])
	conv.mu.Unlock()
	waitFor(t, time.Second, func() bool { replies, _ := conv.snapshot(); return len(replies) == 1 })
	assert.Contains(t, client.snapshotPrompt(), "Answered: Yes", "the fresh turn carries the answer as its request")
	client.mu.Lock()
	assert.Equal(t, "sess-old", client.lastTarget.SessionID, "the fresh turn reuses the conversation's session")
	client.mu.Unlock()
}

func TestAnswerFromChat_Refusals(t *testing.T) {
	svc := NewService(Config{Conversations: newFakeConversations(Conversation{ID: "conv-1"}), Live: &fakeLive{}})
	answer := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Text: "x"}}}
	require.ErrorIs(t, svc.AnswerFromChat(t.Context(), "conv-1", "u-1", " ", answer), apperrs.ErrInvalid)
	require.ErrorIs(t, svc.AnswerFromChat(t.Context(), "conv-1", "u-1", "req-1", harness.QuestionAnswer{}), apperrs.ErrInvalid)
}

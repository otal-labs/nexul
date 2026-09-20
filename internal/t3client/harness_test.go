package t3client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeSubscription is a threadSub test double: pump reads it in a
// goroutine, so buffer generously and close it explicitly per test.
type fakeSubscription struct {
	ch     chan Update
	closed bool
}

func newFakeSubscription(updates ...Update) *fakeSubscription {
	ch := make(chan Update, len(updates)+1)
	for _, u := range updates {
		ch <- u
	}
	close(ch)
	return &fakeSubscription{ch: ch}
}

func (f *fakeSubscription) Updates() <-chan Update { return f.ch }
func (f *fakeSubscription) Close()                 { f.closed = true }

// fakeT3Client is a rpcConn test double letting tests script per-call
// behavior without a real T3 server (mirrors ticket 12's own fake_test.go
// approach, one level up the seam).
type fakeT3Client struct {
	createThreadCalls  int
	createThreadErr    error
	createThreadModels []string
	nextThreadID       string

	subscribeErr      error
	subscription      *fakeSubscription
	subscribeFailOnce bool
	subscribeCalls    int

	startTurnErr      error
	startTurnFailOnce bool
	startTurnCalls    int
	sentPrompts       []string
	sentAttachments   [][]harness.Attachment

	interruptErr      error
	interruptThreadID string

	respondApprovalCalls []string                          // requestIDs
	answered             map[string]harness.QuestionAnswer // requestID -> answer
	answeredThreadID     string

	providers    []harness.Provider
	providersErr error

	closed bool
}

func testPrompts() harness.TurnPrompts {
	return harness.TurnPrompts{Full: "full-prompt", Incremental: "incremental-prompt"}
}

func (f *fakeT3Client) CreateThread(_ context.Context, _, _, _, model, _ string) (string, error) {
	f.createThreadCalls++
	f.createThreadModels = append(f.createThreadModels, model)
	if f.createThreadErr != nil {
		return "", f.createThreadErr
	}
	return f.nextThreadID, nil
}

func (f *fakeT3Client) StartTurn(_ context.Context, _, text, _ string, attachments []harness.Attachment) error {
	f.startTurnCalls++
	f.sentPrompts = append(f.sentPrompts, text)
	f.sentAttachments = append(f.sentAttachments, attachments)
	if f.startTurnFailOnce && f.startTurnCalls == 1 {
		return errors.New("turn start failed")
	}
	return f.startTurnErr
}

func (f *fakeT3Client) Interrupt(_ context.Context, threadID string) error {
	f.interruptThreadID = threadID
	return f.interruptErr
}

func (f *fakeT3Client) RespondApproval(_ context.Context, _, requestID, _ string) error {
	f.respondApprovalCalls = append(f.respondApprovalCalls, requestID)
	return nil
}

func (f *fakeT3Client) RespondUserInput(_ context.Context, threadID, requestID string, answer harness.QuestionAnswer) error {
	if f.answered == nil {
		f.answered = map[string]harness.QuestionAnswer{}
	}
	f.answeredThreadID = threadID
	f.answered[requestID] = answer
	return nil
}

func (f *fakeT3Client) SubscribeThread(_ context.Context, _ string) (threadSub, error) {
	f.subscribeCalls++
	if f.subscribeFailOnce && f.subscribeCalls == 1 {
		return nil, errors.New("subscribe failed")
	}
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	return f.subscription, nil
}

func (f *fakeT3Client) Providers() ([]harness.Provider, error) {
	return f.providers, f.providersErr
}

func (f *fakeT3Client) Close() error {
	f.closed = true
	return nil
}

func harnessWithFake(fake *fakeT3Client) *Harness {
	b := NewHarness(Options{})
	b.connect = func(_ context.Context, _ harness.Session, _ Options) (rpcConn, error) {
		return fake, nil
	}
	return b
}

func drain(t *testing.T, updates <-chan harness.Update) []harness.Update {
	t.Helper()
	var out []harness.Update
	for u := range updates {
		out = append(out, u)
	}
	return out
}

func TestHarness_StartTurn_CreatesThreadWhenSessionEmpty(t *testing.T) {
	fake := &fakeT3Client{
		nextThreadID: "thread-new",
		subscription: newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}}),
	}
	b := harnessWithFake(fake)

	result, err := b.StartTurn(context.Background(), harness.Target{}, "title", testPrompts())
	require.NoError(t, err)
	assert.Equal(t, "thread-new", result.SessionID)
	assert.Equal(t, 1, fake.createThreadCalls)
	assert.Equal(t, []string{"full-prompt"}, fake.sentPrompts, "a fresh thread gets the full prompt")
	updates := drain(t, result.Updates)
	require.Len(t, updates, 1)
	assert.Equal(t, harness.TurnDone, updates[0].Terminal.State)
	assert.True(t, fake.closed)
}

func TestHarness_StartTurn_ReusesStoredSessionID(t *testing.T) {
	fake := &fakeT3Client{
		subscription: newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}}),
	}
	b := harnessWithFake(fake)

	result, err := b.StartTurn(context.Background(), harness.Target{SessionID: "thread-existing"}, "title", testPrompts())
	require.NoError(t, err)
	assert.Equal(t, "thread-existing", result.SessionID)
	assert.Equal(t, 0, fake.createThreadCalls)
	assert.Equal(t, []string{"incremental-prompt"}, fake.sentPrompts, "a reused thread already holds the instructions and history")
	drain(t, result.Updates)
}

func TestHarness_StartTurn_RecreatesGoneThreadAndRetriesOnce(t *testing.T) {
	fake := &fakeT3Client{
		nextThreadID:      "thread-fresh",
		subscribeFailOnce: true,
		subscription:      newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}}),
	}
	b := harnessWithFake(fake)

	result, err := b.StartTurn(context.Background(), harness.Target{SessionID: "thread-gone"}, "title", testPrompts())
	require.NoError(t, err)
	assert.Equal(t, "thread-fresh", result.SessionID)
	assert.Equal(t, 1, fake.createThreadCalls)
	assert.Equal(t, 2, fake.subscribeCalls)
	assert.Equal(t, []string{"full-prompt"}, fake.sentPrompts, "the fresh replacement thread has no context — full prompt")
	drain(t, result.Updates)
}

func TestHarness_StartTurn_AttachmentsRideAlongOnTheFreshThreadRetry(t *testing.T) {
	fake := &fakeT3Client{
		nextThreadID:      "thread-fresh",
		startTurnFailOnce: true,
		subscription:      newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}}),
	}
	b := harnessWithFake(fake)
	prompts := testPrompts()
	prompts.Attachments = []harness.Attachment{{Name: "shot.png", MIME: "image/png", Bytes: []byte{1, 2, 3}}}

	result, err := b.StartTurn(context.Background(), harness.Target{SessionID: "thread-gone"}, "title", prompts)
	require.NoError(t, err)
	assert.Equal(t, "thread-fresh", result.SessionID)
	require.Len(t, fake.sentAttachments, 2)
	assert.Equal(t, prompts.Attachments, fake.sentAttachments[0], "the reused thread gets the attachments")
	assert.Equal(t, prompts.Attachments, fake.sentAttachments[1], "so does the fresh replacement thread")
	drain(t, result.Updates)
}

func TestHarness_StartTurn_FreshThreadFailureDoesNotRetryForever(t *testing.T) {
	fake := &fakeT3Client{
		nextThreadID: "thread-new",
		subscribeErr: errors.New("still broken"),
	}
	b := harnessWithFake(fake)

	_, err := b.StartTurn(context.Background(), harness.Target{}, "title", testPrompts())
	require.Error(t, err)
	// SessionID was empty (fresh thread already), so a failure is not retried.
	assert.Equal(t, 1, fake.createThreadCalls)
	assert.Equal(t, 1, fake.subscribeCalls)
	assert.True(t, fake.closed)
}

func TestHarness_StartTurn_ForwardsToolActivity(t *testing.T) {
	fake := &fakeT3Client{
		subscription: newFakeSubscription(
			Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Read", Summary: `{"file_path":"main.go"}`}},
			Update{Terminal: &TurnResult{State: TurnDone}},
		),
	}
	b := harnessWithFake(fake)

	result, err := b.StartTurn(context.Background(), harness.Target{SessionID: "thread-1"}, "title", testPrompts())
	require.NoError(t, err)
	updates := drain(t, result.Updates)
	require.Len(t, updates, 2)
	assert.Equal(t, &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Read", Summary: `{"file_path":"main.go"}`}, updates[0].Activity)
}

func TestHarness_StartTurn_AutoDeclinesApprovalAndForwardsIt(t *testing.T) {
	fake := &fakeT3Client{
		subscription: newFakeSubscription(
			Update{Approval: &ApprovalRequest{RequestID: "req-1", Kind: "shell", Summary: "run rm -rf"}},
			Update{Terminal: &TurnResult{State: TurnDone}},
		),
	}
	b := harnessWithFake(fake)

	result, err := b.StartTurn(context.Background(), harness.Target{SessionID: "thread-1"}, "title", testPrompts())
	require.NoError(t, err)
	updates := drain(t, result.Updates)
	require.Len(t, updates, 2)
	require.NotNil(t, updates[0].Approval)
	assert.Equal(t, "shell", updates[0].Approval.Kind)
	assert.Equal(t, []string{"req-1"}, fake.respondApprovalCalls)
}

func TestHarness_StartTurn_EmptyModelResolvesProviderDefault(t *testing.T) {
	fake := &fakeT3Client{
		nextThreadID: "thread-new",
		providers: []harness.Provider{
			{ID: "claude-code", Models: []harness.ProviderModel{
				{Slug: "haiku-4"},
				{Slug: "opus-4", IsDefault: true},
			}},
		},
		subscription: newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}}),
	}
	b := harnessWithFake(fake)

	_, err := b.StartTurn(context.Background(), harness.Target{Provider: "claude-code"}, "title", testPrompts())
	require.NoError(t, err)
	assert.Equal(t, []string{"opus-4"}, fake.createThreadModels)
}

func TestHarness_StartTurn_EmptyModelFallsBackToFirstWhenNoDefault(t *testing.T) {
	fake := &fakeT3Client{
		nextThreadID: "thread-new",
		providers: []harness.Provider{
			{ID: "claude-code", Models: []harness.ProviderModel{{Slug: "haiku-4"}, {Slug: "opus-4"}}},
		},
		subscription: newFakeSubscription(Update{Terminal: &TurnResult{State: TurnDone}}),
	}
	b := harnessWithFake(fake)

	_, err := b.StartTurn(context.Background(), harness.Target{Provider: "claude-code"}, "title", testPrompts())
	require.NoError(t, err)
	assert.Equal(t, []string{"haiku-4"}, fake.createThreadModels)
}

func TestHarness_StartTurn_EmptyModelWithNoModelsIsRefused(t *testing.T) {
	fake := &fakeT3Client{
		providers: []harness.Provider{{ID: "claude-code", Models: nil}},
	}
	b := harnessWithFake(fake)

	_, err := b.StartTurn(context.Background(), harness.Target{Provider: "claude-code"}, "title", testPrompts())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Equal(t, 0, fake.createThreadCalls)
}

func TestHarness_StartTurn_EmptyModelWithUnknownProviderIsRefused(t *testing.T) {
	fake := &fakeT3Client{providers: nil}
	b := harnessWithFake(fake)

	_, err := b.StartTurn(context.Background(), harness.Target{Provider: "ghost"}, "title", testPrompts())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestHarness_Interrupt_RequiresSessionID(t *testing.T) {
	b := harnessWithFake(&fakeT3Client{})
	err := b.Interrupt(context.Background(), harness.Target{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestHarness_Interrupt_CallsClientWithSessionID(t *testing.T) {
	fake := &fakeT3Client{}
	b := harnessWithFake(fake)
	require.NoError(t, b.Interrupt(context.Background(), harness.Target{SessionID: "thread-1"}))
	assert.Equal(t, "thread-1", fake.interruptThreadID)
	assert.True(t, fake.closed)
}

func TestHarness_Answer_RequiresSessionID(t *testing.T) {
	b := harnessWithFake(&fakeT3Client{})
	err := b.Answer(context.Background(), harness.Target{}, "req-1", harness.QuestionAnswer{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestHarness_Answer_RespondsOnTheSessionAndCloses(t *testing.T) {
	fake := &fakeT3Client{}
	b := harnessWithFake(fake)
	answer := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
	require.NoError(t, b.Answer(context.Background(), harness.Target{SessionID: "thread-1"}, "req-1", answer))
	assert.Equal(t, "thread-1", fake.answeredThreadID)
	assert.Equal(t, answer, fake.answered["req-1"])
	assert.True(t, fake.closed)
}

func TestHarness_StartTurn_ForwardsQuestions(t *testing.T) {
	q := &harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{ID: "q1", Text: "Proceed?"}}}
	fake := &fakeT3Client{nextThreadID: "th-1", subscription: newFakeSubscription(Update{Question: q}, Update{Terminal: &TurnResult{State: TurnDone}})}
	b := harnessWithFake(fake)
	res, err := b.StartTurn(context.Background(), harness.Target{}, "title", testPrompts())
	require.NoError(t, err)
	updates := drain(t, res.Updates)
	require.Len(t, updates, 2)
	assert.Equal(t, q, updates[0].Question)
}

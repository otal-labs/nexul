package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// --- fakes -----------------------------------------------------------------

type fakePost struct {
	conversationID, viaUserID, body string
	handoffs                        []harness.Handoff
}

type fakeConversations struct {
	mu sync.Mutex

	conv    Conversation
	getErr  error
	history []ConversationMessage
	histErr error

	threads     map[string]string
	syncedAt    map[string]time.Time
	replies     []fakePost
	systemPosts []fakePost
	userPosts   []fakePost
	relayed     []fakeRelay
	seen        map[string]string
	// now, when set, stamps each posted Agent reply into history the way chat stores it.
	now           func() time.Time
	postReplyErr  error
	postSystemErr error
	setThreadErr  error
	markSyncedErr error
}

func newFakeConversations(conv Conversation) *fakeConversations {
	return &fakeConversations{conv: conv, threads: map[string]string{}, syncedAt: map[string]time.Time{}, seen: map[string]string{}}
}

func (f *fakeConversations) GetConversation(_ context.Context, id string) (Conversation, error) {
	if f.getErr != nil {
		return Conversation{}, f.getErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.conv
	if thread, ok := f.threads[id]; ok {
		c.ThreadID = thread
	}
	if at, ok := f.syncedAt[id]; ok {
		c.SyncedAt = at
	}
	if marker, ok := f.seen[id]; ok {
		c.SeenMarker = marker
	}
	return c, nil
}

func (f *fakeConversations) MarkSeen(_ context.Context, conversationID, marker string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen[conversationID] = marker
	return nil
}

// ConversationByThread finds the one conversation the fake holds when its thread is threadID.
func (f *fakeConversations) ConversationByThread(ctx context.Context, threadID string) (Conversation, error) {
	c, err := f.GetConversation(ctx, f.conv.ID)
	if err != nil {
		return Conversation{}, err
	}
	if c.ThreadID != threadID {
		return Conversation{}, apperrs.ErrNotFound
	}
	return c, nil
}

func (f *fakeConversations) seenMarker(id string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seen[id]
}

func (f *fakeConversations) MessagesSince(_ context.Context, _ string, since time.Time) ([]ConversationMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ConversationMessage
	for _, m := range f.history {
		if since.IsZero() || m.CreatedAt.After(since) {
			out = append(out, m)
		}
	}
	return out, f.histErr
}

func (f *fakeConversations) SetThread(_ context.Context, conversationID, threadID string) error {
	if f.setThreadErr != nil {
		return f.setThreadErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.threads[conversationID] = threadID
	return nil
}

func (f *fakeConversations) MarkSynced(_ context.Context, conversationID string, at time.Time) error {
	if f.markSyncedErr != nil {
		return f.markSyncedErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.syncedAt[conversationID] = at
	return nil
}

func (f *fakeConversations) PostAgentReply(_ context.Context, conversationID, viaUserID, body string, handoffs []harness.Handoff) (string, error) {
	if f.postReplyErr != nil {
		return "", f.postReplyErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies = append(f.replies, fakePost{conversationID, viaUserID, body, handoffs})
	if f.now != nil {
		f.history = append(f.history, ConversationMessage{AuthorID: viaUserID, AuthorKind: "agent", Body: body, CreatedAt: f.now()})
	}
	return fmt.Sprintf("reply-%d", len(f.replies)), nil
}

func (f *fakeConversations) PostSystemMessage(_ context.Context, conversationID, viaUserID, body string) error {
	if f.postSystemErr != nil {
		return f.postSystemErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.systemPosts = append(f.systemPosts, fakePost{conversationID: conversationID, viaUserID: viaUserID, body: body})
	return nil
}

func (f *fakeConversations) PostUserMessage(_ context.Context, conversationID, userID, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userPosts = append(f.userPosts, fakePost{conversationID: conversationID, viaUserID: userID, body: body})
	return nil
}

// fakeRelay is one message relayed from the harness, as PostHarnessMessage received it.
type fakeRelay struct {
	conversationID, userID, body, via, key string
	at                                     time.Time
}

func (f *fakeConversations) PostHarnessMessage(_ context.Context, conversationID, userID, body, via, key string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.relayed = append(f.relayed, fakeRelay{conversationID, userID, body, via, key, at})
	return nil
}

func (f *fakeConversations) snapshot() ([]fakePost, []fakePost) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakePost{}, f.replies...), append([]fakePost{}, f.systemPosts...)
}

type fakeTargets struct {
	target *pairing.ResolvedTarget
	err    error
	// override records the last ResolveTargetOverride call, so a test can assert an override reached the seam.
	override *TargetOverride
	hang     bool
}

func (f *fakeTargets) ResolveConfirmedTarget(_ context.Context, _ string, target pairing.ResolvedTarget) (*pairing.ResolvedTarget, error) {
	return &target, f.err
}

func (f *fakeTargets) ResolveTarget(ctx context.Context, _, _ string) (*pairing.ResolvedTarget, error) {
	if f.hang {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.target, f.err
}

func (f *fakeTargets) ResolveTargetOverride(_ context.Context, _, _, computerID, provider, model string, options []harness.OptionSetting) (*pairing.ResolvedTarget, error) {
	f.override = &TargetOverride{ComputerID: computerID, Provider: provider, Model: model, ModelOptions: options}
	return f.target, f.err
}

type fakeDocs struct {
	doc Doc
	err error
}

func (f *fakeDocs) Get(_ context.Context, _ string) (Doc, error) {
	return f.doc, f.err
}

type fakeTickets struct {
	ticket Ticket
	err    error
}

func (f *fakeTickets) Get(_ context.Context, _ string) (Ticket, error) {
	return f.ticket, f.err
}

type fakeMemories struct {
	mu            sync.Mutex
	called        bool
	calledProject string
	items         []MemoryItem
	err           error
}

func (f *fakeMemories) ListMemories(_ context.Context, projectID string) ([]MemoryItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = true
	f.calledProject = projectID
	return f.items, f.err
}

type fakeHarness struct {
	harnesstest.Client
	mu sync.Mutex

	startErr        error
	startResult     harness.StartResult
	interruptCh     chan struct{}
	interruptErr    error
	lastTarget      harness.Target
	lastTitle       string
	lastPrompt      string
	lastIncremental string
	lastAnswer      *harness.PendingAnswer
	watchResult     harness.StartResult
	watchErr        error
	watched         []harness.Target
}

func (f *fakeHarness) Watch(_ context.Context, target harness.Target) (harness.StartResult, error) {
	f.mu.Lock()
	f.watched = append(f.watched, target)
	f.mu.Unlock()
	return f.watchResult, f.watchErr
}

func (f *fakeHarness) StartTurn(_ context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	f.mu.Lock()
	f.lastTarget = target
	f.lastTitle = title
	f.lastPrompt = prompts.Full
	f.lastIncremental = prompts.Incremental
	f.lastAnswer = prompts.Answer
	f.mu.Unlock()
	return f.startResult, f.startErr
}

func (f *fakeHarness) snapshotPrompt() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastPrompt
}

func (f *fakeHarness) Interrupt(_ context.Context, _ harness.Target) error {
	if f.interruptCh != nil {
		close(f.interruptCh)
	}
	return f.interruptErr
}

type fakeLive struct {
	mu     sync.Mutex
	frames []StreamFrame
}

func (f *fakeLive) Publish(_ context.Context, _ string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if raw, ok := payload.(json.RawMessage); ok {
		var frame StreamFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			return err
		}
		payload = frame
	}
	f.frames = append(f.frames, payload.(StreamFrame))
	return nil
}

func (f *fakeLive) snapshot() []StreamFrame {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]StreamFrame{}, f.frames...)
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition never became true")
}

func registryOf(h *fakeHarness) harness.Registry {
	return harness.Registry{harness.KindT3Code: h}
}

func testTarget() *pairing.ResolvedTarget {
	return &pairing.ResolvedTarget{
		Computer:         pairing.Computer{ID: "c-1", Kind: harness.KindT3Code, ServerURL: "http://t3.local", HarnessVersion: "0.0.34"},
		HarnessProjectID: "proj-1",
	}
}

func updatesChan(updates ...harness.Update) <-chan harness.Update {
	ch := make(chan harness.Update, len(updates))
	for _, u := range updates {
		ch <- u
	}
	close(ch)
	return ch
}

// --- HandleMessageCreated ----------------------------------------------------

func messageCreatedEvent(t *testing.T, conversationID, authorID, body string, mentionsAgentFlag bool) eventbus.Event {
	return messageCreatedEventKind(t, conversationID, authorID, "user", body, mentionsAgentFlag)
}

func messageCreatedEventKind(t *testing.T, conversationID, authorID, authorKind, body string, mentionsAgentFlag bool) eventbus.Event {
	t.Helper()
	mentions := []map[string]string{}
	if mentionsAgentFlag {
		mentions = append(mentions, map[string]string{"kind": "agent"})
	}
	payload, err := json.Marshal(map[string]any{"message": map[string]any{
		"conversation_id": conversationID,
		"author_id":       authorID,
		"author_kind":     authorKind,
		"body":            body,
		"mentions":        mentions,
	}})
	require.NoError(t, err)
	return eventbus.Event{Topic: "chat.message.created", Payload: payload}
}

func TestHandleMessageCreated_IgnoresNonUserAuthors(t *testing.T) {
	// Regression: the pipeline's own system replies contain "@Agent" and get
	// mentions parsed; without the author-kind guard they re-trigger turns
	// forever (found by the first live smoke test).
	for _, kind := range []string{"system", "agent", ""} {
		conv := newFakeConversations(Conversation{ID: "conv-1"})
		client := &fakeHarness{}
		svc := NewService(Config{
			Conversations: conv,
			Targets:       &fakeTargets{target: testTarget()},
			Harnesses:     registryOf(client),
			Live:          &fakeLive{},
		})
		ev := messageCreatedEventKind(t, "conv-1", "user-1", kind, "@Agent needs a paired T3 Code computer", true)
		require.NoError(t, svc.HandleMessageCreated(context.Background(), ev))
		time.Sleep(20 * time.Millisecond)
		replies, systemPosts := conv.snapshot()
		assert.Empty(t, replies, "author_kind %q must not trigger a turn", kind)
		assert.Empty(t, systemPosts, "author_kind %q must not trigger a turn", kind)
		_ = client
	}
}

func TestHandleMessageCreated_IgnoresNonMentionMessages(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "just chatting", false)))
	time.Sleep(20 * time.Millisecond)
	replies, systemPosts := conv.snapshot()
	assert.Empty(t, replies)
	assert.Empty(t, systemPosts)
}

func TestRunTurn_EmptyFinalizeFrameDoesNotWipeTheReply(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "the actual reply", Streaming: true}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent hi", true)))
	waitFor(t, time.Second, func() bool { replies, _ := conv.snapshot(); return len(replies) == 1 })
	replies, systemPosts := conv.snapshot()
	assert.Equal(t, "the actual reply", replies[0].body)
	assert.Empty(t, systemPosts)
}

func TestRunTurn_MessageFinishedMidTurn_KeepsStreaming(t *testing.T) {
	live := &fakeLive{}
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Reading the handler.", Streaming: false}},
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Read", Summary: "Read main.go"}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-2", Text: "", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: live})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { r, _ := conv.snapshot(); return len(r) == 1 })

	frames := live.snapshot()
	require.Len(t, frames, 4)
	for _, f := range frames {
		assert.True(t, f.Streaming, "a finished message or an empty close marker mid-turn must not end the bubble: %+v", f)
	}
	assert.Equal(t, "Read main.go", frames[2].Activity)
}

func TestRunTurn_ReplyPersistFails_ClearsTheBubble(t *testing.T) {
	live := &fakeLive{}
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "done!", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	conv.postReplyErr = errors.New("disk full")
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: live})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { f := live.snapshot(); return len(f) > 0 && !f[len(f)-1].Streaming })

	frames := live.snapshot()
	assert.Equal(t, StreamFrame{ConversationID: "conv-1", Streaming: false}, frames[len(frames)-1])
}

// --- resolution failure -> system reply -------------------------------------

func TestRunTurn_ResolveTargetNotConfigured_PostsSystemReply(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&pairing.NotConfiguredError{Reason: pairing.ReasonUnpaired}, "add one in Settings → Computers"},
		{&pairing.NotConfiguredError{Reason: pairing.ReasonExpiredToken}, "re-pair it"},
		{&pairing.NotConfiguredError{Reason: pairing.ReasonNoDefault}, "link this project in Settings → Computers → Projects"},
		{&pairing.NotConfiguredError{Reason: pairing.ReasonNoDefaultComputer}, "pick a default one"},
		{
			&pairing.NotConfiguredError{Reason: pairing.ReasonSetupRequired, Provider: "Codex", Computer: "Onik's laptop"},
			"@Agent can't use Codex on Onik's laptop until its setup is done — run setup for Onik's laptop in Settings → Computers.",
		},
		{
			&pairing.NotConfiguredError{Reason: pairing.ReasonOffline, Computer: "Onik's laptop"},
			"Onik's laptop is offline: T3 Code isn't answering there.",
		},
		{
			harness.ProtocolRefusal("T3 Code on Onik's laptop went back to its old orchestrator; Nexul only moves forward. Update T3 Code there."),
			"T3 Code on Onik's laptop went back to its old orchestrator; Nexul only moves forward. Update T3 Code there.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			conv := newFakeConversations(Conversation{ID: "conv-1"})
			client := &fakeHarness{}
			svc := NewService(Config{
				Conversations: conv,
				Targets:       &fakeTargets{err: tc.err},
				Harnesses:     registryOf(client),
				Live:          &fakeLive{},
			})
			require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent help", true)))
			waitFor(t, time.Second, func() bool { _, systemPosts := conv.snapshot(); return len(systemPosts) == 1 })
			_, systemPosts := conv.snapshot()
			assert.Contains(t, systemPosts[0].body, tc.want)
			assert.Equal(t, "u-1", systemPosts[0].viaUserID)
		})
	}
}

// --- happy path: streamed frames + final persisted message ------------------

func TestRunTurn_HappyPath_StreamsFramesAndPersistsFinalReply(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "thread-old"})
	live := &fakeLive{}
	client := &fakeHarness{startResult: harness.StartResult{
		SessionID: "thread-new",
		Updates: updatesChan(
			harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Tool: "Read", Summary: "Read main.go started"}},
			harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "wor", Streaming: true}},
			harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolResult, Tool: "Bash", Summary: "Bash"}},
			harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "working on it", Streaming: true}},
			harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "done!", Streaming: false}},
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
		),
	}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          live,
	})

	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { r, _ := conv.snapshot(); return len(r) == 1 })

	frames := live.snapshot()
	require.Len(t, frames, 6)
	assert.Equal(t, StreamFrame{ConversationID: "conv-1", Streaming: true}, frames[0], "an immediate empty working frame precedes the first snapshot")
	assert.Equal(t, StreamFrame{ConversationID: "conv-1", Streaming: true, Activity: "Read main.go started", ActivityKind: harness.ActivityToolCall, ActivityTool: "Read"}, frames[1], "a tool step publishes before any text")
	assert.Equal(t, StreamFrame{ConversationID: "conv-1", MessageID: "m-1", Text: "wor", Streaming: true}, frames[2], "text clears the stale tool step")
	assert.Equal(t, StreamFrame{ConversationID: "conv-1", MessageID: "m-1", Text: "wor", Streaming: true, Activity: "Bash", ActivityKind: harness.ActivityToolResult, ActivityTool: "Bash"}, frames[3], "a tool step keeps the text so far")
	assert.Equal(t, StreamFrame{ConversationID: "conv-1", MessageID: "m-1", Text: "done!", Streaming: true}, frames[5], "the persisted reply, not the last snapshot, ends the bubble")

	replies, systemPosts := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, "done!", replies[0].body)
	assert.Equal(t, "u-1", replies[0].viaUserID)
	assert.Nil(t, replies[0].handoffs, "a turn that handed nothing off stores none")
	assert.Empty(t, systemPosts)

	assert.Equal(t, "thread-new", conv.threads["conv-1"])
	assert.Contains(t, client.lastPrompt, "New message from")
}

func TestRunTurn_MessageWrittenInTheHarness_IsRelayedAsTheUsersOwnAndRecordedNotStreamed(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "thread-1"})
	live := &fakeLive{}
	typed := harness.Activity{Kind: harness.ActivityUserMessage, CallID: "turn-item:message:m-9", Tool: "T3", Summary: "Use two threads",
		Detail: "Use two threads\nand a native worker", At: minuteOf(20)}
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "thread-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Which policy?", Streaming: false}},
		harness.Update{Activity: &typed},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-2", Text: "Building the worker.", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	obs := &fakeObserver{}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: live})

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go", Observer: obs})

	assert.Equal(t, []fakeRelay{{"conv-1", "u-1", "Use two threads\nand a native worker", "T3", "turn-item:message:m-9", minuteOf(20)}}, conv.relayed,
		"the full text lands as the user's own message, at the time they wrote it, keyed by the harness's own id")
	kinds := make([]harness.ActivityKind, 0, len(obs.activity))
	for _, a := range obs.activity {
		kinds = append(kinds, a.Kind)
	}
	assert.Equal(t, []harness.ActivityKind{harness.ActivityText, harness.ActivityUserMessage, harness.ActivityText}, kinds,
		"the trail keeps the message between what the Agent said before and after it")
	for _, f := range live.snapshot() {
		assert.NotEqual(t, harness.ActivityUserMessage, f.ActivityKind, "the user's message is not the Agent's live step")
	}
	replies, _ := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, "Building the worker.", replies[0].body)
}

func TestRunTurn_FollowUp_LeavesOutWhatWasRelayedFromTheHarness(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1", "thread-2"}, updates: []<-chan harness.Update{replyThenDone("first answer"), replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)}}
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})
	conv.history = append(conv.history,
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "typed in T3", CreatedAt: minuteOf(34), Via: "T3"},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(35)},
	)
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})

	turns := h.started()
	require.Len(t, turns, 2)
	assert.NotContains(t, turns[1].prompts.Incremental, "typed in T3", "the harness session already holds what was written in it")
	assert.Contains(t, turns[1].prompts.Full, "u-1: typed in T3", "a replacement session never saw it")
}

// TestRunTurn_BotMessage_IsNamedAsItShowedAndKeptInTheFollowUp: a bot's message reads under the name it showed, marked
// a bot, never as the user its id is not; its via names the bot, not a harness, so a follow-up still carries it.
func TestRunTurn_BotMessage_IsNamedAsItShowedAndKeptInTheFollowUp(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1", "thread-2"}, updates: []<-chan harness.Update{replyThenDone("first answer"), replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)}}
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})
	conv.history = append(conv.history,
		ConversationMessage{AuthorID: "b-ci", AuthorKind: "bot", AuthorName: "GitHub Actions", Via: "CI", Body: "build failed", CreatedAt: minuteOf(34)},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(35)},
	)
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})

	turns := h.started()
	require.Len(t, turns, 2)
	assert.Contains(t, turns[1].prompts.Incremental, "GitHub Actions (bot): build failed")
	assert.NotContains(t, turns[1].prompts.Full, "b-ci")
}

// --- memories index wiring ---------------------------------------------

func TestRunTurn_PlainChat_NamesNoMemories(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", WorkspaceID: "workspace-default"})
	client := &fakeHarness{startResult: harness.StartResult{
		Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}),
	}}
	mem := &fakeMemories{items: []MemoryItem{{ID: "m-1", Name: "Coding style", AlwaysIncluded: true}}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Memories:      mem,
		Live:          &fakeLive{},
	})

	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { return client.snapshotPrompt() != "" })

	assert.False(t, mem.called, "a conversation with no ticket or doc has no project, so no memories (ADR 0099)")
	assert.NotContains(t, client.snapshotPrompt(), "Coding style")
	assert.NotContains(t, client.snapshotPrompt(), memoriesLine)
}

func TestRunTurn_MemoriesLookupFailure_IsBestEffort(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ProjectID: "proj-7"})
	client := &fakeHarness{startResult: harness.StartResult{
		Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}),
	}}
	mem := &fakeMemories{err: errors.New("docs unavailable")}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Memories:      mem,
		Live:          &fakeLive{},
	})

	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { return client.snapshotPrompt() != "" })

	mem.mu.Lock()
	defer mem.mu.Unlock()
	require.True(t, mem.called)
	assert.NotContains(t, client.snapshotPrompt(), memoriesLine, "a failed lookup names no memories rather than blocking the turn")
}

func TestRunTurn_TurnError_PostsSystemMessageWithLastError(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{
		SessionID: "thread-1",
		Updates: updatesChan(
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: "provider unreachable"}},
		),
	}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { _, n := conv.snapshot(); return len(n) == 1 })
	_, systemPosts := conv.snapshot()
	assert.Contains(t, systemPosts[0].body, "provider unreachable")
}

func TestRunTurn_StartTurnFails_PostsSystemMessage(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startErr: errors.New("connect refused")}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { _, n := conv.snapshot(); return len(n) == 1 })
	_, systemPosts := conv.snapshot()
	assert.Contains(t, systemPosts[0].body, "connect refused")
}

// --- approval decline ---------------------------------------------------------

func TestRunTurn_ApprovalUpdate_PostsSystemMessage(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{
		SessionID: "thread-1",
		Updates: updatesChan(
			harness.Update{Approval: &harness.Approval{Kind: "shell", Summary: "rm -rf /"}},
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
		),
	}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { _, n := conv.snapshot(); return len(n) == 1 })
	_, systemPosts := conv.snapshot()
	assert.Contains(t, systemPosts[0].body, "auto-declined")
	assert.Contains(t, systemPosts[0].body, "shell")
}

// --- version warning -----------------------------------------------------------

func TestRunTurn_VersionCheck_WarnsOnlyOnBaseReleaseChange(t *testing.T) {
	t.Run("nightly churn within the same base release does not warn", func(t *testing.T) {
		conv := newFakeConversations(Conversation{ID: "conv-1"})
		client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
		client.VersionFn = func(context.Context, string) (string, error) { return "0.0.34-nightly.9", nil }
		svc := NewService(Config{
			Conversations: conv,
			Targets:       &fakeTargets{target: testTarget()}, // recorded HarnessVersion: "0.0.34"
			Harnesses:     registryOf(client),
			Live:          &fakeLive{},
		})
		svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})
		_, systemPosts := conv.snapshot()
		assert.Empty(t, systemPosts)
	})

	t.Run("a base release change warns with a system message", func(t *testing.T) {
		conv := newFakeConversations(Conversation{ID: "conv-1"})
		client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
		client.VersionFn = func(context.Context, string) (string, error) { return "0.0.35", nil }
		svc := NewService(Config{
			Conversations: conv,
			Targets:       &fakeTargets{target: testTarget()}, // recorded HarnessVersion: "0.0.34"
			Harnesses:     registryOf(client),
			Live:          &fakeLive{},
		})
		require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
		waitFor(t, time.Second, func() bool { _, n := conv.snapshot(); return len(n) == 1 })
		_, systemPosts := conv.snapshot()
		require.Len(t, systemPosts, 1)
		assert.Contains(t, systemPosts[0].body, "0.0.34")
		assert.Contains(t, systemPosts[0].body, "0.0.35")
	})
}

// --- context cap ---------------------------------------------------------------

func TestRunTurn_ContextMessagesAreCapped(t *testing.T) {
	var history []ConversationMessage
	for i := 0; i < 2000; i++ {
		history = append(history, ConversationMessage{AuthorID: "u-2", AuthorKind: "user", Body: "line filler content that adds up over many messages"})
	}
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	conv.history = history
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})
	assert.LessOrEqual(t, len(client.lastPrompt), MaxPromptChars)
}

func TestRunTurn_InterviewThread_LoadsItsProjectsMemories(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", WorkspaceID: "workspace-default", ProjectID: "proj-7"})
	client := &fakeHarness{startResult: harness.StartResult{
		Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}),
	}}
	mem := &fakeMemories{}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Memories:      mem,
		Live:          &fakeLive{},
	})

	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { return client.snapshotPrompt() != "" })

	mem.mu.Lock()
	defer mem.mu.Unlock()
	assert.Equal(t, "proj-7", mem.calledProject)
}

func TestRunTurn_SessionTitle_NamesWhatTheConversationIsAbout(t *testing.T) {
	tests := []struct {
		name string
		conv Conversation
		want string
	}{
		{"ticket thread", Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"}, "Login broken"},
		{"doc thread", Conversation{ID: "conv-1", IsDocThread: true, DocID: "doc-1"}, "Runbook"},
		{"interview thread", Conversation{ID: "conv-1", ProjectID: "proj-1", ProjectName: "Globex"}, "Interview: Globex"},
		{"channel", Conversation{ID: "conv-1", Name: "general"}, "#general"},
		{"anything else never shows its id", Conversation{ID: "conv-1"}, "Nexul chat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
			svc := NewService(Config{
				Conversations: newFakeConversations(tt.conv),
				Targets:       &fakeTargets{target: testTarget()},
				Harnesses:     registryOf(client),
				Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Login broken"}},
				Docs:          &fakeDocs{doc: Doc{ProjectID: "proj-1", Title: "Runbook"}},
				Live:          &fakeLive{},
			})

			svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

			client.mu.Lock()
			defer client.mu.Unlock()
			assert.Equal(t, tt.want, client.lastTitle)
		})
	}
}

// --- doc thread context ---------------------------------------------------------

func TestRunTurn_DocThread_PromptNamesTheDocWithoutItsBody(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsDocThread: true, DocID: "doc-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Docs:          &fakeDocs{doc: Doc{ProjectID: "proj-9", Title: "Runbook", BodyMarkdown: "Restart the service like so."}},
		Live:          &fakeLive{},
	})
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	assert.Contains(t, client.lastPrompt, `Doc: "Runbook" (id doc-1). Read it with doc_get before you start.`)
	assert.NotContains(t, client.lastPrompt, "Restart the service like so.")
	assert.NotContains(t, client.lastPrompt, imagesLine, "no image was attached")
}

func TestRunTurn_DocThread_NoDocReader_RunsWithoutDocContext(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsDocThread: true, DocID: "doc-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	assert.NotContains(t, client.lastPrompt, "Doc:")
}

// --- RunTurn as a use-case ---------------------------------------------------

func TestRunTurn_TargetOverride_ResolvedThroughTheOverrideSeam(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	var gotTarget harness.Target
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, target harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		gotTarget = target
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	targets := &fakeTargets{target: &pairing.ResolvedTarget{
		Computer:         pairing.Computer{ID: "c-2", Kind: harness.KindT3Code, ServerURL: "http://t3.local"},
		HarnessProjectID: "proj-1", Provider: "claude", Model: "sonnet-5", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}},
		Worktree: true,
	}}
	svc := NewService(Config{Conversations: conv, Targets: targets, Harnesses: harnesstest.Registry(client), Live: &fakeLive{}})

	override := TargetOverride{ComputerID: "c-2", Provider: "claude", Model: "sonnet-5", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}}}
	svc.RunTurn(context.Background(), TurnRequest{
		ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "run it",
		Target: &override,
	})

	require.NotNil(t, targets.override)
	assert.Equal(t, override, *targets.override)
	assert.Equal(t, "proj-1", gotTarget.ProjectID)
	assert.Equal(t, "claude", gotTarget.Provider)
	assert.Equal(t, "sonnet-5", gotTarget.Model)
	assert.Equal(t, []harness.OptionSetting{{ID: "effort", Value: "high"}}, gotTarget.ModelOptions, "the resolved options reach the harness turn")
	assert.True(t, gotTarget.Worktree, "the person's start_in reaches the harness turn")
}

func TestRunTurn_TicketThread_PDFReference_AttachesNothing(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "tix-1"})
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	reader := newFakeAttachmentReader()
	reader.items["att-1"] = StoredAttachment{Name: "spec.pdf", MIME: "application/pdf", Bytes: []byte{1, 2, 3}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(client),
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Bug", Body: "see [spec](/api/attachments/att-1)"}},
		Attachments:   reader,
		Live:          &fakeLive{},
	})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent look"})

	assert.Empty(t, got.Attachments)
	assert.NotContains(t, got.Full, imagesLine, "the images line promises only what was attached")
}

// fourImageTypes is a harness client that sends its agent only gif, jpeg, png and webp images, as T3 Code on protocol 2 does.
type fourImageTypes struct{ *harnesstest.Client }

func (fourImageTypes) TakesImage(mime string) bool {
	return mime == "image/gif" || mime == "image/jpeg" || mime == "image/png" || mime == "image/webp"
}

func TestRunTurn_TicketImagesTheHarnessDoesNotTake_AreLinkedNotNamedAsAttached(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "tix-1"})
	var got harness.TurnPrompts
	client := fourImageTypes{&harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}}
	reader := newFakeAttachmentReader()
	reader.items["png-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte{1}}
	reader.items["svg-1"] = StoredAttachment{Name: "diagram.svg", MIME: "image/svg+xml", Bytes: []byte{2}}
	reader.items["bmp-1"] = StoredAttachment{Name: "scan.bmp", MIME: "image/bmp", Bytes: []byte{3}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harness.Registry{harness.KindT3Code: client},
		Tickets: &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Key: "SRC-3", Title: "Bug",
			Body: "![a](/api/attachments/svg-1) ![b](/api/attachments/png-1) ![c](/api/attachments/bmp-1)"}},
		Attachments: reader,
		Live:        &fakeLive{},
	})

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent look"})

	assert.Equal(t, []harness.Attachment{{Name: "shot.png", MIME: "image/png", Bytes: []byte{1}}}, got.Attachments)
	assert.Contains(t, got.Full, "Ticket: SRC-3 \"Bug\". Read it with ticket_get before you start.\n"+
		"Open these images in its body with attachment_get; they are not attached to this message:\n"+
		"- diagram.svg: /api/attachments/svg-1\n- scan.bmp: /api/attachments/bmp-1\n"+
		"The other images in its body are attached to this message, in order.\n\n")
	assert.NotContains(t, got.Full, imagesLine, "the prompt never claims an image the agent did not get")
}

// --- interrupt -------------------------------------------------------------

func TestInterrupt_NoActiveTurn_ReturnsNotFound(t *testing.T) {
	svc := NewService(Config{
		Conversations: newFakeConversations(Conversation{}),
		Targets:       &fakeTargets{},
		Harnesses:     registryOf(&fakeHarness{}),
		Live:          &fakeLive{},
	})
	err := svc.Interrupt(context.Background(), "conv-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestInterrupt_CallsHarnessWithTheActiveTarget(t *testing.T) {
	release := make(chan harness.Update)
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{
		startResult: harness.StartResult{SessionID: "thread-1", Updates: release},
		interruptCh: make(chan struct{}),
	}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))

	// Wait until the turn is registered as active (the client has been asked
	// to start, but the update stream is still open).
	waitFor(t, time.Second, func() bool {
		err := svc.Interrupt(context.Background(), "missing-conversation")
		return errors.Is(err, apperrs.ErrNotFound) // sanity: service is up and routing correctly
	})
	waitFor(t, time.Second, func() bool {
		svc.mu.Lock()
		_, ok := svc.active["conv-1"]
		svc.mu.Unlock()
		return ok
	})

	require.NoError(t, svc.Interrupt(context.Background(), "conv-1"))
	<-client.interruptCh
	close(release)
}

func TestInterruptAndAnswer_TwoMentionsInOneConversation_ReachEveryLiveTurn(t *testing.T) {
	first, second := make(chan harness.Update), make(chan harness.Update)
	h := &followUpHarness{sessionIDs: []string{"thread-1", "thread-2"}, updates: []<-chan harness.Update{first, second}}
	var mu sync.Mutex
	var interrupted, answered []string
	client := h.client()
	client.InterruptFn = func(_ context.Context, target harness.Target) error {
		mu.Lock()
		defer mu.Unlock()
		interrupted = append(interrupted, target.SessionID)
		return nil
	}
	client.AnswerFn = func(_ context.Context, target harness.Target, _ string, _ harness.QuestionAnswer) error {
		mu.Lock()
		defer mu.Unlock()
		answered = append(answered, target.SessionID)
		if target.SessionID == "thread-2" {
			return fmt.Errorf("%w: runtime request req-1 was not found", apperrs.ErrInvalid)
		}
		return nil
	}
	svc := NewService(Config{Conversations: newFakeConversations(Conversation{ID: "conv-1"}), Targets: &fakeTargets{target: testTarget()},
		Harnesses: harnesstest.Registry(client), Live: &fakeLive{}})
	run := func(updates chan harness.Update, body string) <-chan struct{} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: body})
		}()
		// Taken once the turn drains, so it is registered under its own session by then.
		updates <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Summary: "Read main.go"}}
		return done
	}
	firstDone := run(first, "@Agent first")
	secondDone := run(second, "@Agent second")
	require.Len(t, h.started(), 2)

	require.NoError(t, svc.Interrupt(t.Context(), "conv-1"))
	assert.Equal(t, []string{"thread-2", "thread-1"}, interrupted, "Stop reaches every live turn, newest first")
	answer := harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
	require.NoError(t, svc.Answer(t.Context(), "conv-1", "req-1", answer))
	assert.Equal(t, []string{"thread-2", "thread-1"}, answered, "a turn whose harness does not know the question passes it to the next")

	second <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	close(second)
	<-secondDone
	interrupted, answered = nil, nil
	require.NoError(t, svc.Interrupt(t.Context(), "conv-1"))
	require.NoError(t, svc.Answer(t.Context(), "conv-1", "req-1", answer))
	assert.Equal(t, []string{"thread-1"}, interrupted, "the second turn's end left the first one reachable")
	assert.Equal(t, []string{"thread-1"}, answered)

	close(first)
	<-firstDone
}

func TestRunTurn_HandedOffWorkLeftRunning_ReplyEndsWithTheLine(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "I handed the audit to another agent.", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone, LeftRunning: true}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent audit"})

	replies, systemPosts := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, "I handed the audit to another agent.\n\nPart of this work is still running in T3 Code.", replies[0].body)
	assert.Empty(t, systemPosts, "a turn that stopped waiting is done, not failed")
}

func TestRunTurn_Handoffs_EachFrameCarriesOneAndTheReplyKeepsTheFinalSetRedacted(t *testing.T) {
	token := "dep_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-_"
	at := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)
	audit := harness.Handoff{ID: "task-1", Driver: "claudeAgent", Title: "Audit", Prompt: "Audit the handlers.", State: harness.HandoffRunning,
		Steps: []harness.Activity{{Kind: harness.ActivityToolResult, CallID: "c-1", Tool: "Shell", Summary: "curl -H 'Authorization: Bearer " + token + "'", At: at}}}
	callers := harness.Handoff{ID: "native-8", Driver: "claudeAgent", Title: "Find callers", State: harness.HandoffRunning}
	audited := audit
	audited.State, audited.Reply = harness.HandoffDone, "Three issues."
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	live := &fakeLive{}
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
		harness.Update{Handoff: &audit},
		harness.Update{Handoff: &callers},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "Both helpers are done.", Streaming: false}},
		harness.Update{Handoff: &audited},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: live})
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent audit"})

	var pushed []string
	for _, f := range live.snapshot() {
		if f.Handoff != nil {
			pushed = append(pushed, f.Handoff.ID+" "+f.Handoff.State)
		}
	}
	assert.Equal(t, []string{"task-1 running", "native-8 running", "task-1 done"}, pushed, "one hand-off per frame, as it changed")
	redacted := "curl -H 'Authorization: Bearer " + redact.Placeholder + "'"
	assert.Equal(t, redacted, live.snapshot()[1].Handoff.Steps[0].Summary, "the live frame leaves redacted")

	replies, _ := conv.snapshot()
	require.Len(t, replies, 1)
	audited.Steps = []harness.Activity{{Kind: harness.ActivityToolResult, CallID: "c-1", Tool: "Shell", Summary: redacted, At: at}}
	assert.Equal(t, []harness.Handoff{audited, callers}, replies[0].handoffs, "the latest of each, in the order they started, stored redacted")
}

// --- follow-up turns on a live session -------------------------------------------

type startedTurn struct {
	sessionID string
	prompts   harness.TurnPrompts
}

// followUpHarness records every StartTurn and answers each with sessionIDs[i] (the last one repeats) and updates[i].
type followUpHarness struct {
	mu         sync.Mutex
	turns      []startedTurn
	sessionIDs []string
	updates    []<-chan harness.Update
	// answersInPlace has a turn carrying a pending answer send no prompt, as a harness resuming the asking run does.
	answersInPlace bool
}

func (h *followUpHarness) client() *harnesstest.Client {
	return &harnesstest.Client{StartTurnFn: func(_ context.Context, target harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		i := len(h.turns)
		h.turns = append(h.turns, startedTurn{sessionID: target.SessionID, prompts: prompts})
		return harness.StartResult{SessionID: h.sessionIDs[min(i, len(h.sessionIDs)-1)], Updates: h.updates[i],
			PromptSent: prompts.Answer == nil || !h.answersInPlace}, nil
	}}
}

func (h *followUpHarness) started() []startedTurn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]startedTurn{}, h.turns...)
}

func replyThenDone(text string) <-chan harness.Update {
	return updatesChan(
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-" + text, Text: text, Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)
}

func minuteOf(m int) time.Time { return time.Date(2026, 10, 2, 15, m, 0, 0, time.UTC) }

// ticketThreadWithStandingRules is a ticket thread in a project with an always-included memory, so a re-sent
// full prompt is visible in a follow-up; the Agent's replies land in history at minute 33.
func ticketThreadWithStandingRules(h *followUpHarness) (*Service, *fakeConversations) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"})
	conv.now = func() time.Time { return minuteOf(33) }
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(h.client()),
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Key: "NEX-1", Title: "Login broken", Body: "Steps to reproduce"}},
		Memories: &fakeProjectMemories{byProject: map[string][]MemoryItem{"proj-1": {
			{ID: "m-work", Name: "Working here", AlwaysIncluded: true},
		}}},
		Live: &fakeLive{},
		Now:  func() time.Time { return minuteOf(40) },
	})
	return svc, conv
}

func TestRunTurn_TicketThread_SecondMention_ReusesTheSessionWithOnlyWhatIsNew(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1"}, updates: []<-chan harness.Update{replyThenDone("first answer"), replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{
		{AuthorID: "u-2", AuthorKind: "user", Body: "context from before", CreatedAt: minuteOf(30)},
		{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)},
	}
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})

	conv.history = append(conv.history,
		ConversationMessage{AuthorID: "u-2", AuthorKind: "user", Body: "looks good", CreatedAt: minuteOf(34)},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "system", Body: "Agent turn failed: provider unreachable", CreatedAt: minuteOf(34)},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(35)},
	)
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})

	turns := h.started()
	require.Len(t, turns, 2)
	assert.Equal(t, "thread-1", turns[1].sessionID, "the second mention runs on the first one's session")
	assert.Equal(t, "New messages since your last turn:\n[2026-10-02 15:34] u-2: looks good\n\nNew message from u-1 at 2026-10-02 15:40:\n@Agent second", turns[1].prompts.Incremental,
		"only what was posted after the first turn, without the Agent's own reply, a system notice, or the request twice")
}

func TestRunTurn_MentionWhileTheLastTurnRuns_SendsOnlyMessagesAfterThatTurnsPrompt(t *testing.T) {
	running := make(chan harness.Update)
	h := &followUpHarness{sessionIDs: []string{"thread-1"}, updates: []<-chan harness.Update{running, replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)}}
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})
	}()
	// The first turn is draining once it takes an update, so its session and cursor are already recorded.
	running <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, Summary: "Read main.go"}}

	conv.mu.Lock()
	conv.history = append(conv.history,
		ConversationMessage{AuthorID: "u-2", AuthorKind: "user", Body: "while you work", CreatedAt: minuteOf(33)},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(34)},
	)
	conv.mu.Unlock()
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})
	close(running)
	<-firstDone

	turns := h.started()
	require.Len(t, turns, 2)
	assert.Equal(t, "thread-1", turns[1].sessionID)
	assert.True(t, strings.HasPrefix(turns[1].prompts.Incremental, "New messages since your last turn:\n[2026-10-02 15:33] u-2: while you work\n\nNew message from"),
		"the running turn's messages were already sent: %q", turns[1].prompts.Incremental)
}

func TestRunTurn_FollowUpOnALostSession_FullPromptRebuildsIt(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1", "thread-2"}, updates: []<-chan harness.Update{replyThenDone("first answer"), replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)}}
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})
	conv.history = append(conv.history, ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(35)})

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})

	turns := h.started()
	require.Len(t, turns, 2)
	full := turns[1].prompts.Full
	assert.Contains(t, full, "You are Agent")
	assert.Contains(t, full, "- Working here (id m-work)")
	assert.Contains(t, full, `Ticket: NEX-1 "Login broken". Read it with ticket_get before you start.`)
	assert.Contains(t, full, "Agent: first answer", "a replacement session never saw the Agent's own reply")
	assert.Equal(t, 1, strings.Count(full, "@Agent second"), "the request is not repeated in the history")
	conv.mu.Lock()
	defer conv.mu.Unlock()
	assert.Equal(t, "thread-2", conv.threads["conv-1"], "the next mention reuses the replacement session")
}

func TestRunTurn_FollowUp_CarriesANoteLeftSinceTheLastTurn(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1"}, updates: []<-chan harness.Update{replyThenDone("first answer"), replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)}}
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})

	conv.history = append(conv.history,
		ConversationMessage{AuthorID: "u-2", AuthorKind: "agent", Body: "Left a handoff", CreatedAt: minuteOf(34),
			Note: &NoteFile{Name: "handoff.md", Markdown: "# Handoff\nLogin retries twice."}},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(35)},
	)
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})

	turns := h.started()
	require.Len(t, turns, 2)
	note := "Agent: Left a handoff\n\nNote file handoff.md:\n# Handoff\nLogin retries twice."
	assert.Contains(t, turns[1].prompts.Incremental, note, "a note was left over MCP, so the live session never saw it")
	assert.Contains(t, turns[1].prompts.Full, note)
}

func TestRunTurn_AnswerTakenInPlace_KeepsTheMessagesSinceForTheNextPrompt(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1"}, answersInPlace: true,
		updates: []<-chan harness.Update{replyThenDone("which db?"), replyThenDone("used sqlite"), replyThenDone("second answer")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent first", CreatedAt: minuteOf(32)}}
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent first"})

	conv.history = append(conv.history,
		ConversationMessage{AuthorID: "u-2", AuthorKind: "user", Body: "while it asked", CreatedAt: minuteOf(34)},
		ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "Answered: sqlite", CreatedAt: minuteOf(35)},
	)
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "Answered: sqlite",
		Answer: &harness.PendingAnswer{RequestID: "rq-1"}})
	conv.history = append(conv.history, ConversationMessage{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent second", CreatedAt: minuteOf(36)})
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent second"})

	turns := h.started()
	require.Len(t, turns, 3)
	assert.Contains(t, turns[2].prompts.Incremental, "u-2: while it asked",
		"only the answer reached the harness, so the next prompt still carries what was posted meanwhile")
}

func TestRunTurn_PlayStart_CarriesNoHistoryButAdvancesTheCursor(t *testing.T) {
	h := &followUpHarness{sessionIDs: []string{"thread-1"}, updates: []<-chan harness.Update{replyThenDone("fixed")}}
	svc, conv := ticketThreadWithStandingRules(h)
	conv.history = []ConversationMessage{
		{AuthorID: "u-2", AuthorKind: "user", Body: "context from before", CreatedAt: minuteOf(30)},
		{AuthorID: "u-1", AuthorKind: "user", Body: "Started Fix with AI", CreatedAt: minuteOf(32)},
	}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", Play: &PlayContext{Label: "Fix with AI", Instructions: "Fix it."}})

	turns := h.started()
	require.Len(t, turns, 1)
	for name, prompt := range map[string]string{"full": turns[0].prompts.Full, "incremental": turns[0].prompts.Incremental} {
		assert.NotContains(t, prompt, "context from before", name)
		assert.NotContains(t, prompt, "Started Fix with AI", name)
		assert.NotContains(t, prompt, "New message from", name)
	}
	conv.mu.Lock()
	defer conv.mu.Unlock()
	assert.Equal(t, minuteOf(32), conv.syncedAt["conv-1"], "a later mention on the session does not resend what the run skipped")
}

// --- watch after a restart --------------------------------------------------------

func TestRunTurn_Watch_FollowsTheThreadsTurnWithoutStartingOne(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "thread-1"})
	client := &fakeHarness{watchResult: harness.StartResult{SessionID: "thread-1", Updates: updatesChan(
		harness.Update{Activity: &harness.Activity{Kind: harness.ActivityToolCall, CallID: "c-1", Tool: "Bash", Summary: "go test"}},
		harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "All green.", Streaming: false}},
		harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
	)}}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", Watch: true, Observer: obs})

	require.Len(t, client.watched, 1)
	assert.Equal(t, "thread-1", client.watched[0].SessionID)
	assert.Empty(t, client.snapshotPrompt(), "nothing is sent to the harness")
	assert.Equal(t, "thread-1", obs.sessionID)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnDone, obs.result.State)
	replies, _ := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, "All green.", replies[0].body)
}

func TestRunTurn_WatchWithoutThread_FailsWithoutCallingTheHarness(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	client := &fakeHarness{}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", Watch: true, Observer: obs})

	assert.Empty(t, client.watched)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnError, obs.result.State)
}

func TestRunTurn_WatchRefused_PostsReconnectFailure(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "thread-1"})
	client := &fakeHarness{watchErr: errors.New("computer offline")}
	svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: testTarget()}, Harnesses: registryOf(client), Live: &fakeLive{}})
	obs := &fakeObserver{}

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", Watch: true, Observer: obs})

	_, systemPosts := conv.snapshot()
	require.Len(t, systemPosts, 1)
	assert.Equal(t, "Agent turn failed to reconnect: computer offline", systemPosts[0].body)
	require.NotNil(t, obs.result)
	assert.Equal(t, harness.TurnError, obs.result.State)
}

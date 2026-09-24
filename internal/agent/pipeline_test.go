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
)

// --- fakes -----------------------------------------------------------------

type fakeNote struct {
	conversationID, viaUserID, body string
}

type fakeConversations struct {
	mu sync.Mutex

	conv    Conversation
	getErr  error
	history []ConversationMessage
	histErr error

	threads       map[string]string
	syncedAt      map[string]time.Time
	replies       []fakeNote
	notes         []fakeNote
	userPosts     []fakeNote
	postReplyErr  error
	postNoteErr   error
	setThreadErr  error
	markSyncedErr error
}

func newFakeConversations(conv Conversation) *fakeConversations {
	return &fakeConversations{conv: conv, threads: map[string]string{}, syncedAt: map[string]time.Time{}}
}

func (f *fakeConversations) GetConversation(_ context.Context, _ string) (Conversation, error) {
	if f.getErr != nil {
		return Conversation{}, f.getErr
	}
	return f.conv, nil
}

func (f *fakeConversations) MessagesSince(_ context.Context, _ string, _ time.Time) ([]ConversationMessage, error) {
	return f.history, f.histErr
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

func (f *fakeConversations) PostAgentReply(_ context.Context, conversationID, viaUserID, body string) (string, error) {
	if f.postReplyErr != nil {
		return "", f.postReplyErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies = append(f.replies, fakeNote{conversationID, viaUserID, body})
	return fmt.Sprintf("reply-%d", len(f.replies)), nil
}

func (f *fakeConversations) PostSystemNote(_ context.Context, conversationID, viaUserID, body string) error {
	if f.postNoteErr != nil {
		return f.postNoteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notes = append(f.notes, fakeNote{conversationID, viaUserID, body})
	return nil
}

func (f *fakeConversations) PostUserMessage(_ context.Context, conversationID, userID, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userPosts = append(f.userPosts, fakeNote{conversationID, userID, body})
	return nil
}

func (f *fakeConversations) snapshot() ([]fakeNote, []fakeNote) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeNote{}, f.replies...), append([]fakeNote{}, f.notes...)
}

type fakeTargets struct {
	target *pairing.ResolvedTarget
	err    error
	// override records the last ResolveTargetOverride call, so a test can assert an override reached the seam.
	override *TargetOverride
}

func (f *fakeTargets) ResolveTarget(_ context.Context, _, _ string) (*pairing.ResolvedTarget, error) {
	return f.target, f.err
}

func (f *fakeTargets) ResolveTargetOverride(_ context.Context, _, _, computerID, provider, model string) (*pairing.ResolvedTarget, error) {
	f.override = &TargetOverride{ComputerID: computerID, Provider: provider, Model: model}
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
	mu              sync.Mutex
	calledWorkspace string
	calledProject   string
	idx             MemoriesIndex
	err             error
}

func (f *fakeMemories) ListMemories(_ context.Context, workspaceID, projectID string) (MemoriesIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calledWorkspace = workspaceID
	f.calledProject = projectID
	return f.idx, f.err
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
}

func (f *fakeHarness) StartTurn(_ context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	f.mu.Lock()
	f.lastTarget = target
	f.lastTitle = title
	f.lastPrompt = prompts.Full
	f.lastIncremental = prompts.Incremental
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
		replies, notes := conv.snapshot()
		assert.Empty(t, replies, "author_kind %q must not trigger a turn", kind)
		assert.Empty(t, notes, "author_kind %q must not trigger a turn", kind)
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
	replies, notes := conv.snapshot()
	assert.Empty(t, replies)
	assert.Empty(t, notes)
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
	replies, notes := conv.snapshot()
	assert.Equal(t, "the actual reply", replies[0].body)
	assert.Empty(t, notes)
}

// --- resolution failure -> system reply -------------------------------------

func TestRunTurn_ResolveTargetNotConfigured_PostsSystemReply(t *testing.T) {
	cases := []struct {
		err  *pairing.NotConfiguredError
		want string
	}{
		{&pairing.NotConfiguredError{Reason: pairing.ReasonUnpaired}, "connect one in Settings"},
		{&pairing.NotConfiguredError{Reason: pairing.ReasonExpiredToken}, "re-pair it"},
		{&pairing.NotConfiguredError{Reason: pairing.ReasonNoDefault}, "link one in this project's settings"},
		{&pairing.NotConfiguredError{Reason: pairing.ReasonNoDefaultComputer}, "pick a default one"},
		{
			&pairing.NotConfiguredError{Reason: pairing.ReasonSetupRequired, Provider: "Codex", Computer: "Onik's laptop"},
			"@Agent can't use Codex on Onik's laptop until its setup is done — run setup for Onik's laptop in Settings → T3 pairing.",
		},
		{
			&pairing.NotConfiguredError{Reason: pairing.ReasonOffline, Computer: "Onik's laptop"},
			"@Agent can't reach Onik's laptop — is T3 Code running there?",
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.err.Reason), func(t *testing.T) {
			conv := newFakeConversations(Conversation{ID: "conv-1"})
			client := &fakeHarness{}
			svc := NewService(Config{
				Conversations: conv,
				Targets:       &fakeTargets{err: tc.err},
				Harnesses:     registryOf(client),
				Live:          &fakeLive{},
			})
			require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent help", true)))
			waitFor(t, time.Second, func() bool { _, notes := conv.snapshot(); return len(notes) == 1 })
			_, notes := conv.snapshot()
			assert.Contains(t, notes[0].body, tc.want)
			assert.Equal(t, "u-1", notes[0].viaUserID)
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
	assert.False(t, frames[5].Streaming)

	replies, notes := conv.snapshot()
	require.Len(t, replies, 1)
	assert.Equal(t, "done!", replies[0].body)
	assert.Equal(t, "u-1", replies[0].viaUserID)
	assert.Empty(t, notes)

	assert.Equal(t, "thread-new", conv.threads["conv-1"])
	assert.False(t, conv.syncedAt["conv-1"].IsZero())
	assert.Contains(t, client.lastPrompt, "New message from")
}

// --- memories index wiring ---------------------------------------------

func TestRunTurn_LoadsMemoriesIndexIntoPrompt(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", WorkspaceID: "workspace-default"})
	client := &fakeHarness{startResult: harness.StartResult{
		Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}),
	}}
	mem := &fakeMemories{idx: MemoriesIndex{Project: []MemoryItem{{Name: "Coding style", WhenToUse: "always"}}}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Memories:      mem,
		Live:          &fakeLive{},
	})

	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool { return client.snapshotPrompt() != "" })

	assert.Equal(t, "", mem.calledProject, "a non-ticket-thread conversation has no project")
	assert.Equal(t, "workspace-default", mem.calledWorkspace, "the conversation's workspace still resolves with no project (ADR 0059)")
	assert.Contains(t, client.snapshotPrompt(), "Coding style: always")
}

func TestRunTurn_MemoriesLookupFailure_IsBestEffort(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
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

	assert.Contains(t, client.snapshotPrompt(), "no memories saved yet", "a failed lookup falls back to an empty index rather than blocking the turn")
}

func TestRunTurn_TurnError_PostsSystemNoteWithLastError(t *testing.T) {
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
	_, notes := conv.snapshot()
	assert.Contains(t, notes[0].body, "provider unreachable")
}

func TestRunTurn_StartTurnFails_PostsSystemNote(t *testing.T) {
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
	_, notes := conv.snapshot()
	assert.Contains(t, notes[0].body, "connect refused")
}

// --- approval decline ---------------------------------------------------------

func TestRunTurn_ApprovalUpdate_PostsSystemNote(t *testing.T) {
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
	_, notes := conv.snapshot()
	assert.Contains(t, notes[0].body, "auto-declined")
	assert.Contains(t, notes[0].body, "shell")
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
		require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
		waitFor(t, time.Second, func() bool {
			conv.mu.Lock()
			defer conv.mu.Unlock()
			return !conv.syncedAt["conv-1"].IsZero()
		})
		_, notes := conv.snapshot()
		assert.Empty(t, notes)
	})

	t.Run("a base release change warns with a system note", func(t *testing.T) {
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
		_, notes := conv.snapshot()
		require.Len(t, notes, 1)
		assert.Contains(t, notes[0].body, "0.0.34")
		assert.Contains(t, notes[0].body, "0.0.35")
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
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool {
		conv.mu.Lock()
		defer conv.mu.Unlock()
		return !conv.syncedAt["conv-1"].IsZero()
	})
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
		{"interview thread", Conversation{ID: "conv-1", ProjectID: "proj-1", ProjectName: "Shopkeepers"}, "Interview: Shopkeepers"},
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

func TestRunTurn_DocThread_PromptIncludesDocTitleAndBody(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsDocThread: true, DocID: "doc-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Docs:          &fakeDocs{doc: Doc{ProjectID: "proj-9", Title: "Runbook", BodyMarkdown: "Restart the service like so."}},
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool {
		conv.mu.Lock()
		defer conv.mu.Unlock()
		return !conv.syncedAt["conv-1"].IsZero()
	})

	assert.Contains(t, client.lastPrompt, "Doc: Runbook")
	assert.Contains(t, client.lastPrompt, "Restart the service like so.")
}

func TestRunTurn_DocThread_OversizedBodyIsTrimmedWithNote(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsDocThread: true, DocID: "doc-1"})
	client := &fakeHarness{startResult: harness.StartResult{SessionID: "t-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     registryOf(client),
		Docs:          &fakeDocs{doc: Doc{ProjectID: "proj-9", Title: "Huge doc", BodyMarkdown: strings.Repeat("x", MaxPromptChars*2)}},
		Live:          &fakeLive{},
	})
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool {
		conv.mu.Lock()
		defer conv.mu.Unlock()
		return !conv.syncedAt["conv-1"].IsZero()
	})

	assert.LessOrEqual(t, len(client.lastPrompt), MaxPromptChars)
	assert.Contains(t, client.lastPrompt, docTrimNote)
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
	require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-1", "@Agent go", true)))
	waitFor(t, time.Second, func() bool {
		conv.mu.Lock()
		defer conv.mu.Unlock()
		return !conv.syncedAt["conv-1"].IsZero()
	})

	assert.NotContains(t, client.lastPrompt, "Doc:")
}

// --- RunTurn as a use-case ---------------------------------------------------

func TestRunTurn_ExtraRequestBlocksLandInBothPrompts(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "thread-reused"})
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "thread-reused", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(client),
		Live:          &fakeLive{},
	})

	svc.RunTurn(context.Background(), TurnRequest{
		ConversationID:     "conv-1",
		ViaUserID:          "u-1",
		RequestBody:        "Started Fix with AI",
		ExtraRequestBlocks: []string{"Play: Fix with AI\nDo the fix.", "Instructions from u-1 for this run:\nkeep it small"},
	})

	wantTail := "Started Fix with AI\n\nPlay: Fix with AI\nDo the fix.\n\nInstructions from u-1 for this run:\nkeep it small"
	assert.True(t, strings.HasSuffix(got.Full, wantTail), "full prompt tail:\n%s", got.Full)
	assert.True(t, strings.HasSuffix(got.Incremental, wantTail), "incremental prompt tail:\n%s", got.Incremental)
	assert.Contains(t, got.Incremental, "New message from u-1", "a reused session still gets the request block")
	assert.NotContains(t, got.Incremental, "You are Agent")
	_, notes := conv.snapshot()
	assert.Empty(t, notes)
}

func TestRunTurn_TargetOverride_ResolvedThroughTheOverrideSeam(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	var gotTarget harness.Target
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, target harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		gotTarget = target
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	targets := &fakeTargets{target: &pairing.ResolvedTarget{
		Computer:         pairing.Computer{ID: "c-2", Kind: harness.KindT3Code, ServerURL: "http://t3.local"},
		HarnessProjectID: "proj-1", Provider: "claude", Model: "sonnet-5",
	}}
	svc := NewService(Config{Conversations: conv, Targets: targets, Harnesses: harnesstest.Registry(client), Live: &fakeLive{}})

	svc.RunTurn(context.Background(), TurnRequest{
		ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "run it",
		Target: &TargetOverride{ComputerID: "c-2", Provider: "claude", Model: "sonnet-5"},
	})

	require.NotNil(t, targets.override)
	assert.Equal(t, TargetOverride{ComputerID: "c-2", Provider: "claude", Model: "sonnet-5"}, *targets.override)
	assert.Equal(t, "claude", gotTarget.Provider)
	assert.Equal(t, "sonnet-5", gotTarget.Model)
}

func TestRunTurn_AttachmentsReachTheHarness(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"})
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(client),
		Live:          &fakeLive{},
	})
	attachments := []harness.Attachment{{Name: "shot.png", MIME: "image/png", Bytes: []byte{1, 2, 3}}}

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "look", Attachments: attachments})

	assert.Equal(t, attachments, got.Attachments)
}

func TestRunTurn_TicketThread_EmbeddedImage_AttachesAndRewritesPrompt(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "tix-1"})
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	reader := newFakeAttachmentReader()
	reader.items["att-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte{1, 2, 3}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(client),
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Bug", Body: "before ![shot](/api/attachments/att-1) after"}},
		Attachments:   reader,
		Live:          &fakeLive{},
	})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent look"})

	require.Len(t, got.Attachments, 1)
	assert.Equal(t, "shot.png", got.Attachments[0].Name)
	assert.Equal(t, "image/png", got.Attachments[0].MIME)
	assert.Contains(t, got.Full, "before [image: shot.png, attached to this turn] after")
}

func TestRunTurn_TicketThread_PDFReference_OmittedWithNoAttachment(t *testing.T) {
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
	assert.Contains(t, got.Full, "see [attachment omitted: spec.pdf]")
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

package agent

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
)

// fakeProjectMemories is a MemoriesReader keyed by workspace and project id, so a test can prove a call for
// the wrong (or empty) id gets nothing back for that slice — real memories.Service.ListMemoryItems behaves
// the same way (ADR 0059: workspace-scoped memories reach every turn, project ones only their own thread).
type fakeProjectMemories struct {
	byWorkspace map[string][]MemoryItem
	byProject   map[string][]MemoryItem
}

func (f *fakeProjectMemories) ListMemories(_ context.Context, workspaceID, projectID string) (MemoriesIndex, error) {
	return MemoriesIndex{Workspace: f.byWorkspace[workspaceID], Project: f.byProject[projectID]}, nil
}

// captureFullAndIncremental wires a Service whose harness client records both prompts StartTurn was called with.
func captureFullAndIncremental(t *testing.T, cfg Config) (svc *Service, got *harness.TurnPrompts) {
	t.Helper()
	got = &harness.TurnPrompts{}
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		*got = prompts
		return harness.StartResult{Updates: updatesChan(
			harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "done", Streaming: false}},
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
		)}, nil
	}}
	cfg.Harnesses = harnesstest.Registry(client)
	if cfg.Targets == nil {
		cfg.Targets = &fakeTargets{target: testTarget()}
	}
	if cfg.Live == nil {
		cfg.Live = &fakeLive{}
	}
	return NewService(cfg), got
}

func TestRunTurn_TicketThread_AlwaysIncludedMemoryInlinedInFullInBothPrompts(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1", ThreadID: "thread-reused"})
	mem := &fakeProjectMemories{byProject: map[string][]MemoryItem{
		"proj-1": {
			{Name: "Working in this project", WhenToUse: "always", AlwaysIncluded: true, Body: "Standing rule body text."},
		},
	}}
	svc, got := captureFullAndIncremental(t, Config{
		Conversations: conv,
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Ticket title", Body: "Ticket body"}},
		Memories:      mem,
	})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	for _, prompt := range []string{got.Full, got.Incremental} {
		assert.Contains(t, prompt, "Always-included memories, follow them:")
		assert.Contains(t, prompt, "### Working in this project\nStanding rule body text.")
	}
}

func TestRunTurn_ChannelWithNoProject_InlinesWorkspaceAlwaysIncludedMemories(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", WorkspaceID: "ws-1"}) // no ticket, no doc: no project
	mem := &fakeProjectMemories{byWorkspace: map[string][]MemoryItem{
		"ws-1": {
			{Name: "Team tone", WhenToUse: "always", AlwaysIncluded: true, Body: "Standing rule body text."},
		},
	}}
	svc, got := captureFullAndIncremental(t, Config{Conversations: conv, Memories: mem})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	for _, prompt := range []string{got.Full, got.Incremental} {
		assert.Contains(t, prompt, "Always-included memories, follow them:")
		assert.Contains(t, prompt, "### Team tone\nStanding rule body text.")
	}
}

func TestRunTurn_ChannelWithNoProjectOrWorkspace_InlinesNoAlwaysIncludedMemories(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1"}) // no ticket, no doc, no workspace
	mem := &fakeProjectMemories{byWorkspace: map[string][]MemoryItem{
		"ws-1": {
			{Name: "Team tone", WhenToUse: "always", AlwaysIncluded: true, Body: "Standing rule body text."},
		},
	}}
	svc, got := captureFullAndIncremental(t, Config{Conversations: conv, Memories: mem})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	assert.NotContains(t, got.Full, "Always-included memories, follow them:")
	assert.NotContains(t, got.Full, "Standing rule body text")
	assert.NotContains(t, got.Incremental, "Standing rule body text")
}

func TestRunTurn_MemoriesIndex_ExcludesAlwaysIncludedMemories(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"})
	mem := &fakeProjectMemories{byProject: map[string][]MemoryItem{
		"proj-1": {
			{Name: "Working in this project", WhenToUse: "always rules", AlwaysIncluded: true, Body: "Standing rule body text."},
			{Name: "Deploy quirks", WhenToUse: "use this if touching deploy config"},
		},
	}}
	svc, got := captureFullAndIncremental(t, Config{
		Conversations: conv,
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1"}},
		Memories:      mem,
	})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	assert.Contains(t, got.Full, "- Deploy quirks: use this if touching deploy config")
	assert.NotContains(t, got.Full, "- Working in this project:", "an always-included memory is inlined, never listed in the index")
}

func TestRunTurn_OverCeilingAlwaysIncludedMemories_TrimmedWithNoteLoggedTurnStillRuns(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	// Four memories at 19,000 chars each: each is under the 20,000 per-memory ceiling on its own, but the
	// four together (76,000) are well over the 60,000 per-run ceiling, forcing a drop from the end.
	items := make([]MemoryItem, 4)
	for i := range items {
		items[i] = MemoryItem{Name: fmt.Sprintf("Memory %d", i), WhenToUse: "always", AlwaysIncluded: true, Body: strings.Repeat("x", 19_000)}
	}
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"})
	mem := &fakeProjectMemories{byProject: map[string][]MemoryItem{"proj-1": items}}
	svc, got := captureFullAndIncremental(t, Config{
		Conversations: conv,
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1"}},
		Memories:      mem,
		Logger:        log,
	})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	assert.Contains(t, got.Full, "[always-included memories trimmed to fit: 3 of 4 inlined]")
	assert.Contains(t, logBuf.String(), "trimmed to fit ceiling")
	assert.Contains(t, logBuf.String(), "level=WARN")

	// RunTurn blocks until the turn ends; by the time it returns the trim must not have kept the turn from
	// completing normally.
	replies, notes := conv.snapshot()
	assert.Empty(t, notes, "the turn must complete normally despite the trim, not fail")
	assert.NotEmpty(t, replies)
}

func TestRunTurn_AlwaysIncludedMemoryImage_TravelsAsAttachment(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"})
	mem := &fakeProjectMemories{byProject: map[string][]MemoryItem{
		"proj-1": {
			{Name: "Working in this project", AlwaysIncluded: true, Body: "Layout: ![diagram](/api/attachments/att-9) end"},
		},
	}}
	reader := newFakeAttachmentReader()
	reader.items["att-9"] = StoredAttachment{Name: "diagram.png", MIME: "image/png", Bytes: []byte{9}}
	var got harness.TurnPrompts
	client := &harnesstest.Client{StartTurnFn: func(_ context.Context, _ harness.Target, _ string, prompts harness.TurnPrompts) (harness.StartResult, error) {
		got = prompts
		return harness.StartResult{SessionID: "thread-1", Updates: updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}})}, nil
	}}
	svc := NewService(Config{
		Conversations: conv,
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(client),
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Ticket", Body: "plain"}},
		Memories:      mem,
		Attachments:   reader,
		Live:          &fakeLive{},
	})

	svc.RunTurn(context.Background(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

	require.Len(t, got.Attachments, 1)
	assert.Equal(t, "diagram.png", got.Attachments[0].Name)
	assert.Contains(t, got.Full, "Layout: [image: diagram.png, attached to this turn] end")
	assert.Contains(t, got.Incremental, "Layout: [image: diagram.png, attached to this turn] end")
}

func TestSplitAlwaysIncluded_InterviewLeads_SoATrimNeverDropsIt(t *testing.T) {
	_, always := splitAlwaysIncluded(MemoriesIndex{
		Workspace: []MemoryItem{{Name: "Team tone", AlwaysIncluded: true, Body: strings.Repeat("w", MaxMemoryChars)}},
		Project: []MemoryItem{
			{Name: "Working here", AlwaysIncluded: true, Body: "p"},
			{Name: "Interview", AlwaysIncluded: true, Interview: true, Body: "Go only."},
		},
	})
	require.Len(t, always, 3)
	assert.Equal(t, "Interview", always[0].Title)

	block := InlineMemoriesTrimmed(always, InlineLimits{PerMemory: MaxMemoryChars, PerRun: MaxMemoryChars}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	assert.Contains(t, block, "### Interview\nGo only.")
}

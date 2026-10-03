package agent

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
)

// fakeProjectMemories is a MemoriesReader keyed by project id, so a test can prove a call for the wrong (or
// empty) id gets nothing back, as real memories.Service.ListMemoryItems does.
type fakeProjectMemories struct {
	byProject map[string][]MemoryItem
}

func (f *fakeProjectMemories) ListMemories(_ context.Context, projectID string) ([]MemoryItem, error) {
	return f.byProject[projectID], nil
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

func TestRunTurn_TicketThreadMention_NamesItsContextInsteadOfCarryingIt(t *testing.T) {
	conv := newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"})
	conv.history = []ConversationMessage{
		{AuthorID: "u-2", AuthorKind: "user", Body: "it 500s on submit", CreatedAt: time.Date(2026, 10, 3, 12, 18, 0, 0, time.UTC)},
		{AuthorID: "u-1", AuthorKind: "system", Body: "This computer's agent harness changed release", CreatedAt: time.Date(2026, 10, 3, 12, 19, 0, 0, time.UTC)},
		{AuthorID: "u-1", AuthorKind: "user", Body: "@Agent why?", CreatedAt: time.Date(2026, 10, 3, 12, 20, 0, 0, time.UTC)},
	}
	reader := newFakeAttachmentReader()
	reader.items["att-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte{1, 2, 3}}
	svc, got := captureFullAndIncremental(t, Config{
		Conversations: conv,
		Tickets:       &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Key: "SRC-3", Title: "Login fails", Body: "Steps ![shot](/api/attachments/att-1) spec body"}},
		Memories: &fakeProjectMemories{byProject: map[string][]MemoryItem{"proj-1": {
			{ID: "m-work", Name: "Working here", AlwaysIncluded: true},
			{ID: "m-deploy", Name: "Deploy quirks"},
			{ID: "m-int", Name: "Interview", AlwaysIncluded: true, Interview: true},
		}}},
		Attachments: reader,
		Now:         func() time.Time { return time.Date(2026, 10, 3, 12, 21, 0, 0, time.UTC) },
	})

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent why?"})

	assert.Equal(t, DefaultIntro+"\n\n"+
		"Right now you are running @Agent in a chat, mentioned by u-1.\n\n"+
		"Ticket: SRC-3 \"Login fails\". Read it with ticket_get before you start.\n"+
		"The images in its body are attached to this message, in order.\n\n"+
		"Read these memories with memory_get before you start; they are your context:\n"+
		"- Interview (id m-int)\n- Working here (id m-work)\n\n"+
		"Conversation so far:\n[2026-10-03 12:18] u-2: it 500s on submit\n\n"+
		"New message from u-1 at 2026-10-03 12:21:\n@Agent why?\n\n"+
		DefaultFooter, got.Full)
	require.Len(t, got.Attachments, 1)
	assert.Equal(t, "shot.png", got.Attachments[0].Name)
}

// promptTemplates is the Templates seam: a body per key, or one error for every read.
type promptTemplates struct {
	bodies map[string]string
	err    error
}

func (p promptTemplates) Effective(_ context.Context, kind, key string) (string, error) {
	if kind != TemplateKind {
		return "", errors.New("wrong kind " + kind)
	}
	return p.bodies[key], p.err
}

func TestRunTurn_PromptTemplates_FrameTheFullPrompt(t *testing.T) {
	tests := []struct {
		name       string
		templates  promptTemplates
		wantPrefix string
		wantSuffix string
		wantWarn   bool
	}{
		{"edited intro, emptied footer", promptTemplates{bodies: map[string]string{TemplateIntro: "Be terse.", TemplateFooter: ""}}, "Be terse.\n\nRight now", "@Agent go", false},
		{"a failed read falls back to the code defaults", promptTemplates{err: errors.New("db down")}, DefaultIntro + "\n\n", DefaultFooter, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			svc, got := captureFullAndIncremental(t, Config{
				Conversations: newFakeConversations(Conversation{ID: "conv-1"}),
				Templates:     tt.templates,
				Logger:        slog.New(slog.NewTextHandler(&logs, nil)),
			})

			svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})

			assert.True(t, strings.HasPrefix(got.Full, tt.wantPrefix), "full prompt:\n%s", got.Full)
			assert.True(t, strings.HasSuffix(got.Full, tt.wantSuffix), "full prompt:\n%s", got.Full)
			assert.NotContains(t, got.Incremental, "Be terse.", "a live session already holds the intro")
			assert.Equal(t, tt.wantWarn, bytes.Contains(logs.Bytes(), []byte("read prompt template failed")))
		})
	}
}

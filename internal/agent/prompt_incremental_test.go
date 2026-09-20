package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComposeIncrementalPrompt(t *testing.T) {
	at := time.Date(2026, 8, 26, 15, 4, 0, 0, time.UTC)
	in := PromptInput{
		Ticket:          &TicketContext{Title: "T", Body: "body"},
		Memories:        MemoriesIndex{Project: []MemoryItem{{Name: "conventions", WhenToUse: "always"}}},
		ContextMessages: []ContextMessage{{Author: "lena", Body: "ship it", At: at}},
		RequestAuthor:   "Onik97",
		RequestBody:     "@Agent do the thing",
		RequestAt:       at.Add(time.Minute),
	}
	got := ComposeIncrementalPrompt(in)

	assert.NotContains(t, got, "You are Agent", "instructions live in the reused thread already")
	assert.NotContains(t, got, "memories", "memories index is not re-sent")
	assert.NotContains(t, got, "Ticket:", "ticket block is not re-sent")
	assert.Contains(t, got, "New messages since your last turn:")
	assert.Contains(t, got, "[2026-08-26 15:04] lena: ship it")
	assert.Contains(t, got, "New message from Onik97 at 2026-08-26 15:05:\n@Agent do the thing")
}

func TestComposeIncrementalPrompt_NoNewContextIsJustTheRequest(t *testing.T) {
	got := ComposeIncrementalPrompt(PromptInput{RequestAuthor: "Onik97", RequestBody: "@Agent hi"})
	assert.Equal(t, "New message from Onik97:\n@Agent hi", got)
}

func TestComposeIncrementalPrompt_ExtraRequestBlocksFollowTheBody(t *testing.T) {
	tests := []struct {
		name   string
		blocks []string
		want   string
	}{
		{"none", nil, "New message from Onik97:\n@Agent hi"},
		{"one", []string{"Play: Fix with AI"}, "New message from Onik97:\n@Agent hi\n\nPlay: Fix with AI"},
		{"two in order", []string{"Play: Fix with AI", "Instructions from Onik97:\nbe brief"}, "New message from Onik97:\n@Agent hi\n\nPlay: Fix with AI\n\nInstructions from Onik97:\nbe brief"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComposeIncrementalPrompt(PromptInput{RequestAuthor: "Onik97", RequestBody: "@Agent hi", ExtraRequestBlocks: tt.blocks})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComposePrompt_TimestampsContextLines(t *testing.T) {
	at := time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)
	got := ComposePrompt(PromptInput{
		ContextMessages: []ContextMessage{{Author: "sam", Body: "hello", At: at}},
		RequestAuthor:   "Onik97", RequestBody: "@Agent hey", RequestAt: at,
	})
	assert.Contains(t, got, "[2026-08-26 09:30] sam: hello")
	if !strings.Contains(got, "New message from Onik97 at 2026-08-26 09:30:") {
		t.Fatalf("request block missing timestamp: %q", got)
	}
}

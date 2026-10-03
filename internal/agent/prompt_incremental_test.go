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
		Intro: DefaultIntro, Footer: DefaultFooter,
		Target:          `Ticket: NEX-1 "T". Read it with ticket_get before you start.`,
		Memories:        []MemoryRef{{ID: "m-1", Name: "conventions"}},
		ContextMessages: []ContextMessage{{Author: "lena", Body: "ship it", At: at}},
		RequestAuthor:   "Onik97",
		RequestBody:     "@Agent do the thing",
		RequestAt:       at.Add(time.Minute),
	}
	got := ComposeIncrementalPrompt(in)

	assert.Equal(t, "New messages since your last turn:\n[2026-08-26 15:04] lena: ship it\n\n"+
		"New message from Onik97 at 2026-08-26 15:05:\n@Agent do the thing", got, "the session already holds the intro, the ticket, and the memories")
}

func TestComposeIncrementalPrompt_NoNewContextIsJustTheRequest(t *testing.T) {
	got := ComposeIncrementalPrompt(PromptInput{RequestAuthor: "Onik97", RequestBody: "@Agent hi"})
	assert.Equal(t, "New message from Onik97:\n@Agent hi", got)
}

func TestComposeIncrementalPrompt_Play(t *testing.T) {
	start := playAnswer()
	start.RequestBody = ""
	start.Play.Blocks = []string{"This ticket was blocked by these tickets, all now done:\n- NEX-4 \"schema\": done"}
	start.Play.Custom = "Touch only the docs."
	assert.Equal(t, "Right now you are running a play, started by onik97.\n\n"+
		"Play: Fix with AI\nFix the ticket.\n\n"+
		`Ticket: SRC-3 "Fix login". Read it with ticket_get before you start.`+"\n\n"+
		"This ticket was blocked by these tickets, all now done:\n- NEX-4 \"schema\": done\n\n"+
		"Read these memories with memory_get before you start; they are your context:\n- Interview (id m-1)\n\n"+
		"Instructions for this run from onik97; where they conflict with the play's, these win:\nTouch only the docs.",
		ComposeIncrementalPrompt(start), "a run's start on a live session is everything but the intro and footer")

	assert.Equal(t, "New message from onik97 at 2026-10-03 12:30:\nAnswered: Yes", ComposeIncrementalPrompt(playAnswer()),
		"an answer on a live session is only the answer")
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

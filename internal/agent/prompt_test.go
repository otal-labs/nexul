package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// playAnswer is a resumed play run on a session the harness lost: the play, the target, the memories, and the answer.
func playAnswer() PromptInput {
	return PromptInput{
		Intro: "INTRO", Footer: "FOOTER",
		Target:        `Ticket: SRC-3 "Fix login". Read it with ticket_get before you start.`,
		Play:          &PlayContext{Label: "Fix with AI", Instructions: "Fix the ticket.", Memories: []MemoryRef{{ID: "m-1", Name: "Interview"}}},
		Memories:      []MemoryRef{{ID: "m-1", Name: "Interview"}},
		RequestAuthor: "onik97", RequestBody: "Answered: Yes", RequestAt: time.Date(2026, 10, 3, 12, 30, 0, 0, time.UTC),
	}
}

func TestComposePrompt_PlayAnswer_CarriesThePlayAndTheAnswer(t *testing.T) {
	assert.Equal(t, "INTRO\n\n"+
		"Right now you are running a play, started by onik97.\n\n"+
		"Play: Fix with AI\nFix the ticket.\n\n"+
		`Ticket: SRC-3 "Fix login". Read it with ticket_get before you start.`+"\n\n"+
		"Read these memories with memory_get before you start; they are your context:\n- Interview (id m-1)\n\n"+
		"New message from onik97 at 2026-10-03 12:30:\nAnswered: Yes\n\n"+
		"FOOTER", ComposePrompt(playAnswer()))
}

func TestComposePrompt_FooterMemories_FollowTheRunInstructions(t *testing.T) {
	in := playAnswer()
	in.RequestBody = ""
	in.Play.Custom = "Touch only the docs."
	in.Play.Conclude = []MemoryRef{{ID: "m-9", Name: "Where tickets go"}}
	assert.True(t, strings.HasSuffix(ComposePrompt(in), "Touch only the docs.\n\n"+
		concludeLine+"\n- Where tickets go (id m-9)\n\nFOOTER"))
}

func TestComposePrompt_EmptyTemplatesAndAnUnreadPlayAreLeftOut(t *testing.T) {
	in := playAnswer()
	in.Intro, in.Footer = "", ""
	in.Play.Label, in.Play.Instructions = "", ""
	got := ComposePrompt(in)
	assert.True(t, strings.HasPrefix(got, "Right now you are running a play"), got)
	assert.True(t, strings.HasSuffix(got, "Answered: Yes"), got)
	assert.NotContains(t, got, "Play:")
}

func TestComposePrompt_TruncatesOldestContextFirstToFitCap(t *testing.T) {
	var msgs []ContextMessage
	for i := 0; i < 2000; i++ {
		msgs = append(msgs, ContextMessage{Author: "onik97", Body: strings.Repeat("x", 100)})
	}
	msgs = append(msgs, ContextMessage{Author: "onik97", Body: "the newest line, must survive"})

	out := ComposePrompt(PromptInput{
		Intro: DefaultIntro, Footer: DefaultFooter,
		ContextMessages: msgs,
		RequestAuthor:   "onik97",
		RequestBody:     "@Agent go",
	})

	assert.LessOrEqual(t, len(out), MaxPromptChars)
	assert.Contains(t, out, truncationNote)
	assert.Contains(t, out, "the newest line, must survive")
	assert.True(t, strings.HasSuffix(out, DefaultFooter))
}

func TestFitContext_KeepsEverythingWhenItFits(t *testing.T) {
	lines, truncated := fitContext([]ContextMessage{{Author: "a", Body: "b"}}, 1000)
	assert.False(t, truncated)
	assert.Equal(t, []string{"a: b"}, lines)
}

func TestFitContext_DropsOldestFirst(t *testing.T) {
	msgs := []ContextMessage{
		{Author: "a", Body: "oldest"},
		{Author: "a", Body: "middle"},
		{Author: "a", Body: "newest"},
	}
	// Budget only large enough for the newest line.
	lines, truncated := fitContext(msgs, len("a: newest")+1)
	assert.True(t, truncated)
	assert.Equal(t, []string{"a: newest"}, lines)
}

func TestFitContext_NegativeBudgetDropsEverything(t *testing.T) {
	lines, truncated := fitContext([]ContextMessage{{Author: "a", Body: "b"}}, -5)
	assert.True(t, truncated)
	assert.Empty(t, lines)
}

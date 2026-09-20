package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposePrompt_IncludesInstructionsContextAndRequest(t *testing.T) {
	out := ComposePrompt(PromptInput{
		ContextMessages: []ContextMessage{
			{Author: "onik97", Body: "first message"},
			{Author: "Agent", Body: "previous reply"},
		},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent do the thing",
	})
	assert.Contains(t, out, "whoami-style tool")
	assert.Contains(t, out, "memories index")
	assert.Contains(t, out, "onik97: first message")
	assert.Contains(t, out, "Agent: previous reply")
	assert.Contains(t, out, "New message from onik97:\n@Agent do the thing")
	assert.NotContains(t, out, truncationNote)
}

func TestComposePrompt_ExtraRequestBlocksFollowTheBody(t *testing.T) {
	tests := []struct {
		name   string
		blocks []string
		want   string
	}{
		{"none", nil, "New message from onik97:\n@Agent go"},
		{"one", []string{"Play: Fix with AI"}, "New message from onik97:\n@Agent go\n\nPlay: Fix with AI"},
		{"two in order", []string{"Play: Fix with AI", "Instructions from onik97:\nbe brief"}, "New message from onik97:\n@Agent go\n\nPlay: Fix with AI\n\nInstructions from onik97:\nbe brief"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := ComposePrompt(PromptInput{RequestAuthor: "onik97", RequestBody: "@Agent go", ExtraRequestBlocks: tt.blocks})
			assert.True(t, strings.HasSuffix(out, "\n\n"+tt.want), "prompt must end with the request block, got:\n%s", out)
		})
	}
}

func TestComposePrompt_TicketThreadIncludesTicket(t *testing.T) {
	out := ComposePrompt(PromptInput{
		Ticket:        &TicketContext{Title: "Fix the thing", Body: "It's broken."},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent status?",
	})
	assert.Contains(t, out, "Ticket: Fix the thing")
	assert.Contains(t, out, "It's broken.")
}

func TestComposePrompt_NoTicketOmitsTicketBlock(t *testing.T) {
	out := ComposePrompt(PromptInput{RequestAuthor: "onik97", RequestBody: "hi"})
	assert.NotContains(t, out, "Ticket:")
}

func TestComposePrompt_DocThreadIncludesDoc(t *testing.T) {
	out := ComposePrompt(PromptInput{
		Doc:           &DocContext{Title: "Runbook", Body: "Restart the service like so."},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent status?",
	})
	assert.Contains(t, out, "Doc: Runbook")
	assert.Contains(t, out, "Restart the service like so.")
	assert.NotContains(t, out, docTrimNote)
}

func TestComposePrompt_NoDocOmitsDocBlock(t *testing.T) {
	out := ComposePrompt(PromptInput{RequestAuthor: "onik97", RequestBody: "hi"})
	assert.NotContains(t, out, "Doc:")
}

func TestComposePrompt_OversizedDocBodyIsTrimmedWithNote(t *testing.T) {
	out := ComposePrompt(PromptInput{
		Doc:           &DocContext{Title: "Huge doc", Body: strings.Repeat("x", MaxPromptChars*2)},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent summarize",
	})
	assert.LessOrEqual(t, len(out), MaxPromptChars)
	assert.Contains(t, out, "Doc: Huge doc")
	assert.Contains(t, out, docTrimNote)
	assert.Contains(t, out, "New message from onik97:\n@Agent summarize")
}

func TestComposePrompt_TruncatesOldestContextFirstToFitCap(t *testing.T) {
	// Build enough context that the whole thing can't fit MaxPromptChars, and
	// assert the newest lines survive while the oldest are dropped with a note.
	var msgs []ContextMessage
	for i := 0; i < 2000; i++ {
		msgs = append(msgs, ContextMessage{Author: "onik97", Body: strings.Repeat("x", 100)})
	}
	msgs = append(msgs, ContextMessage{Author: "onik97", Body: "the newest line, must survive"})

	out := ComposePrompt(PromptInput{
		ContextMessages: msgs,
		RequestAuthor:   "onik97",
		RequestBody:     "@Agent go",
	})

	assert.LessOrEqual(t, len(out), MaxPromptChars)
	assert.Contains(t, out, truncationNote)
	assert.Contains(t, out, "the newest line, must survive")
}

func TestComposePrompt_HugeTicketBodyStillDropsAllContext(t *testing.T) {
	// ponytail: a pathologically huge ticket/request can still blow the cap
	// even after dropping all context — known ceiling, not handled beyond
	// that (the composer never truncates the ticket or request themselves).
	out := ComposePrompt(PromptInput{
		Ticket: &TicketContext{Title: "T", Body: strings.Repeat("y", MaxPromptChars)},
		ContextMessages: []ContextMessage{
			{Author: "onik97", Body: strings.Repeat("z", 500)},
		},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent go",
	})
	assert.NotContains(t, out, "zzz")
	assert.Contains(t, out, truncationNote)
}

func TestComposePrompt_MemoriesIndexRendersTheProjectScope(t *testing.T) {
	out := ComposePrompt(PromptInput{
		Memories: MemoriesIndex{
			Project: []MemoryItem{{Name: "Deploy quirks", WhenToUse: "use this if you are touching deploy config"}},
		},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent go",
	})
	assert.Contains(t, out, "This project's memories:")
	assert.Contains(t, out, "- Deploy quirks: use this if you are touching deploy config")
	assert.Contains(t, out, "memory_get")
	assert.Contains(t, out, "@Agent remember")
}

func TestComposePrompt_MemoriesIndexRendersWorkspaceBeforeProject(t *testing.T) {
	out := ComposePrompt(PromptInput{
		Memories: MemoriesIndex{
			Workspace: []MemoryItem{{Name: "Team tone", WhenToUse: "always applies"}},
			Project:   []MemoryItem{{Name: "Deploy quirks", WhenToUse: "use this if you are touching deploy config"}},
		},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent go",
	})
	assert.Contains(t, out, "Workspace memories:")
	assert.Contains(t, out, "- Team tone: always applies")
	assert.Contains(t, out, "This project's memories:")
	assert.Contains(t, out, "- Deploy quirks: use this if you are touching deploy config")
	assert.Less(t, strings.Index(out, "Workspace memories:"), strings.Index(out, "This project's memories:"), "workspace memories render before the project's own")
}

func TestComposePrompt_NoMemoriesFallsBackGracefully(t *testing.T) {
	out := ComposePrompt(PromptInput{RequestAuthor: "onik97", RequestBody: "@Agent go"})
	assert.Contains(t, out, "no memories saved yet")
	assert.Contains(t, out, "nexul-memory skill")
}

func TestComposePrompt_MemoriesIndexIsCappedRegardlessOfPromptBudget(t *testing.T) {
	var items []MemoryItem
	for i := 0; i < 500; i++ {
		items = append(items, MemoryItem{Name: "Memory", WhenToUse: strings.Repeat("x", 100)})
	}
	out := ComposePrompt(PromptInput{
		Memories:      MemoriesIndex{Project: items},
		RequestAuthor: "onik97",
		RequestBody:   "@Agent go",
	})
	// The memories section itself must respect MaxMemoriesIndexChars even
	// though the overall prompt has ample room under MaxPromptChars.
	idxStart := strings.Index(out, "This turn's memories index:")
	require.GreaterOrEqual(t, idxStart, 0)
	assert.Contains(t, out, memoriesTruncationNote)
	assert.Less(t, len(out)-idxStart, MaxMemoriesIndexChars+1000)
}

func TestFitMemoriesIndex_KeepsEverythingWhenItFits(t *testing.T) {
	lines, truncated := fitMemoriesIndex(MemoriesIndex{Project: []MemoryItem{{Name: "a", WhenToUse: "b"}}}, 1000)
	assert.False(t, truncated)
	assert.Equal(t, []string{"This project's memories:", "- a: b"}, lines)
}

func TestFitMemoriesIndex_WorkspaceBeforeProject(t *testing.T) {
	mem := MemoriesIndex{
		Workspace: []MemoryItem{{Name: "w", WhenToUse: "x"}},
		Project:   []MemoryItem{{Name: "a", WhenToUse: "b"}},
	}
	lines, truncated := fitMemoriesIndex(mem, 1000)
	assert.False(t, truncated)
	assert.Equal(t, []string{"Workspace memories:", "- w: x", "This project's memories:", "- a: b"}, lines)
}

func TestFitMemoriesIndex_DropsFromTheEndWhenOverBudget(t *testing.T) {
	mem := MemoriesIndex{Project: []MemoryItem{{Name: "a", WhenToUse: "b"}, {Name: "c", WhenToUse: "d"}}}
	lines, truncated := fitMemoriesIndex(mem, len("This project's memories:")+1+len("- a: b")+1)
	assert.True(t, truncated)
	assert.Equal(t, []string{"This project's memories:", "- a: b"}, lines)
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

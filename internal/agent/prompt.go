package agent

import (
	"fmt"
	"strings"
	"time"
)

// MaxPromptChars is the harness turn-input ceiling (T3 Code's limit today); oldest context is truncated first to stay under it.
const MaxPromptChars = 120_000

// MaxAttachmentBytes caps one attachment handed to the harness (T3's per-attachment limit, ticket 12).
// MaxTurnAttachmentBytes caps one turn's total attachment bytes across every body it draws them from.
const (
	MaxAttachmentBytes     = 10 << 20
	MaxTurnAttachmentBytes = 25 << 20
)

// MaxMemoryChars caps one memory inlined in full; MaxInlinedMemoryChars caps a whole run's selection.
const (
	MaxMemoryChars        = 20_000
	MaxInlinedMemoryChars = 60_000
)

// truncationNote is inserted when oldest context had to be dropped to fit MaxPromptChars.
const truncationNote = "(earlier messages omitted to fit the turn size limit)"

// MaxMemoriesIndexChars caps the rendered memories index well below MaxPromptChars.
const MaxMemoriesIndexChars = 4_000

// memoriesTruncationNote is appended when the memories index had to drop entries to fit MaxMemoriesIndexChars.
const memoriesTruncationNote = "(older memories omitted — curate the memory set if this happens often)"

// MemoryItem is one memory available to a turn. Most carry only a name and a when-to-use line for the index;
// an always-included memory also carries its full markdown Body, inlined rather than indexed (ticket 27).
type MemoryItem struct {
	Name           string
	WhenToUse      string
	AlwaysIncluded bool
	// Interview marks the project's interview memory, inlined first so a trim never drops it (ADR 0065).
	Interview bool
	Body      string
}

// MemoriesIndex is every memory available to a turn: the workspace's workspace-scoped memories, reaching
// every turn including a plain chat with no ticket or doc, and the current project's own memories, if any
// (ADR 0059). The zero value is nil-safe and renders as "no memories saved yet". Callers building a prompt
// split this with splitAlwaysIncluded before handing it to instructionsBlock/memoriesBlock, which render the
// index of the rest only.
type MemoriesIndex struct {
	Workspace []MemoryItem
	Project   []MemoryItem
}

// splitAlwaysIncluded separates always-included memories from the index; the project's interview leads them.
func splitAlwaysIncluded(mem MemoriesIndex) (index MemoriesIndex, always []InlinedMemory) {
	for _, m := range mem.Workspace {
		if !m.AlwaysIncluded {
			index.Workspace = append(index.Workspace, m)
			continue
		}
		always = append(always, InlinedMemory{Title: m.Name, Body: m.Body})
	}
	for _, m := range mem.Project {
		if !m.AlwaysIncluded {
			index.Project = append(index.Project, m)
			continue
		}
		if m.Interview {
			always = append([]InlinedMemory{{Title: m.Name, Body: m.Body}}, always...)
			continue
		}
		always = append(always, InlinedMemory{Title: m.Name, Body: m.Body})
	}
	return index, always
}

// TicketContext is a ticket thread's ticket, included in the prompt.
type TicketContext struct {
	Title string
	Body  string
}

// DocContext is a doc thread's doc, included in the prompt as markdown; trimmed with a note when oversized.
type DocContext struct {
	Title string
	Body  string
}

// docTrimNote is appended when a doc's body had to be cut to fit MaxPromptChars.
const docTrimNote = "\n\n(doc body trimmed to fit the turn size limit)"

// ContextMessage is one resolved context line; Author is already a display label, not a raw id.
type ContextMessage struct {
	Author string
	Body   string
	// At is rendered on every line so a reused thread's model can tell new lines from seen ones.
	At time.Time
}

// PromptInput is everything ComposePrompt needs: the instructions, context, and request blocks.
type PromptInput struct {
	// Ticket is non-nil only for a ticket thread.
	Ticket *TicketContext
	// Doc is non-nil only for a doc thread; mutually exclusive with Ticket.
	Doc *DocContext
	// ContextMessages are new messages since the last mention, oldest-first.
	ContextMessages []ContextMessage
	// Memories is the turn's index; zero value renders as "no memories saved yet", not an error.
	Memories      MemoriesIndex
	RequestAuthor string
	RequestBody   string
	// RequestAt timestamps the triggering mention in the request block.
	RequestAt time.Time
	// ExtraRequestBlocks follow the request body inside the request block, so a reused session still receives them.
	ExtraRequestBlocks []string
}

// ComposePrompt builds one turn's prompt: instructions, context, and request, oldest context dropped first.
func ComposePrompt(in PromptInput) string {
	instructions := instructionsBlock(in.Memories)
	request := requestBlock(in)

	// Ticket and Doc are mutually exclusive (one thread carries one target); a ticket's body is never
	// trimmed today, a doc's is, since only the doc thread ticket calls for it.
	targetBlock := ""
	if in.Ticket != nil {
		targetBlock = fmt.Sprintf("Ticket: %s\n\n%s", in.Ticket.Title, in.Ticket.Body)
	}
	if in.Doc != nil {
		targetBlock = docContextBlock(in.Doc, instructions, request)
	}

	// fixedLen mirrors the assembly below exactly, so the context-line budget is exact, not a guess.
	fixedLen := len(instructions) + len("\n\nConversation so far:") + len("\n\n") + len(request)
	if targetBlock != "" {
		fixedLen += len("\n\n") + len(targetBlock)
	}
	budget := MaxPromptChars - fixedLen - (len("\n") + len(truncationNote))
	lines, truncated := fitContext(in.ContextMessages, budget)

	var b strings.Builder
	b.WriteString(instructions)
	if targetBlock != "" {
		b.WriteString("\n\n")
		b.WriteString(targetBlock)
	}
	b.WriteString("\n\nConversation so far:")
	if truncated {
		b.WriteString("\n")
		b.WriteString(truncationNote)
	}
	for _, l := range lines {
		b.WriteString("\n")
		b.WriteString(l)
	}
	b.WriteString("\n\n")
	b.WriteString(request)
	return b.String()
}

// docContextBlock renders a doc thread's doc. History is trimmed first (fitContext, above); this only
// trims the doc body, with a note, on the rarer turn where even zero history wouldn't make it fit.
func docContextBlock(doc *DocContext, instructions, request string) string {
	full := fmt.Sprintf("Doc: %s\n\n%s", doc.Title, doc.Body)
	fixed := len(instructions) + len("\n\n") + len("\n\nConversation so far:") + len("\n\n") + len(request)
	budget := MaxPromptChars - fixed
	if len(full) <= budget {
		return full
	}
	prefix := fmt.Sprintf("Doc: %s\n\n", doc.Title)
	bodyBudget := budget - len(prefix) - len(docTrimNote)
	if bodyBudget < 0 {
		bodyBudget = 0
	}
	body := doc.Body
	if len(body) > bodyBudget {
		body = body[:bodyBudget]
	}
	return prefix + body + docTrimNote
}

// instructionsBlock is the turn's fixed text: agent identity, the account_whoami check, and the memories protocol.
func instructionsBlock(mem MemoriesIndex) string {
	return "You are Agent, Nexul's in-chat assistant. You reply as the " +
		"\"Agent\" participant, shown as \"via <user>\" for whoever mentioned " +
		"you, and you act only within that user's own Nexul permissions " +
		"through Nexul's MCP server.\n\n" +
		"Before doing anything else, call the MCP server's account_whoami tool " +
		"to confirm you can reach Nexul. If it fails, say so plainly " +
		"instead of guessing or acting further.\n\n" +
		memoriesBlock(mem)
}

// memoriesBlock's fallback text is duplicated in the nexul-memory skill file; edit both together.
func memoriesBlock(mem MemoriesIndex) string {
	fallback := "Memories: durable notes for agents, shared across the team, " +
		"not per-user. A memory belongs to the workspace, reaching every turn, " +
		"or to one project. Always-included memories at either scope are " +
		"inlined in full elsewhere in this prompt as standing rules to follow; " +
		"the index below lists the rest by title and when-to-use only — fetch " +
		"one's full content with the memory_get MCP tool using its id when its " +
		"when-to-use matches. Save a new one, or update an existing one, with " +
		"memory_create/memory_update; omit project_id to save at workspace " +
		"scope. Use your judgment to save a durable fact worth remembering, " +
		"and always save one when a user says something like \"@Agent remember " +
		"X\". Keep the set curated — update an existing memory instead of " +
		"creating a near-duplicate. (If you have the nexul-memory skill " +
		"installed, follow its fuller protocol instead of this summary.)"

	lines, truncated := fitMemoriesIndex(mem, MaxMemoriesIndexChars-len(fallback)-len("\n\n"))
	if len(lines) == 0 {
		return fallback + "\n\nThis turn's memories index: no memories saved yet."
	}
	var b strings.Builder
	b.WriteString(fallback)
	b.WriteString("\n\nThis turn's memories index:")
	if truncated {
		b.WriteString("\n")
		b.WriteString(memoriesTruncationNote)
	}
	for _, l := range lines {
		b.WriteString("\n")
		b.WriteString(l)
	}
	return b.String()
}

// ponytail: fitMemoriesIndex drops newest-first by a greedy trim, not relevance ranking; upgrade if this bites.
func fitMemoriesIndex(mem MemoriesIndex, budget int) ([]string, bool) {
	if budget < 0 {
		budget = 0
	}
	var lines []string
	if len(mem.Workspace) > 0 {
		lines = append(lines, "Workspace memories:")
		for _, m := range mem.Workspace {
			lines = append(lines, memoryLine(m))
		}
	}
	if len(mem.Project) > 0 {
		lines = append(lines, "This project's memories:")
		for _, m := range mem.Project {
			lines = append(lines, memoryLine(m))
		}
	}
	total := 0
	for _, l := range lines {
		total += len(l) + 1
	}
	if total <= budget {
		return lines, false
	}
	end := len(lines)
	for end > 0 && total > budget {
		end--
		total -= len(lines[end]) + 1
	}
	return lines[:end], true
}

func memoryLine(m MemoryItem) string {
	return fmt.Sprintf("- %s: %s", m.Name, m.WhenToUse)
}

// ponytail: fitContext drops oldest lines by a greedy trim, not a summarizer; upgrade if this bites.
func fitContext(msgs []ContextMessage, budget int) ([]string, bool) {
	if budget < 0 {
		budget = 0
	}
	lines := make([]string, len(msgs))
	total := 0
	for i, m := range msgs {
		lines[i] = fmt.Sprintf("%s%s: %s", stamp(m.At), m.Author, m.Body)
		total += len(lines[i]) + 1
	}
	if total <= budget {
		return lines, false
	}
	start := 0
	for start < len(lines) && total > budget {
		total -= len(lines[start]) + 1
		start++
	}
	return lines[start:], true
}

// stamp renders a history timestamp prefix; a zero time renders nothing.
func stamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return "[" + at.UTC().Format("2006-01-02 15:04") + "] "
}

func requestBlock(in PromptInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "New message from %s%s:\n%s", in.RequestAuthor, requestStamp(in.RequestAt), in.RequestBody)
	for _, block := range in.ExtraRequestBlocks {
		b.WriteString("\n\n")
		b.WriteString(block)
	}
	return b.String()
}

func requestStamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return " at " + at.UTC().Format("2006-01-02 15:04")
}

// ComposeIncrementalPrompt is ComposePrompt's shape for a reused harness session: only what's new goes over.
func ComposeIncrementalPrompt(in PromptInput) string {
	request := requestBlock(in)
	header := "New messages since your last turn:"
	budget := MaxPromptChars - len(header) - len("\n\n") - len(request) - (len("\n") + len(truncationNote))
	lines, truncated := fitContext(in.ContextMessages, budget)

	var b strings.Builder
	if len(lines) > 0 {
		b.WriteString(header)
		if truncated {
			b.WriteString("\n")
			b.WriteString(truncationNote)
		}
		for _, l := range lines {
			b.WriteString("\n")
			b.WriteString(l)
		}
		b.WriteString("\n\n")
	}
	b.WriteString(request)
	return b.String()
}

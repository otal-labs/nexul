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

// truncationNote is inserted when oldest context had to be dropped to fit MaxPromptChars.
const truncationNote = "(earlier messages omitted to fit the turn size limit)"

// TemplateKind is the instance-only template kind holding the full prompt's Intro and Footer (ADR 0111).
const (
	TemplateKind   = "agent_prompt"
	TemplateIntro  = "intro"
	TemplateFooter = "footer"
)

// DefaultIntro opens every full prompt until the instance edits its Intro template.
const DefaultIntro = "You are Agent, Nexul's in-chat assistant. You reply as the \"Agent\" participant, shown as \"via <user>\" " +
	"for whoever mentioned you or started the run, and you act only within that user's own Nexul permissions through " +
	"Nexul's MCP server.\n\n" +
	"Before doing anything else, call the MCP server's account_get tool with no arguments to confirm you can reach " +
	"Nexul. If it fails, say so plainly instead of guessing or acting further."

// DefaultFooter closes every full prompt until the instance edits its Footer template.
const DefaultFooter = "Standing rules:\n" +
	"- A ticket's body is its spec: change it only when a person asks. Put anything lasting you learn in a note with " +
	"message_post's file, and if the spec looks wrong, say so and suggest the person edit it.\n" +
	"- Memories are the team's shared notes for agents, one project each. Save a durable fact worth keeping with " +
	"memory_create, or memory_update the one that already covers it, passing the project's project_id; always save one " +
	"when someone says \"remember X\". Find others with memory_list and read them with memory_get when a task needs them."

// The fixed lines naming a turn's context by reference; each names the tool that reads it (ADR 0111).
const (
	ticketLine   = "Ticket: %s %q. Read it with ticket_get before you start."
	docLine      = "Doc: %q (id %s). Read it with doc_get before you start."
	imagesLine   = "The images in its body are attached to this message, in order."
	memoriesLine = "Read these memories with memory_get before you start; they are your context:"
	concludeLine = "When the work is done, before your final reply, read these memories with memory_get and follow them to " +
		"conclude the run, for example to decide which column the ticket now belongs in:"
)

// PromptTexts returns the fixed texts a prompt hands the agent, so the MCP surface test can check the tools they name.
func PromptTexts() []string {
	return []string{DefaultIntro, DefaultFooter, ticketLine, docLine, memoriesLine, concludeLine}
}

// MemoryItem is one of a project's memories as the pipeline sees it: enough to name it, never its body.
type MemoryItem struct {
	ID             string
	Name           string
	AlwaysIncluded bool
	// Interview marks the project's interview memory, named first (ADR 0065).
	Interview bool
}

// MemoryRef names one memory for the agent to read with memory_get.
type MemoryRef struct {
	ID   string
	Name string
}

// PlayContext is a play run's part of a turn: the play, its link blocks, the memories it names, and the starter's
// instructions for the run. A play turn carries no conversation history (ADR 0111).
type PlayContext struct {
	// Label and Instructions are the play's; an empty Label leaves the play block out.
	Label        string
	Instructions string
	Blocks       []string
	Memories     []MemoryRef
	Custom       string
	// Conclude are the run's footer memories, named after its instructions (ADR 0112).
	Conclude []MemoryRef
}

// ContextMessage is one resolved context line; Author is already a display label, not a raw id.
type ContextMessage struct {
	Author string
	Body   string
	// At is rendered on every line so a reused thread's model can tell new lines from seen ones.
	At time.Time
}

// PromptInput is everything the composers need; empty fields render nothing.
type PromptInput struct {
	Intro  string
	Footer string
	// Target is the thread's ticket or doc line, "" for any other thread.
	Target string
	// Play is non-nil for a play run.
	Play            *PlayContext
	Memories        []MemoryRef
	ContextMessages []ContextMessage
	// RequestAuthor is whoever mentioned the Agent or started the run; RequestBody is "" for a play run's start.
	RequestAuthor string
	RequestBody   string
	RequestAt     time.Time
}

// ComposePrompt builds a fresh session's prompt: intro, what is running, its context by reference, the conversation
// (oldest lines dropped first to fit), the request, and the footer.
func ComposePrompt(in PromptInput) string {
	head := append([]string{in.Intro}, contextSections(in)...)
	tail := []string{requestBlock(in), in.Footer}
	header := "Conversation so far:"
	fixed := len(joinSections(append(append([]string{}, head...), tail...)...))
	budget := MaxPromptChars - fixed - len("\n\n") - len(header) - len("\n") - len(truncationNote)
	conversation := conversationBlock(header, in.ContextMessages, budget)
	return joinSections(append(append(head, conversation), tail...)...)
}

// ComposeIncrementalPrompt is the prompt for a reused harness session (ADR 0106): no intro or footer, only what is new.
// A play run's start sends its context sections; any other turn sends the new messages and the request.
func ComposeIncrementalPrompt(in PromptInput) string {
	if in.Play != nil && in.RequestBody == "" {
		return joinSections(contextSections(in)...)
	}
	request := requestBlock(in)
	header := "New messages since your last turn:"
	budget := MaxPromptChars - len(request) - len("\n\n") - len(header) - len("\n") - len(truncationNote)
	return joinSections(conversationBlock(header, in.ContextMessages, budget), request)
}

// contextSections is what is running and its context, every part named for the agent to read itself.
func contextSections(in PromptInput) []string {
	if in.Play == nil {
		source := fmt.Sprintf("Right now you are running @Agent in a chat, mentioned by %s.", in.RequestAuthor)
		return []string{source, in.Target, memoriesBlock(in.Memories)}
	}
	sections := []string{fmt.Sprintf("Right now you are running a play, started by %s.", in.RequestAuthor)}
	if in.Play.Label != "" {
		sections = append(sections, "Play: "+in.Play.Label+"\n"+in.Play.Instructions)
	}
	sections = append(sections, in.Target)
	sections = append(sections, in.Play.Blocks...)
	sections = append(sections, memoriesBlock(in.Memories))
	if in.Play.Custom != "" {
		sections = append(sections, fmt.Sprintf("Instructions for this run from %s; where they conflict with the play's, these win:\n%s", in.RequestAuthor, in.Play.Custom))
	}
	return append(sections, memoryList(concludeLine, in.Play.Conclude))
}

func memoriesBlock(refs []MemoryRef) string {
	return memoryList(memoriesLine, refs)
}

// memoryList names each memory under its opening line, empty when there is none to name.
func memoryList(opening string, refs []MemoryRef) string {
	if len(refs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(opening)
	for _, m := range refs {
		fmt.Fprintf(&b, "\n- %s (id %s)", m.Name, m.ID)
	}
	return b.String()
}

// conversationBlock renders msgs under header within budget; nothing to show, and nothing dropped, renders "".
func conversationBlock(header string, msgs []ContextMessage, budget int) string {
	lines, truncated := fitContext(msgs, budget)
	if len(lines) == 0 && !truncated {
		return ""
	}
	var b strings.Builder
	b.WriteString(header)
	if truncated {
		b.WriteString("\n" + truncationNote)
	}
	for _, l := range lines {
		b.WriteString("\n" + l)
	}
	return b.String()
}

// joinSections joins the non-empty sections with a blank line between them.
func joinSections(sections ...string) string {
	kept := make([]string, 0, len(sections))
	for _, s := range sections {
		if s != "" {
			kept = append(kept, s)
		}
	}
	return strings.Join(kept, "\n\n")
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
	if in.RequestBody == "" {
		return ""
	}
	return fmt.Sprintf("New message from %s%s:\n%s", in.RequestAuthor, requestStamp(in.RequestAt), in.RequestBody)
}

func requestStamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return " at " + at.UTC().Format("2006-01-02 15:04")
}

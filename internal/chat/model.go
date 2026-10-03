// Package chat implements the chat domain core: conversations, messages, and per-user unread state.
package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
)

// Kind is a conversation's shape.
type Kind string

const (
	KindChannel       Kind = "channel"
	KindDM            Kind = "dm"
	KindChannelThread Kind = "channel_thread"
	KindTicketThread  Kind = "ticket_thread"
	KindDocThread     Kind = "doc_thread"
	// KindInterviewThread is a project's interview conversation, where the Interview play asks its questions.
	KindInterviewThread Kind = "interview_thread"
	// KindVoiceChannel is a real conversation like any channel; internal/voice owns the LiveKit side.
	KindVoiceChannel Kind = "voice_channel"
)

// GeneralChannelName is the fixed, auto-created default channel every workspace gets.
const GeneralChannelName = "general"

// AgentHandle is the fixed @Agent mention target; never a real user id.
const AgentHandle = "Agent"

// Conversation is one channel, DM, channel thread, or ticket thread.
type Conversation struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Kind        Kind   `json:"kind"`
	// Name is the channel display name (as typed, e.g. "general"); empty for every other kind.
	Name string `json:"name,omitempty"`
	// TicketID is set only for KindTicketThread; the unique index on it enforces one thread per ticket.
	TicketID string `json:"ticket_id,omitempty"`
	// DocID is set only for KindDocThread; the unique index on it enforces one thread per doc.
	DocID string `json:"doc_id,omitempty"`
	// ProjectID is set only for KindInterviewThread; the unique index on it enforces one interview thread per project.
	ProjectID string `json:"project_id,omitempty"`
	// ParentMessageID is set only for KindChannelThread; no create use-case sets it yet.
	ParentMessageID string    `json:"parent_message_id,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// General marks a workspace's #general, which may be renamed but never deleted, and is always public.
	General bool `json:"general,omitempty"`
	// Private marks a channel only its members read (ADR 0098); its members are its ParticipantIDs.
	Private bool `json:"private"`
	// AgentThreadID is the durable T3 thread id, reused across mentions; internal to the agent pipeline.
	AgentThreadID string `json:"-"`
	// AgentSyncedAt is how far the agent pipeline has sent history as turn context; only later messages are new.
	AgentSyncedAt time.Time `json:"-"`
	// ParticipantIDs is populated for a DM and a private channel; a public channel's readers are implicit.
	ParticipantIDs []string `json:"participant_ids,omitempty"`
}

// membersOnly reports a conversation read by its members alone: a DM or a private channel.
func (c *Conversation) membersOnly() bool {
	return c.Kind == KindDM || c.Private
}

// MentionKind labels a parsed mention's target.
type MentionKind string

const (
	MentionUser  MentionKind = "user"
	MentionAgent MentionKind = "agent"
)

// Mention is one @-mention parsed from a message body; resolving a handle to a live account is the picker's job.
type Mention struct {
	Kind   MentionKind `json:"kind"`
	Handle string      `json:"handle"`
}

// AuthorKind distinguishes how a message renders; AuthorID is always a real user id, even for agent/system.
type AuthorKind string

const (
	AuthorUser   AuthorKind = "user"
	AuthorAgent  AuthorKind = "agent"
	AuthorSystem AuthorKind = "system"
)

// Message is one chat message: markdown body, own edit/delete, a nullable attachment door.
type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	AuthorID       string     `json:"author_id"`
	AuthorKind     AuthorKind `json:"author_kind"`
	Body           string     `json:"body"`
	Mentions       []Mention  `json:"mentions"`
	// AttachmentID is set only on a note: the conversation file holding the note's markdown (ADR 0108).
	AttachmentID string     `json:"attachment_id,omitempty"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	// DeletedAt marks a soft delete; the row stays so surrounding messages keep their order.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	// Reactions are filled on reads, in the order each emoji was first used; a deleted message has none.
	Reactions []Reaction `json:"reactions,omitempty"`
	// Handoffs is the work an Agent reply handed to other agents (ADR 0116); nil on any other message.
	Handoffs []Handoff `json:"handoffs,omitempty"`
}

// Handoff is work an Agent reply handed to another agent, as the reply stores it and the live stream pushes it.
type Handoff struct {
	ID     string        `json:"id"`
	Driver string        `json:"driver"`
	Model  string        `json:"model"`
	Title  string        `json:"title"`
	Prompt string        `json:"prompt"`
	State  string        `json:"state"`
	Reply  string        `json:"reply"`
	Steps  []HandoffStep `json:"steps"`
}

// HandoffStep is one step of a handed-off agent's conversation, shaped like a trail step.
type HandoffStep struct {
	Kind    harness.ActivityKind `json:"kind"`
	CallID  string               `json:"call_id,omitempty"`
	Tool    string               `json:"tool,omitempty"`
	Summary string               `json:"summary"`
	Detail  string               `json:"detail,omitempty"`
	At      time.Time            `json:"at"`
}

// NewHandoff is h as chat stores and pushes it.
func NewHandoff(h harness.Handoff) Handoff {
	steps := make([]HandoffStep, 0, len(h.Steps))
	for _, a := range h.Steps {
		steps = append(steps, HandoffStep(a))
	}
	return Handoff{ID: h.ID, Driver: h.Driver, Model: h.Model, Title: h.Title, Prompt: h.Prompt, State: h.State, Reply: h.Reply, Steps: steps}
}

// maxHandoffBytes caps the hand-offs one reply stores, as JSON.
const maxHandoffBytes = 256 << 10

// storedHandoffs is hs as a reply stores them, nil for none, dropping the oldest steps until they fit maxHandoffBytes.
// ponytail: replies alone can still pass the cap (20 of T3's 32 KiB results); cut replies too if that ever shows up.
func storedHandoffs(hs []harness.Handoff) ([]Handoff, error) {
	if len(hs) == 0 {
		return nil, nil
	}
	out := make([]Handoff, 0, len(hs))
	for _, h := range hs {
		out = append(out, NewHandoff(h))
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("measure hand-offs: %w", err)
	}
	type stepRef struct {
		handoff, step int
		at            time.Time
		size          int
	}
	var refs []stepRef
	for i, h := range out {
		for j, s := range h.Steps {
			step, err := json.Marshal(s)
			if err != nil {
				return nil, fmt.Errorf("measure a hand-off step: %w", err)
			}
			refs = append(refs, stepRef{i, j, s.At, len(step)})
		}
	}
	slices.SortStableFunc(refs, func(a, b stepRef) int { return a.at.Compare(b.at) })
	size, drop := len(raw), map[[2]int]bool{}
	for _, r := range refs {
		if size <= maxHandoffBytes {
			break
		}
		size -= r.size
		drop[[2]int{r.handoff, r.step}] = true
	}
	for i := range out {
		kept := out[i].Steps[:0]
		for j, s := range out[i].Steps {
			if !drop[[2]int{i, j}] {
				kept = append(kept, s)
			}
		}
		out[i].Steps = kept
	}
	return out, nil
}

// Reaction is one emoji on a message and the people who reacted with it, earliest first.
type Reaction struct {
	Emoji   string   `json:"emoji"`
	UserIDs []string `json:"user_ids"`
}

// NoteFileInput is the markdown file a note carries, as posted; Name is forced to end in .md.
type NoteFileInput struct {
	Name     string `json:"name"`
	Markdown string `json:"markdown"`
}

// NoteFile is a note's markdown file, stored as an attachment owned by the note's conversation (ADR 0108).
type NoteFile struct {
	ID       string
	Name     string
	Markdown string
}

// mentionPattern matches a login-shaped @token; it doesn't validate against real workspace members.
var mentionPattern = regexp.MustCompile(`@([\w-]+)`)

// ParseMentions extracts @user and @Agent tokens; "agent" matches case-insensitively but stores as AgentHandle.
func ParseMentions(body string) []Mention {
	matches := mentionPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	mentions := make([]Mention, 0, len(matches))
	for _, m := range matches {
		handle := m[1]
		key := strings.ToLower(handle)
		if seen[key] {
			continue
		}
		seen[key] = true
		if key == strings.ToLower(AgentHandle) {
			mentions = append(mentions, Mention{Kind: MentionAgent, Handle: AgentHandle})
			continue
		}
		mentions = append(mentions, Mention{Kind: MentionUser, Handle: handle})
	}
	return mentions
}

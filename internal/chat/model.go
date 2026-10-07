// Package chat implements the chat domain core: conversations, messages, and per-user unread state.
package chat

import (
	"encoding/json"
	"regexp"
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
	// AgentSeen is the newest harness turn Nexul saw end on AgentThreadID; a later one there is followed (ADR 0127).
	AgentSeen string `json:"-"`
	// ParticipantIDs is populated for a DM and a private channel; a public channel's readers are implicit.
	ParticipantIDs []string `json:"participant_ids,omitempty"`
}

// MembersOnly reports a conversation read by its members alone: a DM or a private channel.
func (c *Conversation) MembersOnly() bool {
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

// AuthorKind distinguishes how a message renders; AuthorID is a real user id for every kind but a bot (ADR 0129).
type AuthorKind string

const (
	AuthorUser   AuthorKind = "user"
	AuthorAgent  AuthorKind = "agent"
	AuthorSystem AuthorKind = "system"
	// AuthorBot is a bot's post through its webhook URL; AuthorID is the bot's id, not a user's.
	AuthorBot AuthorKind = "bot"
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
	// Via names the harness its author wrote it in, such as T3, when it was relayed from there; empty when written in Nexul.
	Via string `json:"via,omitempty"`
	// AuthorName and AuthorAvatarURL are what a bot message showed when posted, kept through the bot's rename or delete.
	AuthorName      string `json:"author_name,omitempty"`
	AuthorAvatarURL string `json:"author_avatar_url,omitempty"`
	// Embeds are a bot message's embeds as the sender posted them, JSON chat stores without reading.
	Embeds json.RawMessage `json:"embeds,omitempty"`
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

// Package chat implements the chat domain core: conversations, messages, and per-user unread state.
package chat

import (
	"regexp"
	"strings"
	"time"
)

// Kind is a conversation's shape.
type Kind string

const (
	KindChannel       Kind = "channel"
	KindDM            Kind = "dm"
	KindChannelThread Kind = "channel_thread"
	KindTicketThread  Kind = "ticket_thread"
	KindDocThread     Kind = "doc_thread"
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
	// Name is the channel display name (lower-cased, e.g. "general"); empty for every other kind.
	Name string `json:"name,omitempty"`
	// TicketID is set only for KindTicketThread; the unique index on it enforces one thread per ticket.
	TicketID string `json:"ticket_id,omitempty"`
	// DocID is set only for KindDocThread; the unique index on it enforces one thread per doc.
	DocID string `json:"doc_id,omitempty"`
	// ParentMessageID is set only for KindChannelThread; no create use-case sets it yet.
	ParentMessageID string    `json:"parent_message_id,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// AgentThreadID is the durable T3 thread id, reused across mentions; internal to the agent pipeline.
	AgentThreadID string `json:"-"`
	// AgentSyncedAt is how far the agent pipeline has sent history as turn context; only later messages are new.
	AgentSyncedAt time.Time `json:"-"`
	// ParticipantIDs is only populated for KindDM; channels' implicit membership makes the list misleading there.
	ParticipantIDs []string `json:"participant_ids,omitempty"`
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
	// AttachmentID is a nullable attachments door; no v1 use-case sets it yet.
	AttachmentID string     `json:"attachment_id,omitempty"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	// DeletedAt marks a soft delete; the row stays so surrounding messages keep their order.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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

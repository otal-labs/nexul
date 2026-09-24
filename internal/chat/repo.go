package chat

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DocAccess is chat's consumer-side seam onto access (ADR 0017), gating a doc_thread conversation's
// listing, messages, and get-or-create by docs:thread (ADR 0057); wired to access.Service.Can.
type DocAccess interface {
	Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error)
}

// Repo is the consumer-side persistence contract for chat; mutations carry outbox events.
type Repo interface {
	// CreateConversation surfaces a duplicate channel/thread as ErrConflict; callers re-fetch instead of failing.
	CreateConversation(ctx context.Context, c *Conversation, participantIDs []string, evts ...eventbus.OutboxEvent) error
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	// GetChannelByName finds a workspace's channel by its (lower-cased) name; apperrs.ErrNotFound if none exists yet.
	GetChannelByName(ctx context.Context, workspaceID, name string) (*Conversation, error)
	// GetTicketThread returns ErrNotFound if the thread hasn't been lazily created yet.
	GetTicketThread(ctx context.Context, ticketID string) (*Conversation, error)
	// HasTicketThreads batches the exists-check for many ticket ids into one query.
	HasTicketThreads(ctx context.Context, ticketIDs []string) (map[string]bool, error)
	// GetDocThread returns ErrNotFound if the thread hasn't been lazily created yet.
	GetDocThread(ctx context.Context, docID string) (*Conversation, error)
	// GetInterviewThread returns ErrNotFound if the project's interview thread hasn't been lazily created yet.
	GetInterviewThread(ctx context.Context, projectID string) (*Conversation, error)
	// ListConversationsForUser includes every workspace channel plus conversations with an explicit participant row.
	ListConversationsForUser(ctx context.Context, workspaceID, userID string) ([]*Conversation, error)

	// SetAgentThread is set on the first agent turn, and again if the backend transparently recreates a gone thread.
	SetAgentThread(ctx context.Context, conversationID, threadID string) error
	// SetAgentSyncedAt is set once a turn's updates are drained, so the next mention only sends what's new.
	SetAgentSyncedAt(ctx context.Context, conversationID string, at time.Time) error

	CreateMessage(ctx context.Context, m *Message, evts ...eventbus.OutboxEvent) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	// ListMessages returns a conversation's messages oldest-first, capped at limit.
	ListMessages(ctx context.Context, conversationID string, limit int) ([]*Message, error)
	// ListMessagesSince returns only messages newer than since, for agent turn context of what's new.
	ListMessagesSince(ctx context.Context, conversationID string, since time.Time) ([]*Message, error)
	// UpdateMessage enqueues the given outbox events in the same transaction; the message must exist.
	UpdateMessage(ctx context.Context, id, body string, mentions []Mention, editedAt time.Time, evts ...eventbus.OutboxEvent) error
	// DeleteMessage soft-deletes so thread ordering survives; enqueues outbox events in the same transaction.
	DeleteMessage(ctx context.Context, id string, deletedAt time.Time, evts ...eventbus.OutboxEvent) error

	// MarkRead advances userID's read cursor on conversationID to at (upsert).
	MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error
	// UnreadCounts includes a conversation with zero unread rather than omitting it; Kind/DocID let the
	// use-case drop doc threads the caller lacks docs:thread on before the count reaches the caller.
	UnreadCounts(ctx context.Context, workspaceID, userID string) ([]UnreadCount, error)
}

// UnreadCount is one conversation's unread state.
type UnreadCount struct {
	ConversationID string
	Kind           Kind
	DocID          string
	Count          int
}

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
	// RenameConversation surfaces a duplicate channel name as ErrConflict, the same as CreateConversation.
	RenameConversation(ctx context.Context, id, name string, at time.Time, evts ...eventbus.OutboxEvent) error
	// DeleteConversation removes the conversation with its messages, participants, read state, attachments, and bots.
	DeleteConversation(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	// SetChannelPrivate switches a channel and replaces its members in one transaction; a public channel keeps none.
	SetChannelPrivate(ctx context.Context, id string, private bool, memberIDs []string, at time.Time, evts ...eventbus.OutboxEvent) error
	// AddParticipants adds members to a conversation; someone already in it stays as they are.
	AddParticipants(ctx context.Context, id string, userIDs []string, at time.Time, evts ...eventbus.OutboxEvent) error
	// RemoveParticipants removes members from a conversation; someone not in it is skipped.
	RemoveParticipants(ctx context.Context, id string, userIDs []string, evts ...eventbus.OutboxEvent) error
	// GetChannelByName finds a workspace's channel by its name, ignoring case; apperrs.ErrNotFound if none exists yet.
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
	// SetAgentSyncedAt is set once a turn's prompt reaches the harness, so the next mention only sends what's new.
	SetAgentSyncedAt(ctx context.Context, conversationID string, at time.Time) error
	// SetAgentSeen records the newest harness turn Nexul saw end on the conversation's agent thread.
	SetAgentSeen(ctx context.Context, conversationID, marker string) error
	// GetConversationByAgentThread is the conversation whose agent thread is threadID; ErrNotFound for none.
	GetConversationByAgentThread(ctx context.Context, threadID string) (*Conversation, error)

	CreateMessage(ctx context.Context, m *Message, evts ...eventbus.OutboxEvent) error
	// CreateBotMessage inserts a bot's message, counts the post on its live bot, and records audit as the bot's action,
	// in one transaction; ErrNotFound once the bot is deleted.
	CreateBotMessage(ctx context.Context, m *Message, audit string, evts ...eventbus.OutboxEvent) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	// ListMessages returns a conversation's messages oldest-first, capped at limit.
	ListMessages(ctx context.Context, conversationID string, limit int) ([]*Message, error)
	// ListMessagesSince returns only messages newer than since, for agent turn context of what's new.
	ListMessagesSince(ctx context.Context, conversationID string, since time.Time) ([]*Message, error)
	// UpdateMessage enqueues the given outbox events in the same transaction; the message must exist.
	UpdateMessage(ctx context.Context, id, body string, mentions []Mention, editedAt time.Time, evts ...eventbus.OutboxEvent) error
	// DeleteMessage soft-deletes so thread ordering survives; enqueues outbox events in the same transaction.
	DeleteMessage(ctx context.Context, id string, deletedAt time.Time, evts ...eventbus.OutboxEvent) error
	// SetReaction adds or removes userID's emoji on a message; the events are enqueued only when that changed a row.
	SetReaction(ctx context.Context, messageID, userID, emoji string, reacted bool, at time.Time, evts ...eventbus.OutboxEvent) error

	// CreateNote stores a note's file as an attachment of m's conversation and m pointing at it, in one transaction.
	CreateNote(ctx context.Context, m *Message, file *NoteFile, evts ...eventbus.OutboxEvent) error
	// ListNoteFiles returns the note files with the given attachment ids; an id with no file is left out.
	ListNoteFiles(ctx context.Context, attachmentIDs []string) ([]*NoteFile, error)
	// ReplaceNoteFile overwrites m's file with markdown and moves m's updated_at to at; ErrNotFound once m is deleted.
	ReplaceNoteFile(ctx context.Context, m *Message, markdown string, at time.Time, evts ...eventbus.OutboxEvent) error
	// DeleteNote soft-deletes m with its file and the images of its conversation named by imageIDs, in one transaction.
	DeleteNote(ctx context.Context, m *Message, imageIDs []string, deletedAt time.Time, evts ...eventbus.OutboxEvent) error
	// IsNoteFile reports whether a message of the conversation carries attachmentID as its note's file.
	IsNoteFile(ctx context.Context, conversationID, attachmentID string) (bool, error)

	// MarkRead advances userID's read cursor on conversationID to at (upsert).
	MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error
	// UnreadCounts includes a conversation with zero unread rather than omitting it.
	UnreadCounts(ctx context.Context, workspaceID, userID string) ([]UnreadCount, error)
}

// UnreadCount is one conversation's unread state.
type UnreadCount struct {
	ConversationID string
	Count          int
}

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ chat.Repo = (*ChatRepo)(nil)

type ChatRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// CreateConversation trips a unique index on a duplicate channel name or ticket thread.
func (r *ChatRepo) CreateConversation(ctx context.Context, c *chat.Conversation, participantIDs []string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.CreateConversation(ctx, sqlcgen.CreateConversationParams{
			ID: c.ID, WorkspaceID: c.WorkspaceID, Kind: string(c.Kind), Name: c.Name,
			TicketID:        sql.NullString{String: c.TicketID, Valid: c.TicketID != ""},
			DocID:           sql.NullString{String: c.DocID, Valid: c.DocID != ""},
			ParentMessageID: c.ParentMessageID, CreatedBy: c.CreatedBy,
			CreatedAt: c.CreatedAt.Unix(), UpdatedAt: c.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert conversation %s: %w", c.ID, classifyWriteErr(err))
		}
		for _, uid := range participantIDs {
			if err := q.InsertConversationParticipant(ctx, sqlcgen.InsertConversationParticipantParams{
				ConversationID: c.ID, UserID: uid, CreatedAt: c.CreatedAt.Unix(),
			}); err != nil {
				return fmt.Errorf("add participant %s to conversation %s: %w", uid, c.ID, classifyWriteErr(err))
			}
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) GetConversation(ctx context.Context, id string) (*chat.Conversation, error) {
	row, err := r.q.GetConversation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", id, notFoundIfNoRows(err))
	}
	return toConversation(row), nil
}

func (r *ChatRepo) GetChannelByName(ctx context.Context, workspaceID, name string) (*chat.Conversation, error) {
	row, err := r.q.GetChannelByName(ctx, sqlcgen.GetChannelByNameParams{WorkspaceID: workspaceID, Name: name})
	if err != nil {
		return nil, fmt.Errorf("get channel %q in workspace %s: %w", name, workspaceID, notFoundIfNoRows(err))
	}
	return toConversation(row), nil
}

func (r *ChatRepo) GetTicketThread(ctx context.Context, ticketID string) (*chat.Conversation, error) {
	row, err := r.q.GetTicketThread(ctx, nullString(ticketID))
	if err != nil {
		return nil, fmt.Errorf("get ticket thread for ticket %s: %w", ticketID, notFoundIfNoRows(err))
	}
	return toConversation(row), nil
}

func (r *ChatRepo) GetDocThread(ctx context.Context, docID string) (*chat.Conversation, error) {
	row, err := r.q.GetDocThread(ctx, nullString(docID))
	if err != nil {
		return nil, fmt.Errorf("get doc thread for doc %s: %w", docID, notFoundIfNoRows(err))
	}
	return toConversation(row), nil
}

// HasTicketThreads batches the thread-exists check for the board's card indicator; ids with no thread are absent.
func (r *ChatRepo) HasTicketThreads(ctx context.Context, ticketIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(ticketIDs))
	if len(ticketIDs) == 0 {
		return out, nil
	}
	ids := make([]sql.NullString, len(ticketIDs))
	for i, id := range ticketIDs {
		ids[i] = nullString(id)
	}
	rows, err := r.q.ListTicketIDsWithThreads(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("has ticket threads: %w", err)
	}
	for _, id := range rows {
		out[id.String] = true
	}
	return out, nil
}

// ListConversationsForUser returns every public channel plus every conversation the user has a participant row in.
func (r *ChatRepo) ListConversationsForUser(ctx context.Context, workspaceID, userID string) ([]*chat.Conversation, error) {
	rows, err := r.q.ListConversationsForUser(ctx, sqlcgen.ListConversationsForUserParams{UserID: userID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, fmt.Errorf("list conversations for user %s: %w", userID, err)
	}
	out := toConversations(rows)
	if err := r.attachDMParticipants(ctx, out); err != nil {
		return nil, fmt.Errorf("attach dm participants: %w", err)
	}
	return out, nil
}

// attachDMParticipants batch-fills ParticipantIDs on every DM — one query for the page, not one per conversation.
func (r *ChatRepo) attachDMParticipants(ctx context.Context, cs []*chat.Conversation) error {
	byID := make(map[string]*chat.Conversation, len(cs))
	ids := make([]string, 0, len(cs))
	for _, c := range cs {
		if c.Kind != chat.KindDM {
			continue
		}
		byID[c.ID] = c
		ids = append(ids, c.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.q.ListParticipantsByConversationIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("list dm participants: %w", err)
	}
	for _, row := range rows {
		if c, ok := byID[row.ConversationID]; ok {
			c.ParticipantIDs = append(c.ParticipantIDs, row.UserID)
		}
	}
	return nil
}

func (r *ChatRepo) CreateMessage(ctx context.Context, m *chat.Message, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		mentionsJSON, err := json.Marshal(m.Mentions)
		if err != nil {
			return fmt.Errorf("marshal mentions: %w", err)
		}
		authorKind := m.AuthorKind
		if authorKind == "" {
			authorKind = chat.AuthorUser
		}
		err = r.q.WithTx(tx).CreateMessage(ctx, sqlcgen.CreateMessageParams{
			ID: m.ID, ConversationID: m.ConversationID, AuthorID: m.AuthorID, AuthorKind: string(authorKind),
			Body: m.Body, Mentions: string(mentionsJSON), AttachmentID: sql.NullString{String: m.AttachmentID, Valid: m.AttachmentID != ""},
			CreatedAt: m.CreatedAt.Unix(), UpdatedAt: m.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert message %s: %w", m.ID, classifyWriteErr(err))
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

// SetAgentThread persists a conversation's durable T3 thread id (ticket 13).
func (r *ChatRepo) SetAgentThread(ctx context.Context, conversationID, threadID string) error {
	n, err := r.q.SetAgentThread(ctx, sqlcgen.SetAgentThreadParams{AgentThreadID: threadID, ID: conversationID})
	if err != nil {
		return fmt.Errorf("set agent thread for conversation %s: %w", conversationID, classifyWriteErr(err))
	}
	if n == 0 {
		return fmt.Errorf("set agent thread for conversation %s: %w", conversationID, apperrs.ErrNotFound)
	}
	return nil
}

// SetAgentSyncedAt advances a conversation's agent-context sync cursor (ticket 13).
func (r *ChatRepo) SetAgentSyncedAt(ctx context.Context, conversationID string, at time.Time) error {
	n, err := r.q.SetAgentSyncedAt(ctx, sqlcgen.SetAgentSyncedAtParams{AgentSyncedAt: at.Unix(), ID: conversationID})
	if err != nil {
		return fmt.Errorf("set agent synced at for conversation %s: %w", conversationID, classifyWriteErr(err))
	}
	if n == 0 {
		return fmt.Errorf("set agent synced at for conversation %s: %w", conversationID, apperrs.ErrNotFound)
	}
	return nil
}

func (r *ChatRepo) GetMessage(ctx context.Context, id string) (*chat.Message, error) {
	row, err := r.q.GetMessage(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", id, notFoundIfNoRows(err))
	}
	return toMessage(row)
}

func (r *ChatRepo) ListMessages(ctx context.Context, conversationID string, limit int) ([]*chat.Message, error) {
	rows, err := r.q.ListMessages(ctx, sqlcgen.ListMessagesParams{ConversationID: conversationID, Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list messages for conversation %s: %w", conversationID, err)
	}
	return toMessages(rows)
}

// ListMessagesSince returns a conversation's non-deleted messages created strictly after since, oldest-first.
func (r *ChatRepo) ListMessagesSince(ctx context.Context, conversationID string, since time.Time) ([]*chat.Message, error) {
	rows, err := r.q.ListMessagesSince(ctx, sqlcgen.ListMessagesSinceParams{ConversationID: conversationID, CreatedAt: since.Unix()})
	if err != nil {
		return nil, fmt.Errorf("list messages since %s for conversation %s: %w", since, conversationID, err)
	}
	return toMessages(rows)
}

func (r *ChatRepo) UpdateMessage(ctx context.Context, id, body string, mentions []chat.Mention, editedAt time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		mentionsJSON, err := json.Marshal(mentions)
		if err != nil {
			return fmt.Errorf("marshal mentions: %w", err)
		}
		n, err := r.q.WithTx(tx).UpdateMessage(ctx, sqlcgen.UpdateMessageParams{
			Body: body, Mentions: string(mentionsJSON),
			EditedAt: sql.NullInt64{Int64: editedAt.Unix(), Valid: true}, UpdatedAt: editedAt.Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("update message %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update message %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

// DeleteMessage soft-deletes: body/mentions cleared, deleted_at stamped, row stays to keep message order.
func (r *ChatRepo) DeleteMessage(ctx context.Context, id string, deletedAt time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteMessage(ctx, sqlcgen.DeleteMessageParams{
			DeletedAt: sql.NullInt64{Int64: deletedAt.Unix(), Valid: true}, UpdatedAt: deletedAt.Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("delete message %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("delete message %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).MarkRead(ctx, sqlcgen.MarkReadParams{UserID: userID, ConversationID: conversationID, LastReadAt: at.Unix()})
		if err != nil {
			return fmt.Errorf("mark conversation %s read for user %s: %w", conversationID, userID, classifyWriteErr(err))
		}
		return nil
	})
}

// UnreadCounts counts messages newer than the read cursor per conversation; every conversation is present, even at 0.
func (r *ChatRepo) UnreadCounts(ctx context.Context, workspaceID, userID string) ([]chat.UnreadCount, error) {
	rows, err := r.q.UnreadCounts(ctx, sqlcgen.UnreadCountsParams{UserID: userID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, fmt.Errorf("unread counts for user %s: %w", userID, err)
	}
	out := make([]chat.UnreadCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, chat.UnreadCount{ConversationID: row.ID, Kind: chat.Kind(row.Kind), DocID: row.DocID.String, Count: int(row.Count)})
	}
	return out, nil
}

func toConversation(row sqlcgen.Conversation) *chat.Conversation {
	return &chat.Conversation{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Kind: chat.Kind(row.Kind), Name: row.Name,
		TicketID: row.TicketID.String, DocID: row.DocID.String, ParentMessageID: row.ParentMessageID, CreatedBy: row.CreatedBy,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		AgentThreadID: row.AgentThreadID, AgentSyncedAt: time.Unix(row.AgentSyncedAt, 0).UTC(),
	}
}

func toConversations(rows []sqlcgen.Conversation) []*chat.Conversation {
	out := make([]*chat.Conversation, 0, len(rows))
	for _, row := range rows {
		out = append(out, toConversation(row))
	}
	return out
}

func toMessage(row sqlcgen.Message) (*chat.Message, error) {
	m := &chat.Message{
		ID: row.ID, ConversationID: row.ConversationID, AuthorID: row.AuthorID, AuthorKind: chat.AuthorKind(row.AuthorKind),
		Body: row.Body, AttachmentID: row.AttachmentID.String,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.Mentions), &m.Mentions); err != nil {
		return nil, fmt.Errorf("unmarshal mentions for message %s: %w", m.ID, err)
	}
	if row.EditedAt.Valid {
		m.EditedAt = atPtr(time.Unix(row.EditedAt.Int64, 0).UTC())
	}
	if row.DeletedAt.Valid {
		m.DeletedAt = atPtr(time.Unix(row.DeletedAt.Int64, 0).UTC())
	}
	return m, nil
}

func toMessages(rows []sqlcgen.Message) ([]*chat.Message, error) {
	out := make([]*chat.Message, 0, len(rows))
	for _, row := range rows {
		m, err := toMessage(row)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func enqueueChatOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}

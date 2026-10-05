package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
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
			ProjectID:       sql.NullString{String: c.ProjectID, Valid: c.ProjectID != ""},
			ParentMessageID: c.ParentMessageID, CreatedBy: c.CreatedBy,
			CreatedAt: c.CreatedAt.Unix(), UpdatedAt: c.UpdatedAt.Unix(), IsGeneral: boolToInt(c.General), Private: boolToInt(c.Private),
		})
		if err != nil {
			return fmt.Errorf("insert conversation %s: %w", c.ID, classifyWriteErr(err))
		}
		if err := insertParticipants(ctx, q, c.ID, participantIDs, c.CreatedAt); err != nil {
			return err
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func insertParticipants(ctx context.Context, q *sqlcgen.Queries, conversationID string, userIDs []string, at time.Time) error {
	for _, uid := range userIDs {
		if err := q.InsertConversationParticipant(ctx, sqlcgen.InsertConversationParticipantParams{
			ConversationID: conversationID, UserID: uid, CreatedAt: at.Unix(),
		}); err != nil {
			return fmt.Errorf("add participant %s to conversation %s: %w", uid, conversationID, classifyWriteErr(err))
		}
	}
	return nil
}

// SetChannelPrivate drops every participant row first, so a public channel's leftover creator row never makes a member.
func (r *ChatRepo) SetChannelPrivate(ctx context.Context, id string, private bool, memberIDs []string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.SetConversationPrivate(ctx, sqlcgen.SetConversationPrivateParams{Private: boolToInt(private), UpdatedAt: at.Unix(), ID: id})
		if err != nil {
			return fmt.Errorf("set conversation %s private: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("set conversation %s private: %w", id, apperrs.ErrNotFound)
		}
		if err := q.DeleteConversationParticipants(ctx, id); err != nil {
			return fmt.Errorf("clear participants of conversation %s: %w", id, err)
		}
		if err := insertParticipants(ctx, q, id, memberIDs, at); err != nil {
			return err
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) AddParticipants(ctx context.Context, id string, userIDs []string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := insertParticipants(ctx, r.q.WithTx(tx), id, userIDs, at); err != nil {
			return err
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) RemoveParticipants(ctx context.Context, id string, userIDs []string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, uid := range userIDs {
			if err := q.DeleteConversationParticipant(ctx, sqlcgen.DeleteConversationParticipantParams{ConversationID: id, UserID: uid}); err != nil {
				return fmt.Errorf("remove participant %s from conversation %s: %w", uid, id, err)
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
	c := toConversation(row)
	if err := r.attachParticipants(ctx, []*chat.Conversation{c}); err != nil {
		return nil, fmt.Errorf("attach participants: %w", err)
	}
	return c, nil
}

// RenameConversation trips the channel-name unique index on a duplicate, the same conflict create reports.
func (r *ChatRepo) RenameConversation(ctx context.Context, id, name string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RenameConversation(ctx, sqlcgen.RenameConversationParams{Name: name, UpdatedAt: at.Unix(), ID: id})
		if err != nil {
			return fmt.Errorf("rename conversation %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("rename conversation %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

// DeleteConversation relies on ON DELETE CASCADE for messages, participants, read state, and attachments.
func (r *ChatRepo) DeleteConversation(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteConversation(ctx, id)
		if err != nil {
			return fmt.Errorf("delete conversation %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete conversation %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
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

func (r *ChatRepo) GetInterviewThread(ctx context.Context, projectID string) (*chat.Conversation, error) {
	row, err := r.q.GetInterviewThread(ctx, nullString(projectID))
	if err != nil {
		return nil, fmt.Errorf("get interview thread for project %s: %w", projectID, notFoundIfNoRows(err))
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
	if err := r.attachParticipants(ctx, out); err != nil {
		return nil, fmt.Errorf("attach participants: %w", err)
	}
	return out, nil
}

// attachParticipants batch-fills ParticipantIDs on every DM and private channel — one query for the page, not one per conversation.
func (r *ChatRepo) attachParticipants(ctx context.Context, cs []*chat.Conversation) error {
	byID := make(map[string]*chat.Conversation, len(cs))
	ids := make([]string, 0, len(cs))
	for _, c := range cs {
		if c.Kind != chat.KindDM && !c.Private {
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
		return fmt.Errorf("list participants: %w", err)
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
		if err := insertMessage(ctx, r.q.WithTx(tx), m); err != nil {
			return err
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func insertMessage(ctx context.Context, q *sqlcgen.Queries, m *chat.Message) error {
	mentionsJSON, err := json.Marshal(m.Mentions)
	if err != nil {
		return fmt.Errorf("marshal mentions: %w", err)
	}
	authorKind := m.AuthorKind
	if authorKind == "" {
		authorKind = chat.AuthorUser
	}
	var handoffs sql.NullString
	if len(m.Handoffs) > 0 {
		raw, err := json.Marshal(m.Handoffs)
		if err != nil {
			return fmt.Errorf("marshal hand-offs: %w", err)
		}
		handoffs = sql.NullString{String: string(raw), Valid: true}
	}
	err = q.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		ID: m.ID, ConversationID: m.ConversationID, AuthorID: m.AuthorID, AuthorKind: string(authorKind),
		Body: m.Body, Mentions: string(mentionsJSON), AttachmentID: sql.NullString{String: m.AttachmentID, Valid: m.AttachmentID != ""},
		Handoffs: handoffs, Via: m.Via, CreatedAt: m.CreatedAt.Unix(), UpdatedAt: m.UpdatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert message %s: %w", m.ID, classifyWriteErr(err))
	}
	return nil
}

// noteContentType is fixed rather than sniffed: the server writes a note's file, and markdown is never served inline.
const noteContentType = "text/markdown; charset=utf-8"

// CreateNote inserts the file before the message, so the message never points at a file that is not there.
func (r *ChatRepo) CreateNote(ctx context.Context, m *chat.Message, file *chat.NoteFile, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.CreateAttachment(ctx, sqlcgen.CreateAttachmentParams{
			ID: file.ID, ConversationID: nullString(m.ConversationID), Name: file.Name, ContentType: noteContentType,
			Size: int64(len(file.Markdown)), UploadedBy: m.AuthorID, CreatedAt: m.CreatedAt.Unix(), Data: []byte(file.Markdown),
		})
		if err != nil {
			return fmt.Errorf("insert note file %s: %w", file.ID, classifyWriteErr(err))
		}
		if err := insertMessage(ctx, q, m); err != nil {
			return err
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) ListNoteFiles(ctx context.Context, attachmentIDs []string) ([]*chat.NoteFile, error) {
	if len(attachmentIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListNoteFiles(ctx, attachmentIDs)
	if err != nil {
		return nil, fmt.Errorf("list note files: %w", err)
	}
	out := make([]*chat.NoteFile, 0, len(rows))
	for _, row := range rows {
		out = append(out, &chat.NoteFile{ID: row.ID, Name: row.Name, Markdown: string(row.Data)})
	}
	return out, nil
}

// ReplaceNoteFile touches the message first, so a note deleted meanwhile refuses the write instead of reviving its file.
func (r *ChatRepo) ReplaceNoteFile(ctx context.Context, m *chat.Message, markdown string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.TouchNoteMessage(ctx, sqlcgen.TouchNoteMessageParams{UpdatedAt: at.Unix(), ID: m.ID, AttachmentID: nullString(m.AttachmentID)})
		if err != nil {
			return fmt.Errorf("touch note %s: %w", m.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("replace note %s: %w", m.ID, apperrs.ErrNotFound)
		}
		n, err = q.ReplaceNoteFileData(ctx, sqlcgen.ReplaceNoteFileDataParams{Data: []byte(markdown), Size: int64(len(markdown)), ID: m.AttachmentID})
		if err != nil {
			return fmt.Errorf("replace note file %s: %w", m.AttachmentID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("replace note file %s: %w", m.AttachmentID, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

// DeleteNote has no foreign key to lean on (messages.attachment_id has none), so it deletes the file by hand.
func (r *ChatRepo) DeleteNote(ctx context.Context, m *chat.Message, imageIDs []string, deletedAt time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.DeleteMessage(ctx, sqlcgen.DeleteMessageParams{
			DeletedAt: sql.NullInt64{Int64: deletedAt.Unix(), Valid: true}, UpdatedAt: deletedAt.Unix(), ID: m.ID,
		})
		if err != nil {
			return fmt.Errorf("delete note %s: %w", m.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("delete note %s: %w", m.ID, apperrs.ErrNotFound)
		}
		if _, err := q.DeleteAttachment(ctx, m.AttachmentID); err != nil {
			return fmt.Errorf("delete note file %s: %w", m.AttachmentID, err)
		}
		if len(imageIDs) > 0 {
			err := q.DeleteNoteImages(ctx, sqlcgen.DeleteNoteImagesParams{ConversationID: nullString(m.ConversationID), Ids: imageIDs})
			if err != nil {
				return fmt.Errorf("delete images of note %s: %w", m.ID, err)
			}
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) IsNoteFile(ctx context.Context, conversationID, attachmentID string) (bool, error) {
	ok, err := r.q.IsNoteFile(ctx, sqlcgen.IsNoteFileParams{ConversationID: conversationID, AttachmentID: nullString(attachmentID)})
	if err != nil {
		return false, fmt.Errorf("is note file %s: %w", attachmentID, err)
	}
	return ok, nil
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
	m, err := toMessage(row)
	if err != nil {
		return nil, err
	}
	if err := r.attachReactions(ctx, []*chat.Message{m}); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *ChatRepo) ListMessages(ctx context.Context, conversationID string, limit int) ([]*chat.Message, error) {
	rows, err := r.q.ListMessages(ctx, sqlcgen.ListMessagesParams{ConversationID: conversationID, Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list messages for conversation %s: %w", conversationID, err)
	}
	// The query takes the newest rows so a long thread keeps its latest messages; callers read them oldest-first.
	slices.Reverse(rows)
	ms, err := toMessages(rows)
	if err != nil {
		return nil, err
	}
	if err := r.attachReactions(ctx, ms); err != nil {
		return nil, err
	}
	return ms, nil
}

// attachReactions batch-fills Reactions on every live message, one query for the page; a deleted message keeps none.
func (r *ChatRepo) attachReactions(ctx context.Context, ms []*chat.Message) error {
	byID := make(map[string]*chat.Message, len(ms))
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		if m.DeletedAt != nil {
			continue
		}
		byID[m.ID] = m
		ids = append(ids, m.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.q.ListMessageReactions(ctx, ids)
	if err != nil {
		return fmt.Errorf("list message reactions: %w", err)
	}
	for _, row := range rows {
		m := byID[row.MessageID]
		i := slices.IndexFunc(m.Reactions, func(re chat.Reaction) bool { return re.Emoji == row.Emoji })
		if i < 0 {
			m.Reactions = append(m.Reactions, chat.Reaction{Emoji: row.Emoji})
			i = len(m.Reactions) - 1
		}
		m.Reactions[i].UserIDs = append(m.Reactions[i].UserIDs, row.UserID)
	}
	return nil
}

func (r *ChatRepo) SetReaction(ctx context.Context, messageID, userID, emoji string, reacted bool, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := writeReaction(ctx, r.q.WithTx(tx), messageID, userID, emoji, reacted, at)
		if err != nil {
			return fmt.Errorf("set reaction %s on message %s: %w", emoji, messageID, classifyWriteErr(err))
		}
		if n == 0 {
			return nil
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
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

func writeReaction(ctx context.Context, q *sqlcgen.Queries, messageID, userID, emoji string, reacted bool, at time.Time) (int64, error) {
	if !reacted {
		return q.DeleteMessageReaction(ctx, sqlcgen.DeleteMessageReactionParams{MessageID: messageID, Emoji: emoji, UserID: userID})
	}
	return q.InsertMessageReaction(ctx, sqlcgen.InsertMessageReactionParams{MessageID: messageID, Emoji: emoji, UserID: userID, CreatedAt: at.Unix()})
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
		out = append(out, chat.UnreadCount{ConversationID: row.ID, Count: int(row.Count)})
	}
	return out, nil
}

func toConversation(row sqlcgen.Conversation) *chat.Conversation {
	return &chat.Conversation{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Kind: chat.Kind(row.Kind), Name: row.Name,
		TicketID: row.TicketID.String, DocID: row.DocID.String, ProjectID: row.ProjectID.String, ParentMessageID: row.ParentMessageID, CreatedBy: row.CreatedBy,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
		AgentThreadID: row.AgentThreadID, AgentSyncedAt: time.Unix(row.AgentSyncedAt, 0).UTC(), General: row.IsGeneral != 0,
		Private: row.Private != 0,
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
		Body: row.Body, AttachmentID: row.AttachmentID.String, Via: row.Via,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.Mentions), &m.Mentions); err != nil {
		return nil, fmt.Errorf("unmarshal mentions for message %s: %w", m.ID, err)
	}
	if row.Handoffs.Valid {
		if err := json.Unmarshal([]byte(row.Handoffs.String), &m.Handoffs); err != nil {
			return nil, fmt.Errorf("unmarshal hand-offs for message %s: %w", m.ID, err)
		}
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

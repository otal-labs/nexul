package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestAttachment(id string, owner attachments.Owner) *attachments.Attachment {
	return &attachments.Attachment{
		ID: id, DocID: owner.DocID, TicketID: owner.TicketID, ConversationID: owner.ConversationID, Name: "shot.png", ContentType: "image/png",
		Size: 4, Data: []byte("\x89PNG"), UploadedBy: "user-1", CreatedAt: time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
	}
}

func TestAttachmentsRepo_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-1", "doc-1")))
	want := newTestAttachment("a-1", attachments.Owner{DocID: "doc-1"})
	require.NoError(t, s.Attachments.Create(ctx, want))
	require.NoError(t, s.Attachments.Create(ctx, newTestAttachment("a-2", attachments.Owner{TicketID: "t-1"})))

	got, err := s.Attachments.GetByID(ctx, "a-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	docList, err := s.Attachments.ListByOwner(ctx, attachments.Owner{DocID: "doc-1"})
	require.NoError(t, err)
	require.Len(t, docList, 1)
	assert.Equal(t, "a-1", docList[0].ID)
	assert.Nil(t, docList[0].Data)

	ticketList, err := s.Attachments.ListByOwner(ctx, attachments.Owner{TicketID: "t-1"})
	require.NoError(t, err)
	require.Len(t, ticketList, 1)
	assert.Equal(t, "a-2", ticketList[0].ID)
	assert.Equal(t, "t-1", ticketList[0].TicketID)

	empty, err := s.Attachments.ListByOwner(ctx, attachments.Owner{})
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestAttachmentsRepo_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Attachments.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.Attachments.Delete(context.Background(), "missing"), apperrs.ErrNotFound)
}

func TestAttachmentsRepo_Constraints(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-1", "")))

	t.Run("unknown owner is a conflict", func(t *testing.T) {
		err := s.Attachments.Create(ctx, newTestAttachment("a-x", attachments.Owner{TicketID: "nope"}))
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
	t.Run("no owner is a conflict", func(t *testing.T) {
		err := s.Attachments.Create(ctx, newTestAttachment("a-y", attachments.Owner{}))
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
	t.Run("two owners is a conflict", func(t *testing.T) {
		// Both doc-1 and t-1 are real rows, so this isolates the CHECK from any FK failure.
		err := s.Attachments.Create(ctx, newTestAttachment("a-z", attachments.Owner{DocID: "doc-1", TicketID: "t-1"}))
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
	t.Run("duplicate id is a conflict", func(t *testing.T) {
		require.NoError(t, s.Attachments.Create(ctx, newTestAttachment("a-1", attachments.Owner{DocID: "doc-1"})))
		err := s.Attachments.Create(ctx, newTestAttachment("a-1", attachments.Owner{DocID: "doc-1"}))
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
}

func TestAttachmentsRepo_DeleteAndCascade(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-1", "")))
	require.NoError(t, s.Attachments.Create(ctx, newTestAttachment("a-doc", attachments.Owner{DocID: "doc-1"})))
	require.NoError(t, s.Attachments.Create(ctx, newTestAttachment("a-ticket", attachments.Owner{TicketID: "t-1"})))
	require.NoError(t, s.Attachments.Create(ctx, newTestAttachment("a-gone", attachments.Owner{TicketID: "t-1"})))

	require.NoError(t, s.Attachments.Delete(ctx, "a-gone"))
	_, err := s.Attachments.GetByID(ctx, "a-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	require.NoError(t, s.Docs.Delete(ctx, "doc-1"))
	_, err = s.Attachments.GetByID(ctx, "a-doc")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "doc delete cascades to its attachments")

	require.NoError(t, s.Tickets.Delete(ctx, "t-1"))
	_, err = s.Attachments.GetByID(ctx, "a-ticket")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "ticket delete cascades to its attachments")
}

func TestAttachmentsRepo_ConversationOwner_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(ctx, newTestConversation("conv-1", chat.KindChannel, "eng", "", "u-1"), []string{"u-1"}))
	want := newTestAttachment("a-1", attachments.Owner{ConversationID: "conv-1"})
	require.NoError(t, s.Attachments.Create(ctx, want))

	got, err := s.Attachments.GetByID(ctx, "a-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	list, err := s.Attachments.ListByOwner(ctx, attachments.Owner{ConversationID: "conv-1"})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "a-1", list[0].ID)
	assert.Equal(t, "conv-1", list[0].ConversationID)
}

func TestAttachmentsRepo_ConversationOwner_DeleteCascades(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(ctx, newTestConversation("conv-1", chat.KindChannel, "eng", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Attachments.Create(ctx, newTestAttachment("a-1", attachments.Owner{ConversationID: "conv-1"})))

	_, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, "conv-1")
	require.NoError(t, err)

	_, err = s.Attachments.GetByID(ctx, "a-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "conversation delete cascades to its attachments")
}

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/attachments"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ attachments.Repo = (*AttachmentsRepo)(nil)

type AttachmentsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AttachmentsRepo) Create(ctx context.Context, a *attachments.Attachment) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateAttachment(ctx, sqlcgen.CreateAttachmentParams{
			ID:             a.ID,
			DocID:          sql.NullString{String: a.DocID, Valid: a.DocID != ""},
			TicketID:       sql.NullString{String: a.TicketID, Valid: a.TicketID != ""},
			ConversationID: sql.NullString{String: a.ConversationID, Valid: a.ConversationID != ""},
			MemoryID:       sql.NullString{String: a.MemoryID, Valid: a.MemoryID != ""},
			Name:           a.Name,
			ContentType:    a.ContentType,
			Size:           a.Size,
			UploadedBy:     a.UploadedBy,
			CreatedAt:      a.CreatedAt.Unix(),
			Data:           a.Data,
		})
		if err != nil {
			return fmt.Errorf("insert attachment %s: %w", a.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *AttachmentsRepo) GetByID(ctx context.Context, id string) (*attachments.Attachment, error) {
	row, err := r.q.GetAttachment(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get attachment %s: %w", id, notFoundIfNoRows(err))
	}
	a := toAttachmentMeta(row.ID, row.DocID, row.TicketID, row.ConversationID, row.MemoryID, row.Name, row.ContentType, row.Size, row.UploadedBy, row.CreatedAt)
	a.Data = row.Data
	return a, nil
}

// ListByOwner leaves Data unloaded: the listing never needs the bytes, so the query skips the blob column.
func (r *AttachmentsRepo) ListByOwner(ctx context.Context, owner attachments.Owner) ([]*attachments.Attachment, error) {
	rows, err := r.q.ListAttachmentsByOwner(ctx, sqlcgen.ListAttachmentsByOwnerParams{
		DocID:          sql.NullString{String: owner.DocID, Valid: owner.DocID != ""},
		TicketID:       sql.NullString{String: owner.TicketID, Valid: owner.TicketID != ""},
		ConversationID: sql.NullString{String: owner.ConversationID, Valid: owner.ConversationID != ""},
		MemoryID:       sql.NullString{String: owner.MemoryID, Valid: owner.MemoryID != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	var out []*attachments.Attachment
	for _, row := range rows {
		out = append(out, toAttachmentMeta(row.ID, row.DocID, row.TicketID, row.ConversationID, row.MemoryID, row.Name, row.ContentType, row.Size, row.UploadedBy, row.CreatedAt))
	}
	return out, nil
}

func (r *AttachmentsRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteAttachment(ctx, id)
		if err != nil {
			return fmt.Errorf("delete attachment %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete attachment %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toAttachmentMeta(id string, docID, ticketID, conversationID, memoryID sql.NullString, name, contentType string, size int64, uploadedBy string, createdAt int64) *attachments.Attachment {
	return &attachments.Attachment{
		ID:             id,
		DocID:          docID.String,
		TicketID:       ticketID.String,
		ConversationID: conversationID.String,
		MemoryID:       memoryID.String,
		Name:           name,
		ContentType:    contentType,
		Size:           size,
		UploadedBy:     uploadedBy,
		CreatedAt:      time.Unix(createdAt, 0).UTC(),
	}
}

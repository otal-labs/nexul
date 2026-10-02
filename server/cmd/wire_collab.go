package main

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// collabDocWriter lets collab writes inherit docs validation without collab importing docs.
type collabDocWriter struct {
	docs *docs.Service
}

func (w collabDocWriter) CommitCollab(ctx context.Context, docID, title, body string) error {
	return w.docs.CommitCollab(ctx, docID, title, body)
}

func (w collabDocWriter) Locked(ctx context.Context, docID string) (bool, error) {
	return w.docs.Locked(ctx, docID)
}

// collabNoteRooms is the notes hub's join check and commit target on chat, keyed on a note's message id (ADR 0110).
type collabNoteRooms struct {
	chat *chat.Service
}

func (n collabNoteRooms) Can(ctx context.Context, userID, messageID string, action permissions.Action) (bool, error) {
	return n.chat.CanJoinNote(ctx, userID, messageID, action)
}

// CommitCollab ignores the title: a note's file is markdown only, converted from the editor's tree.
func (n collabNoteRooms) CommitCollab(ctx context.Context, messageID, _, body string) error {
	markdown, err := richtext.ToMarkdown(body)
	if err != nil {
		return fmt.Errorf("%w: note %s body is not valid document content", apperrs.ErrInvalid, messageID)
	}
	return n.chat.CommitNote(ctx, messageID, markdown)
}

func (n collabNoteRooms) Locked(ctx context.Context, messageID string) (bool, error) {
	return n.chat.NoteLocked(ctx, messageID)
}

package chat

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// maxNoteFileBytes matches the attachment cap (ADR 0027), since a note's file is stored as an attachment.
const maxNoteFileBytes = 10 << 20

const maxNoteNameLen = 252

// PostNote posts a note: an Agent message on a ticket's thread, on callerID's behalf, carrying a markdown file
// (ADR 0108). It takes tickets:write on the ticket, and starts no turn, since only a person's message does.
func (s *Service) PostNote(ctx context.Context, conversationID, callerID, body string, in NoteFileInput) (*Message, *NoteFile, error) {
	m, err := s.newMessage(conversationID, callerID, body, AuthorAgent)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.noteThread(ctx, m.ConversationID, m.AuthorID)
	if err != nil {
		return nil, nil, err
	}
	file, err := newNoteFile(in)
	if err != nil {
		return nil, nil, err
	}
	m.AttachmentID = file.ID
	if err := s.repo.CreateNote(ctx, m, file, messageCreated(m, c)); err != nil {
		return nil, nil, fmt.Errorf("post note to conversation %s: %w", c.ID, err)
	}
	return m, file, nil
}

// noteThread checks, in order, that callerID reads the conversation, that it is a ticket's thread, and tickets:write.
func (s *Service) noteThread(ctx context.Context, conversationID, callerID string) (*Conversation, error) {
	return s.noteThreadFor(ctx, conversationID, callerID, permissions.TicketsWrite)
}

// noteThreadFor is noteThread with the ticket action to require.
func (s *Service) noteThreadFor(ctx context.Context, conversationID, callerID string, action permissions.Action) (*Conversation, error) {
	c, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", conversationID, err)
	}
	if err := s.requireRead(ctx, c, callerID); err != nil {
		return nil, err
	}
	if c.Kind != KindTicketThread {
		return nil, fmt.Errorf("%w: only a ticket's thread takes a note's file, not a %s", apperrs.ErrInvalid, c.Kind)
	}
	if s.threads == nil {
		return c, s.require(ctx, c.WorkspaceID, action)
	}
	return c, s.threads.RequireTicket(ctx, c.TicketID, action)
}

func newNoteFile(in NoteFileInput) (*NoteFile, error) {
	if strings.TrimSpace(in.Markdown) == "" {
		return nil, fmt.Errorf("%w: a note's file needs its markdown", apperrs.ErrInvalid)
	}
	if err := noteFileFits(in.Markdown); err != nil {
		return nil, err
	}
	return &NoteFile{ID: ids.New(), Name: noteFileName(in.Name), Markdown: in.Markdown}, nil
}

func noteFileFits(markdown string) error {
	if len(markdown) > maxNoteFileBytes {
		return fmt.Errorf("%w: a note's file exceeds %d bytes", apperrs.ErrInvalid, maxNoteFileBytes)
	}
	return nil
}

// ReplaceNote replaces a note's file from outside its live room under tickets:write; open editors reload from it.
func (s *Service) ReplaceNote(ctx context.Context, messageID, callerID, markdown string) (*Message, *NoteFile, error) {
	if strings.TrimSpace(markdown) == "" {
		return nil, nil, fmt.Errorf("%w: a note's file needs its markdown", apperrs.ErrInvalid)
	}
	if err := noteFileFits(markdown); err != nil {
		return nil, nil, err
	}
	m, err := s.repo.GetMessage(ctx, strings.TrimSpace(messageID))
	if err != nil {
		return nil, nil, fmt.Errorf("replace note %s: %w", messageID, err)
	}
	c, err := s.noteThread(ctx, m.ConversationID, callerID)
	if err != nil {
		return nil, nil, err
	}
	if m.DeletedAt != nil {
		return nil, nil, fmt.Errorf("%w: message %s was deleted", apperrs.ErrNotFound, m.ID)
	}
	if m.AttachmentID == "" {
		return nil, nil, fmt.Errorf("%w: message %s is not a note, so it has no file to replace", apperrs.ErrInvalid, m.ID)
	}
	files, err := s.repo.ListNoteFiles(ctx, []string{m.AttachmentID})
	if err != nil {
		return nil, nil, fmt.Errorf("replace note %s: %w", m.ID, err)
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("replace note %s: %w", m.ID, apperrs.ErrNotFound)
	}
	write := func(ctx context.Context) error { return s.writeNote(ctx, m, c.MembersOnly(), markdown) }
	if err := s.fenceNote(ctx, m.ID, write); err != nil {
		return nil, nil, err
	}
	return m, &NoteFile{ID: files[0].ID, Name: files[0].Name, Markdown: markdown}, nil
}

// fenceNote runs write through the note's live room when one is wired, so open editors cannot overwrite it.
func (s *Service) fenceNote(ctx context.Context, messageID string, write func(context.Context) error) error {
	if s.noteLive == nil {
		return write(ctx)
	}
	return s.noteLive.Reset(ctx, messageID, write)
}

// CommitNote writes a note's live room into its file; joining the room already took tickets:write.
func (s *Service) CommitNote(ctx context.Context, messageID, markdown string) error {
	if err := noteFileFits(markdown); err != nil {
		return err
	}
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("commit note %s: %w", messageID, err)
	}
	if m.DeletedAt != nil || m.AttachmentID == "" {
		return fmt.Errorf("%w: note %s was deleted", apperrs.ErrConflict, messageID)
	}
	membersOnly, err := s.MembersOnly(ctx, m.ConversationID)
	if err != nil {
		return fmt.Errorf("commit note %s: %w", messageID, err)
	}
	return s.writeNote(ctx, m, membersOnly, markdown)
}

// writeNote replaces m's file and publishes chat.message.updated in the same transaction.
func (s *Service) writeNote(ctx context.Context, m *Message, membersOnly bool, markdown string) error {
	m.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageUpdated, Payload: MessageUpdatedEvent{Message: *m, MembersOnly: membersOnly}}
	if err := s.repo.ReplaceNoteFile(ctx, m, markdown, m.UpdatedAt, evt); err != nil {
		return fmt.Errorf("replace note %s: %w", m.ID, err)
	}
	return nil
}

// NoteLocked reports a message whose file refuses live edits: one that is deleted or is not a note.
func (s *Service) NoteLocked(ctx context.Context, messageID string) (bool, error) {
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return false, err
	}
	return m.DeletedAt != nil || m.AttachmentID == "", nil
}

// CanJoinNote reports whether userID may join a note's live room under action, checked on the note's ticket.
func (s *Service) CanJoinNote(ctx context.Context, userID, messageID string, action permissions.Action) (bool, error) {
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return false, err
	}
	if m.DeletedAt != nil || m.AttachmentID == "" {
		return false, nil
	}
	if _, err := s.noteThreadFor(ctx, m.ConversationID, userID, action); err != nil {
		return false, err
	}
	return true, nil
}

// noteFileName keeps the base name, so a path never reaches storage, and makes sure it ends in .md.
func noteFileName(name string) string {
	name = path.Base(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")))
	if name == "." || name == "/" {
		name = "note"
	}
	if strings.EqualFold(path.Ext(name), ".md") {
		name = strings.TrimSuffix(name, path.Ext(name))
	}
	if len(name) > maxNoteNameLen {
		name = name[:maxNoteNameLen]
	}
	return name + ".md"
}

// NoteFiles returns the files of the notes among ms, by attachment id; ms come from a call that already checked the read.
func (s *Service) NoteFiles(ctx context.Context, ms []*Message) (map[string]*NoteFile, error) {
	var attachmentIDs []string
	for _, m := range ms {
		if m.AttachmentID != "" {
			attachmentIDs = append(attachmentIDs, m.AttachmentID)
		}
	}
	files, err := s.repo.ListNoteFiles(ctx, attachmentIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*NoteFile, len(files))
	for _, f := range files {
		out[f.ID] = f
	}
	return out, nil
}

// IsNoteFile reports whether a conversation's file is a note's, which goes only with its message; the server's own seam.
func (s *Service) IsNoteFile(ctx context.Context, conversationID, attachmentID string) (bool, error) {
	return s.repo.IsNoteFile(ctx, conversationID, attachmentID)
}

// noteImageRef finds the attachment ids a note's markdown points at, the images pasted into it.
var noteImageRef = regexp.MustCompile(`/api/attachments/([\w-]+)`)

// deleteNote takes tickets:write rather than authorship, and takes the note's file and its images with it.
func (s *Service) deleteNote(ctx context.Context, m *Message, callerID string) error {
	c, err := s.noteThread(ctx, m.ConversationID, callerID)
	if err != nil {
		return err
	}
	files, err := s.repo.ListNoteFiles(ctx, []string{m.AttachmentID})
	if err != nil {
		return fmt.Errorf("delete note %s: %w", m.ID, err)
	}
	var images []string
	for _, f := range files {
		for _, ref := range noteImageRef.FindAllStringSubmatch(f.Markdown, -1) {
			images = append(images, ref[1])
		}
	}
	now := s.now().UTC()
	if err := s.repo.DeleteNote(ctx, m, slices.Compact(slices.Sorted(slices.Values(images))), now, messageDeleted(m, c, now)); err != nil {
		return fmt.Errorf("delete note %s: %w", m.ID, err)
	}
	return nil
}

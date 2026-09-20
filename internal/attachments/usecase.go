package attachments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// MaxSize caps one upload; ponytail: bytes live in SQLite, move to object storage before raising this.
const MaxSize = 10 << 20

const maxNameLen = 255

// AccessChecker lets attachments check inherited doc read/edit bits.
type AccessChecker interface {
	Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error)
}

// MemoryAccessChecker lets attachments check the memories domain's workspace-scoped permission bits for
// memory-owned files (ADR 0017: attachments never imports memories); memoryID resolves to a workspace on
// the other side of the gate.
type MemoryAccessChecker interface {
	Can(ctx context.Context, userID, memoryID string, action permissions.Action) (bool, error)
}

// Service runs permission checks so HTTP and any later MCP adapter inherit them (ADR 0019).
type Service struct {
	repo         Repo
	access       AccessChecker
	memoryAccess MemoryAccessChecker
	now          func() time.Time
}

// NewService wires the attachments use-cases over the given repo, doc access checker, and memory access checker.
func NewService(repo Repo, access AccessChecker, memoryAccess MemoryAccessChecker) *Service {
	return &Service{repo: repo, access: access, memoryAccess: memoryAccess, now: time.Now}
}

// Upload sniffs content type from the bytes, never trusting the client-provided type.
func (s *Service) Upload(ctx context.Context, owner Owner, name string, data []byte) (*Attachment, error) {
	actor, err := s.requireOwner(ctx, owner, permissions.DocsWrite, permissions.MemoriesWrite)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: file is empty", apperrs.ErrInvalid)
	}
	if len(data) > MaxSize {
		return nil, fmt.Errorf("%w: file exceeds %d bytes", apperrs.ErrInvalid, MaxSize)
	}
	a := &Attachment{
		ID:             ids.New(),
		DocID:          owner.DocID,
		TicketID:       owner.TicketID,
		ConversationID: owner.ConversationID,
		MemoryID:       owner.MemoryID,
		Name:           cleanName(name),
		ContentType:    http.DetectContentType(data),
		Size:           int64(len(data)),
		UploadedBy:     actor.ID,
		CreatedAt:      s.now().UTC(),
		Data:           data,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		// The owner columns are FKs, so an unknown doc/ticket/conversation/memory surfaces as a constraint failure.
		if errors.Is(err, apperrs.ErrConflict) {
			return nil, fmt.Errorf("%w: doc, ticket, conversation, or memory does not exist", apperrs.ErrNotFound)
		}
		return nil, fmt.Errorf("create attachment: %w", err)
	}
	return a, nil
}

// Get returns an attachment with its bytes; requires read on the owning doc or memory.
func (s *Service) Get(ctx context.Context, id string) (*Attachment, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get attachment %s: %w", id, err)
	}
	if _, err := s.requireOwner(ctx, a.Owner(), permissions.DocsRead, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	return a, nil
}

// List returns an owner's attachments without bytes; requires read on the owning doc or memory.
func (s *Service) List(ctx context.Context, owner Owner) ([]*Attachment, error) {
	if _, err := s.requireOwner(ctx, owner, permissions.DocsRead, permissions.MemoriesRead); err != nil {
		return nil, err
	}
	as, err := s.repo.ListByOwner(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	return as, nil
}

// Delete leaves any body still referencing this attachment with a dangling image.
func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete attachment %s: %w", id, err)
	}
	if _, err := s.requireOwner(ctx, a.Owner(), permissions.DocsWrite, permissions.MemoriesWrite); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete attachment %s: %w", id, err)
	}
	return nil
}

// requireOwner needs only an authenticated user for ticket/conversation owners (workspace/chat-wide perms).
// docAction gates a doc owner via the per-doc AccessChecker; memoryAction gates a memory owner via the
// workspace-scoped MemoryAccessChecker (memories has no per-resource overwrite grid, unlike docs).
func (s *Service) requireOwner(ctx context.Context, owner Owner, docAction, memoryAction permissions.Action) (identity.Actor, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return identity.Actor{}, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	set := 0
	for _, id := range []string{owner.DocID, owner.TicketID, owner.ConversationID, owner.MemoryID} {
		if id != "" {
			set++
		}
	}
	if set != 1 {
		return identity.Actor{}, fmt.Errorf("%w: exactly one of doc_id, ticket_id, conversation_id, or memory_id is required", apperrs.ErrInvalid)
	}
	if owner.MemoryID != "" {
		// Fail closed: a service without a wired memory access checker must deny, never silently allow.
		if s.memoryAccess == nil {
			return identity.Actor{}, fmt.Errorf("%w: no %s permission on memory %s", apperrs.ErrForbidden, memoryAction, owner.MemoryID)
		}
		allowed, err := s.memoryAccess.Can(ctx, actor.ID, owner.MemoryID, memoryAction)
		if err != nil || !allowed {
			return identity.Actor{}, fmt.Errorf("%w: no %s permission on memory %s", apperrs.ErrForbidden, memoryAction, owner.MemoryID)
		}
		return actor, nil
	}
	if owner.DocID == "" {
		return actor, nil
	}
	// Fail closed: a service without a wired access checker must deny, never silently allow.
	if s.access == nil {
		return identity.Actor{}, fmt.Errorf("%w: no %s permission on doc %s", apperrs.ErrForbidden, docAction, owner.DocID)
	}
	allowed, err := s.access.Can(ctx, actor.ID, owner.DocID, docAction)
	if err != nil || !allowed {
		return identity.Actor{}, fmt.Errorf("%w: no %s permission on doc %s", apperrs.ErrForbidden, docAction, owner.DocID)
	}
	return actor, nil
}

// ListOwnerAttachmentIDs returns an owner's attachment ids with no permission check; a trusted cross-domain
// call used by memories Clone (ADR 0017), which has already authorized the clone itself.
func (s *Service) ListOwnerAttachmentIDs(ctx context.Context, owner Owner) ([]string, error) {
	as, err := s.repo.ListByOwner(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("list attachment ids: %w", err)
	}
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.ID
	}
	return out, nil
}

// CopyAttachmentsWithIDs duplicates from's attachments as new rows owned by to, under the caller-assigned
// ids in idMap (so a cloned body's references can be rewritten before the clone row exists). Trusted, no
// permission check: the caller (memories Clone) has already authorized the action.
func (s *Service) CopyAttachmentsWithIDs(ctx context.Context, from, to Owner, idMap map[string]string) error {
	as, err := s.repo.ListByOwner(ctx, from)
	if err != nil {
		return fmt.Errorf("list attachments to copy: %w", err)
	}
	for _, a := range as {
		newID, ok := idMap[a.ID]
		if !ok {
			continue
		}
		full, err := s.repo.GetByID(ctx, a.ID)
		if err != nil {
			return fmt.Errorf("get attachment %s to copy: %w", a.ID, err)
		}
		clone := &Attachment{
			ID:             newID,
			DocID:          to.DocID,
			TicketID:       to.TicketID,
			ConversationID: to.ConversationID,
			MemoryID:       to.MemoryID,
			Name:           full.Name,
			ContentType:    full.ContentType,
			Size:           full.Size,
			UploadedBy:     full.UploadedBy,
			CreatedAt:      s.now().UTC(),
			Data:           full.Data,
		}
		if err := s.repo.Create(ctx, clone); err != nil {
			return fmt.Errorf("copy attachment %s: %w", a.ID, err)
		}
	}
	return nil
}

func cleanName(name string) string {
	name = path.Base(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")))
	if name == "." || name == "/" || name == "" {
		name = "file"
	}
	if len(name) > maxNameLen {
		name = name[len(name)-maxNameLen:]
	}
	return name
}

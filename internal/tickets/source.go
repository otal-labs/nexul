package tickets

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SourceDocs checks a source doc exists and the caller may read it, without importing docs (ADR 0017).
type SourceDocs interface {
	RequireReadable(ctx context.Context, docID string) error
}

// SetSourceDocs wires the source doc check; unset, any doc id is accepted.
func (s *Service) SetSourceDocs(d SourceDocs) { s.docs = d }

func (s *Service) requireSource(ctx context.Context, docID string) error {
	if docID == "" || s.docs == nil {
		return nil
	}
	if err := s.docs.RequireReadable(ctx, docID); err != nil {
		return fmt.Errorf("source doc %s: %w", docID, err)
	}
	return nil
}

// SetSource sets the doc a ticket was derived from; an empty docID clears it.
func (s *Service) SetSource(ctx context.Context, id, docID string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	docID = strings.TrimSpace(docID)
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("set source on ticket %s: %w", id, err)
	}
	if current.DocID == docID {
		return current, nil
	}
	if err := s.requireSource(ctx, docID); err != nil {
		return nil, fmt.Errorf("set source on ticket %s: %w", id, err)
	}
	updated := *current
	updated.DocID = docID
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: UpdatedEvent{Ticket: updated, ActorID: statusActor(ctx).UserID}}
	if err := s.repo.UpdateDoc(ctx, current.ID, docID, evt); err != nil {
		return nil, fmt.Errorf("set source on ticket %s: %w", id, err)
	}
	return &updated, nil
}

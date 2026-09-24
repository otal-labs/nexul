package tickets

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Links returns both directions of a ticket's found-in and blocked-by links.
func (s *Service) Links(ctx context.Context, id string) (*LinkSet, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, fmt.Errorf("list links for ticket %s: %w", id, err)
	}
	from, to, err := s.repo.ListLinkEnds(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list links for ticket %s: %w", id, err)
	}
	return buildLinkSet(from, to), nil
}

func buildLinkSet(from, to []LinkEnd) *LinkSet {
	set := &LinkSet{BugsFound: []LinkedTicket{}, BlockedBy: []LinkedTicket{}, Blocks: []LinkedTicket{}}
	for _, e := range from {
		if e.Kind == LinkFoundIn {
			set.FoundIn = e.Ticket
			set.OriginUnknown = e.Ticket == nil
			continue
		}
		if e.Kind == LinkBlockedBy && e.Ticket != nil {
			set.BlockedBy = append(set.BlockedBy, *e.Ticket)
			set.Blocked = set.Blocked || !e.Ticket.Done
		}
	}
	for _, e := range to {
		if e.Ticket == nil {
			continue
		}
		if e.Kind == LinkFoundIn {
			set.BugsFound = append(set.BugsFound, *e.Ticket)
			continue
		}
		if e.Kind == LinkBlockedBy {
			set.Blocks = append(set.Blocks, *e.Ticket)
		}
	}
	return set
}

// SetFoundIn records the ticket a bug was found in, or marks its origin unknown; it replaces any earlier found-in.
func (s *Service) SetFoundIn(ctx context.Context, id, originID string, originUnknown bool) (*LinkSet, error) {
	id, originID = strings.TrimSpace(id), strings.TrimSpace(originID)
	if err := validateFoundIn(id, originID, originUnknown); err != nil {
		return nil, err
	}
	if err := s.mustExist(ctx, id, originID); err != nil {
		return nil, fmt.Errorf("set found-in on ticket %s: %w", id, err)
	}
	current, held, err := s.foundIn(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("set found-in on ticket %s: %w", id, err)
	}
	if held && current == originID {
		return s.Links(ctx, id)
	}
	link := TicketLink{TicketID: id, Kind: LinkFoundIn, TargetID: originID, CreatedAt: s.now().UTC()}
	var evts []eventbus.OutboxEvent
	if held {
		evts = append(evts, linkEvent(TopicLinkDeleted, TicketLink{TicketID: id, Kind: LinkFoundIn, TargetID: current, CreatedAt: link.CreatedAt}))
	}
	evts = append(evts, linkEvent(TopicLinkCreated, link))
	if err := s.repo.PutLink(ctx, link, evts...); err != nil {
		return nil, fmt.Errorf("set found-in on ticket %s: %w", id, err)
	}
	return s.Links(ctx, id)
}

func validateFoundIn(id, originID string, originUnknown bool) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if originID == "" && !originUnknown {
		return fmt.Errorf("%w: origin_id is required unless the origin is marked unknown", apperrs.ErrInvalid)
	}
	if originID != "" && originUnknown {
		return fmt.Errorf("%w: give an origin_id or mark the origin unknown, not both", apperrs.ErrInvalid)
	}
	if originID == id {
		return fmt.Errorf("%w: a ticket cannot be found in itself", apperrs.ErrInvalid)
	}
	return nil
}

// RemoveFoundIn drops a ticket's found-in link or origin-unknown marker; removing a missing one is a no-op.
func (s *Service) RemoveFoundIn(ctx context.Context, id string) (*LinkSet, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	if err := s.mustExist(ctx, id); err != nil {
		return nil, fmt.Errorf("remove found-in from ticket %s: %w", id, err)
	}
	current, held, err := s.foundIn(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("remove found-in from ticket %s: %w", id, err)
	}
	if !held {
		return s.Links(ctx, id)
	}
	link := TicketLink{TicketID: id, Kind: LinkFoundIn, TargetID: current, CreatedAt: s.now().UTC()}
	if _, err := s.repo.DeleteLink(ctx, link, linkEvent(TopicLinkDeleted, link)); err != nil {
		return nil, fmt.Errorf("remove found-in from ticket %s: %w", id, err)
	}
	return s.Links(ctx, id)
}

// foundIn reports the origin a ticket's found-in link points at ("" when unknown) and whether it holds one.
func (s *Service) foundIn(ctx context.Context, id string) (string, bool, error) {
	from, _, err := s.repo.ListLinkEnds(ctx, id)
	if err != nil {
		return "", false, err
	}
	for _, e := range from {
		if e.Kind != LinkFoundIn {
			continue
		}
		if e.Ticket == nil {
			return "", true, nil
		}
		return e.Ticket.ID, true, nil
	}
	return "", false, nil
}

// AddBlocker records that a ticket waits on another reaching done; it refuses a cycle but never gates a status move.
func (s *Service) AddBlocker(ctx context.Context, id, blockerID string) (*LinkSet, error) {
	id, blockerID = strings.TrimSpace(id), strings.TrimSpace(blockerID)
	if id == "" || blockerID == "" {
		return nil, fmt.Errorf("%w: id and blocker_id are required", apperrs.ErrInvalid)
	}
	if id == blockerID {
		return nil, fmt.Errorf("%w: a ticket cannot be blocked by itself", apperrs.ErrInvalid)
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("add blocker to ticket %s: %w", id, err)
	}
	blocker, err := s.repo.GetByID(ctx, blockerID)
	if err != nil {
		return nil, fmt.Errorf("add blocker to ticket %s: %w", id, err)
	}
	existing, err := s.repo.BlockerIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("add blocker to ticket %s: %w", id, err)
	}
	for _, b := range existing {
		if b == blockerID {
			return s.Links(ctx, id)
		}
	}
	// ponytail: checked before the write, not inside it; two opposite links added in the same instant could both pass.
	cycle, err := s.waitsOn(ctx, blockerID, id)
	if err != nil {
		return nil, fmt.Errorf("add blocker to ticket %s: %w", id, err)
	}
	if cycle {
		return nil, fmt.Errorf("%w: that would form a cycle, %q already waits on %q directly or through other tickets", apperrs.ErrConflict, blocker.Title, t.Title)
	}
	link := TicketLink{TicketID: id, Kind: LinkBlockedBy, TargetID: blockerID, CreatedAt: s.now().UTC()}
	if err := s.repo.PutLink(ctx, link, linkEvent(TopicLinkCreated, link)); err != nil {
		return nil, fmt.Errorf("add blocker to ticket %s: %w", id, err)
	}
	return s.Links(ctx, id)
}

// waitsOn walks blocked-by links out from start and reports whether target is among its transitive blockers.
func (s *Service) waitsOn(ctx context.Context, start, target string) (bool, error) {
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		next, err := s.repo.BlockerIDs(ctx, queue[0])
		if err != nil {
			return false, err
		}
		queue = queue[1:]
		for _, id := range next {
			if id == target {
				return true, nil
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			queue = append(queue, id)
		}
	}
	return false, nil
}

// RemoveBlocker drops a blocked-by link; removing a missing one is a no-op.
func (s *Service) RemoveBlocker(ctx context.Context, id, blockerID string) (*LinkSet, error) {
	id, blockerID = strings.TrimSpace(id), strings.TrimSpace(blockerID)
	if id == "" || blockerID == "" {
		return nil, fmt.Errorf("%w: id and blocker_id are required", apperrs.ErrInvalid)
	}
	if err := s.mustExist(ctx, id); err != nil {
		return nil, fmt.Errorf("remove blocker from ticket %s: %w", id, err)
	}
	link := TicketLink{TicketID: id, Kind: LinkBlockedBy, TargetID: blockerID, CreatedAt: s.now().UTC()}
	if _, err := s.repo.DeleteLink(ctx, link, linkEvent(TopicLinkDeleted, link)); err != nil {
		return nil, fmt.Errorf("remove blocker from ticket %s: %w", id, err)
	}
	return s.Links(ctx, id)
}

// UnclearedBlockers maps each blocked ticket id to the blockers it still waits on, for the board's blocked icon.
func (s *Service) UnclearedBlockers(ctx context.Context) (map[string][]LinkedTicket, error) {
	out, err := s.repo.UnclearedBlockers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list uncleared blockers: %w", err)
	}
	return out, nil
}

// mustExist fetches each non-empty id so a missing ticket surfaces as ErrNotFound.
func (s *Service) mustExist(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := s.repo.GetByID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func linkEvent(topic string, link TicketLink) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: LinkEvent{Link: link}}
}

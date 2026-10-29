package tickets

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	dayLayout = "2006-01-02"
	// maxDueDays is the widest range of days DueBetween reads.
	maxDueDays = 62
)

// parseDay reads a YYYY-MM-DD day; field names the argument in the refusal.
func parseDay(field, v string) (time.Time, error) {
	d, err := time.Parse(dayLayout, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s must be a date as YYYY-MM-DD, such as 2026-10-30", apperrs.ErrInvalid, field)
	}
	return d, nil
}

// checkDueRange validates the optional ends of a due-date filter: each a YYYY-MM-DD day, from not after to.
func checkDueRange(from, to string) error {
	var start, end time.Time
	var err error
	if from != "" {
		if start, err = parseDay("due_from", from); err != nil {
			return err
		}
	}
	if to != "" {
		if end, err = parseDay("due_to", to); err != nil {
			return err
		}
	}
	if from != "" && to != "" && start.After(end) {
		return fmt.Errorf("%w: due_from must not be after due_to", apperrs.ErrInvalid)
	}
	return nil
}

// SetDueOn sets a ticket's due date, a YYYY-MM-DD day, in place; an empty dueOn clears it and the same date writes nothing.
func (s *Service) SetDueOn(ctx context.Context, id, dueOn string) (*Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	dueOn = strings.TrimSpace(dueOn)
	if dueOn != "" {
		if _, err := parseDay("due_on", dueOn); err != nil {
			return nil, err
		}
	}
	current, err := s.load(ctx, id, permissions.TicketsWrite)
	if err != nil {
		return nil, fmt.Errorf("set due date on ticket %s: %w", id, err)
	}
	if current.DueOn == dueOn {
		return current, nil
	}
	updated := *current
	updated.DueOn = dueOn
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDueDateChanged, Payload: DueDateChangedEvent{Ticket: updated, PreviousDueOn: current.DueOn, Actor: ActorOf(ctx)}}
	if err := s.repo.UpdateDueOn(ctx, id, dueOn, s.changed(ActorOf(ctx), FieldDueOn, current.DueOn, dueOn), evt); err != nil {
		return nil, fmt.Errorf("set due date on ticket %s: %w", id, err)
	}
	return &updated, nil
}

// DueBetween lists the tickets with a due date from..to inclusive, in every project of the workspace the caller may
// read tickets in, by due date then id; the range is at most 62 days. The caller must belong to the workspace.
func (s *Service) DueBetween(ctx context.Context, workspaceID, from, to string) ([]*Ticket, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if from == "" || to == "" {
		return nil, fmt.Errorf("%w: from and to are required, as YYYY-MM-DD", apperrs.ErrInvalid)
	}
	start, err := parseDay("from", from)
	if err != nil {
		return nil, err
	}
	end, err := parseDay("to", to)
	if err != nil {
		return nil, err
	}
	if end.Before(start) {
		return nil, fmt.Errorf("%w: from must not be after to", apperrs.ErrInvalid)
	}
	if end.Sub(start) >= maxDueDays*24*time.Hour {
		return nil, fmt.Errorf("%w: the range is at most %d days", apperrs.ErrInvalid, maxDueDays)
	}
	scope, err := s.readScopeIn(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ts, err := s.repo.ListDue(ctx, workspaceID, from, to, scope)
	if err != nil {
		return nil, fmt.Errorf("list tickets due %s to %s: %w", from, to, err)
	}
	return ts, nil
}

// readScopeIn is the projects of workspaceID whose tickets the caller may read; the caller must belong to it.
func (s *Service) readScopeIn(ctx context.Context, workspaceID string) (TicketScope, error) {
	if s.gate == nil {
		return TicketScope{All: permissions.Ungated(ctx) == nil}, nil
	}
	if err := s.gate.Require(ctx, workspaceID, permissions.Member); err != nil {
		return TicketScope{}, fmt.Errorf("list tickets due: %w", err)
	}
	projectIDs, all, err := s.gate.CallerProjectsIn(ctx, workspaceID, permissions.TicketsRead)
	if err != nil {
		return TicketScope{}, fmt.Errorf("list tickets due: %w", err)
	}
	return TicketScope{All: all, ProjectIDs: projectIDs}, nil
}

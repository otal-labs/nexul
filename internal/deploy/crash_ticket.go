package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TicketDesk is the tickets side of crash tickets, wired in the composition root (ADR 0017).
type TicketDesk interface {
	// File opens a ticket in the project and returns its id.
	File(ctx context.Context, projectID, title, body string) (string, error)
	// Open reports whether the ticket still exists and is not in the done stage.
	Open(ctx context.Context, ticketID string) (bool, error)
	// Comment adds body to the ticket's thread.
	Comment(ctx context.Context, ticketID, body string) error
}

// SetCrashTickets wires the crash ticket switch's ticket desk and its links; unset, the consumers do nothing.
func (s *Service) SetCrashTickets(desk TicketDesk, links CrashTicketRepo) {
	s.desk, s.crashLinks = desk, links
}

// HandleContainerStopped is the service.container_stopped consumer: a stack with the switch on gets a ticket for the
// service, or a comment on the one it already has open.
func (s *Service) HandleContainerStopped(ctx context.Context, ev eventbus.Event) error {
	p, stack, err := s.crashTarget(ctx, ev)
	if err != nil || stack == nil {
		return err
	}
	return s.FileOrComment(ctx, stack, p.Service, crashTitle(p), crashBody(p), crashRepeat(p))
}

// HandleContainerRecovered is the service.container_recovered consumer: it comments on the open ticket and leaves it
// open, because a crash that stopped is not a crash that was fixed.
func (s *Service) HandleContainerRecovered(ctx context.Context, ev eventbus.Event) error {
	p, stack, err := s.crashTarget(ctx, ev)
	if err != nil || stack == nil {
		return err
	}
	return s.CommentOpen(ctx, stack, p.Service, crashRecovered(p))
}

// crashTarget parses an outage event and answers the stack to act on: nil unless the stack still exists, still has a
// project, and still has the switch on.
func (s *Service) crashTarget(ctx context.Context, ev eventbus.Event) (OutageEvent, *Stack, error) {
	var p OutageEvent
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return p, nil, apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if s.desk == nil {
		return p, nil, nil
	}
	stack, err := s.stacks.GetByID(ctx, p.StackID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return p, nil, nil
	}
	if err != nil {
		return p, nil, fmt.Errorf("get stack %s: %w", p.StackID, err)
	}
	if !stack.CrashTickets || stack.ProjectID == "" {
		return p, nil, nil
	}
	return p, stack, nil
}

// FileOrComment opens a ticket in the stack's project for subject unless its earlier ticket is still open, in which
// case repeat is added to that one. A subject is a service name today; an error-rate episode brings its own key.
func (s *Service) FileOrComment(ctx context.Context, stack *Stack, subject, title, body, repeat string) error {
	id, err := s.openCrashTicket(ctx, stack.ID, subject)
	if err != nil {
		return err
	}
	if id != "" {
		return s.desk.Comment(ctx, id, repeat)
	}
	id, err = s.desk.File(ctx, stack.ProjectID, title, body)
	if err != nil {
		return fmt.Errorf("open ticket for %s on stack %s: %w", subject, stack.ID, err)
	}
	return s.crashLinks.Set(ctx, stack.ID, subject, id)
}

// CommentOpen adds comment to subject's ticket when it is still open, and does nothing otherwise.
func (s *Service) CommentOpen(ctx context.Context, stack *Stack, subject, comment string) error {
	id, err := s.openCrashTicket(ctx, stack.ID, subject)
	if err != nil || id == "" {
		return err
	}
	return s.desk.Comment(ctx, id, comment)
}

// openCrashTicket is the id of subject's linked ticket while it is open, else "".
func (s *Service) openCrashTicket(ctx context.Context, stackID, subject string) (string, error) {
	id, err := s.crashLinks.Get(ctx, stackID, subject)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	open, err := s.desk.Open(ctx, id)
	if err != nil {
		return "", fmt.Errorf("check ticket %s: %w", id, err)
	}
	if !open {
		return "", nil
	}
	return id, nil
}

// crashReason reads an outage's exit like "exit 137, out of memory"; an out-of-memory kill seen before its exit code is just that.
func crashReason(p OutageEvent) string {
	var parts []string
	if p.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("exit %d", p.ExitCode))
	}
	if p.OOM {
		parts = append(parts, "out of memory")
	}
	return strings.Join(parts, ", ")
}

func crashTitle(p OutageEvent) string {
	return fmt.Sprintf("%s crashed: %s", p.Service, crashReason(p))
}

func crashBody(p OutageEvent) string {
	return fmt.Sprintf("Service `%s` of stack `%s` on machine `%s` went down: %s.\n\n"+
		"- Down since: %s\n- Outage: %s\n\n"+
		"Read what the container printed with `stack_get` on stack `%s` and `logs` for service `%s`. "+
		"Container output is never stored, so it is read live.",
		p.Service, p.StackName, p.Machine, crashReason(p), p.StartedAt.UTC().Format(time.RFC3339), p.OutageID, p.StackID, p.Service)
}

func crashRepeat(p OutageEvent) string {
	return fmt.Sprintf("`%s` crashed again: %s. Down since %s (outage %s).", p.Service, crashReason(p), p.StartedAt.UTC().Format(time.RFC3339), p.OutageID)
}

func crashRecovered(p OutageEvent) string {
	back := p.StartedAt
	if p.EndedAt != nil {
		back = *p.EndedAt
	}
	more := ""
	if p.RestartCount > 0 {
		more = fmt.Sprintf(" It crashed %d more times meanwhile.", p.RestartCount)
	}
	return fmt.Sprintf("`%s` came back at %s after being down since %s.%s This ticket stays open until someone closes it.",
		p.Service, back.UTC().Format(time.RFC3339), p.StartedAt.UTC().Format(time.RFC3339), more)
}

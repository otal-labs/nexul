package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// settleTimeout bounds one settle call; it outlives the turn's own context, which may already be spent.
const settleTimeout = 30 * time.Second

// endedTurn is a ticket thread's last turn: the conversation it ran on, the client, and the harness session.
type endedTurn struct {
	conversationID string
	turn           activeTurn
}

// endTurn clears the conversation's live turn, then settles the ticket's harness session if the ticket is done.
// ponytail: last turns live in memory, so a restart forgets them; persist the computer per conversation if that matters.
func (s *Service) endTurn(ctx context.Context, conv Conversation, turn *activeTurn) {
	s.clearActive(conv.ID)
	if !conv.IsTicketThread || conv.TicketID == "" || s.tickets == nil || turn.target.SessionID == "" {
		return
	}
	s.mu.Lock()
	s.ended[conv.TicketID] = endedTurn{conversationID: conv.ID, turn: *turn}
	s.mu.Unlock()
	s.settleTicket(ctx, conv.TicketID)
}

// ticketMoved is the slice of ticket.status_changed the settle reads, declared here so agent never imports tickets.
type ticketMoved struct {
	Ticket struct {
		ID string `json:"id"`
	} `json:"ticket"`
}

// HandleTicketStatusChanged is the ticket.status_changed subscription: a ticket done after its last turn settles it.
func (s *Service) HandleTicketStatusChanged(ctx context.Context, ev eventbus.Event) error {
	var m ticketMoved
	if err := json.Unmarshal(ev.Payload, &m); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if m.Ticket.ID == "" {
		return apperrs.Fatal(fmt.Errorf("%s missing ticket id", ev.Topic))
	}
	s.settleTicket(ctx, m.Ticket.ID)
	return nil
}

// settleTicket is best effort: a refusal (an old harness, an open question) only logs, and settling twice is harmless.
func (s *Service) settleTicket(ctx context.Context, ticketID string) {
	s.mu.Lock()
	e, ok := s.ended[ticketID]
	_, running := s.active[e.conversationID]
	s.mu.Unlock()
	if !ok || running {
		return
	}
	t, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		s.log.Warn("agent: read ticket for settle failed", "ticket", ticketID, "error", err)
		return
	}
	if !t.Done {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleTimeout)
	defer cancel()
	if err := e.turn.client.Settle(ctx, e.turn.target); err != nil {
		s.log.Warn("agent: settle harness session failed", "ticket", ticketID, "session", e.turn.target.SessionID, "error", err)
		return
	}
	s.mu.Lock()
	delete(s.ended, ticketID)
	s.mu.Unlock()
}

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

// endTurn clears this turn from the live ones, then settles its thread if the ticket is done or the doc clarified.
// ponytail: last turns live in memory, so a restart forgets them; persist the computer per conversation if that matters.
func (s *Service) endTurn(ctx context.Context, conv Conversation, turn *activeTurn) {
	s.clearActive(conv.ID, turn)
	if turn.target.SessionID == "" {
		return
	}
	if conv.IsTicketThread && conv.TicketID != "" && s.tickets != nil {
		s.remember(conv.TicketID, conv.ID, turn)
		s.settleWhenDone(ctx, conv.TicketID, s.ticketDone)
		return
	}
	if conv.IsDocThread && conv.DocID != "" && s.docs != nil {
		s.remember(conv.DocID, conv.ID, turn)
		s.settleWhenDone(ctx, conv.DocID, s.docs.Clarified)
	}
}

func (s *Service) remember(targetID, conversationID string, turn *activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ended[targetID] = endedTurn{conversationID: conversationID, turn: *turn}
}

func (s *Service) ticketDone(ctx context.Context, id string) (bool, error) {
	t, err := s.tickets.Get(ctx, id)
	return err == nil && t.Done, err
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
	s.settleWhenDone(ctx, m.Ticket.ID, s.ticketDone)
	return nil
}

// clarificationClosed is the slice of doc.clarification.closed the settle reads, declared so agent never imports docs.
type clarificationClosed struct {
	Doc struct {
		ID string `json:"id"`
	} `json:"doc"`
}

// HandleClarificationClosed is the doc.clarification.closed subscription: a clarification closed after its last turn settles it.
func (s *Service) HandleClarificationClosed(ctx context.Context, ev eventbus.Event) error {
	var c clarificationClosed
	if err := json.Unmarshal(ev.Payload, &c); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if c.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("%s missing doc id", ev.Topic))
	}
	if s.docs == nil {
		return nil
	}
	s.settleWhenDone(ctx, c.Doc.ID, s.docs.Clarified)
	return nil
}

// settleWhenDone settles a ticket's or doc's thread once done says so and no turn runs; best effort, twice is harmless.
func (s *Service) settleWhenDone(ctx context.Context, targetID string, done func(context.Context, string) (bool, error)) {
	s.mu.Lock()
	e, ok := s.ended[targetID]
	running := len(s.active[e.conversationID]) > 0
	s.mu.Unlock()
	if !ok || running {
		return
	}
	isDone, err := done(ctx, targetID)
	if err != nil {
		s.log.Warn("agent: read thread target for settle failed", "target", targetID, "error", err)
		return
	}
	if !isDone {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleTimeout)
	defer cancel()
	if err := e.turn.client.Settle(ctx, e.turn.target); err != nil {
		s.log.Warn("agent: settle harness session failed", "target", targetID, "session", e.turn.target.SessionID, "error", err)
		return
	}
	s.mu.Lock()
	delete(s.ended, targetID)
	s.mu.Unlock()
}

package agent

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// backgroundWork is a ticket thread whose harness reports work left running past its turn.
type backgroundWork struct {
	conversationID string
	ticketID       string
	computerID     string
}

// BackgroundTickets are the given tickets whose thread has work running in the harness after its turn ended.
func (s *Service) BackgroundTickets(ticketIDs []string) map[string]bool {
	asked := make(map[string]bool, len(ticketIDs))
	for _, id := range ticketIDs {
		asked[id] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]bool{}
	for _, w := range s.background {
		if asked[w.ticketID] {
			out[w.ticketID] = true
		}
	}
	return out
}

// noteBackground records that a ticket thread has background work, announcing it the first time.
func (s *Service) noteBackground(ctx context.Context, threadID, computerID string, conv Conversation) {
	if conv.TicketID == "" {
		return
	}
	s.mu.Lock()
	_, known := s.background[threadID]
	s.background[threadID] = backgroundWork{conversationID: conv.ID, ticketID: conv.TicketID, computerID: computerID}
	s.mu.Unlock()
	if !known {
		s.publishBackground(ctx, conv.ID, conv.TicketID, true)
	}
}

func (s *Service) endBackground(ctx context.Context, threadID string) {
	s.mu.Lock()
	w, known := s.background[threadID]
	delete(s.background, threadID)
	s.mu.Unlock()
	if known {
		s.publishBackground(ctx, w.conversationID, w.ticketID, false)
	}
}

func (s *Service) publishBackground(ctx context.Context, conversationID, ticketID string, running bool) {
	if err := s.live.Publish(ctx, TopicAgentStream, StreamFrame{ConversationID: conversationID, TicketID: ticketID, Background: &running}); err != nil {
		s.log.Warn("agent: publish background work frame failed", "conversation", conversationID, "error", err)
	}
}

// presenceChanged is the slice of computer.presence_changed the agent reads, declared so agent never imports pairing.
type presenceChanged struct {
	ComputerID string `json:"computer_id"`
	State      string `json:"state"`
}

// HandlePresenceChanged is the computer.presence_changed subscription: once Nexul stops hearing a computer, what it
// said was running there is no longer known, so its threads stop showing background work.
func (s *Service) HandlePresenceChanged(ctx context.Context, ev eventbus.Event) error {
	var p presenceChanged
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.State == "connected" {
		return nil
	}
	s.mu.Lock()
	var lost []string
	for threadID, w := range s.background {
		if w.computerID == p.ComputerID {
			lost = append(lost, threadID)
		}
	}
	s.mu.Unlock()
	for _, threadID := range lost {
		s.endBackground(ctx, threadID)
	}
	return nil
}

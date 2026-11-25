package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// HandleComputerGrantChanged interrupts the turns a person is running on someone else's computer as soon as its share
// no longer gives them Run agents (ADR 0210).
func (s *Service) HandleComputerGrantChanged(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		ComputerID  string   `json:"computer_id"`
		UserID      string   `json:"user_id"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.UserID == "" || slices.Contains(p.Permissions, string(permissions.ComputersAgents)) {
		return nil
	}
	for _, t := range s.guestTurns(p.ComputerID, p.UserID) {
		if err := t.client.Interrupt(ctx, t.target); err != nil {
			s.log.Warn("agent: interrupt a turn whose share ended", "computer_id", p.ComputerID, "error", err)
		}
	}
	return nil
}

// guestTurns are the in-flight turns userID runs on computerID, which someone else owns.
func (s *Service) guestTurns(computerID, userID string) []*activeTurn {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*activeTurn
	for _, turns := range s.active {
		for _, t := range turns {
			if t.computerID == computerID && t.viaUserID == userID && t.ownerID != userID {
				out = append(out, t)
			}
		}
	}
	return out
}

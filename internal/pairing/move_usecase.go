package pairing

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// onRunner reports a computer whose session is through its personal runner, not a tunnel or an address.
func onRunner(c Computer) bool {
	return c.ServerURL == runnerAddress(c.ID)
}

// hasRunner reports a computer with a personal runner enrolled; a failed read counts as none.
func (s *Service) hasRunner(ctx context.Context, computerID string) bool {
	if s.runners == nil {
		return false
	}
	_, err := s.runners.ComputerRunner(ctx, computerID)
	return err == nil
}

// retireTunnel deletes the tunnel, DNS record and Access app of a computer now paired through its runner, then forgets
// them; a failure keeps the tunnel, logged, and the runner's next report retries it.
func (s *Service) retireTunnel(ctx context.Context, c Computer) bool {
	log := logging.FromCtx(ctx)
	tunnels, err := s.tunnelSeam()
	if err != nil {
		log.Warn("moved computer keeps its tunnel", "computer_id", c.ID, "error", err)
		return false
	}
	if err := tunnels.DeleteTunnel(ctx, *c.Tunnel); err != nil {
		log.Warn("moved computer keeps its tunnel", "computer_id", c.ID, "error", err)
		return false
	}
	if err := s.repo.ClearTunnel(ctx, c.UserID, c.ID, s.now().UTC(), tunnelEvent(TopicTunnelRemoved, c)); err != nil {
		log.Warn("moved computer's tunnel deleted but not forgotten", "computer_id", c.ID, "error", err)
		return false
	}
	s.stopWatching(c.ID)
	s.notifyComputersChanged(c.UserID)
	log.Info("moved computer's tunnel retired", "computer_id", c.ID, "tunnel_id", c.Tunnel.TunnelID)
	return true
}

// AnnounceMoves tells each owner of a computer still paired through a tunnel or by URL, once, to move it to the Nexul
// app. Its stored receipt keeps the notice to once across restarts, and its time starts the owner's 30 days (ADR 0146).
func (s *Service) AnnounceMoves(ctx context.Context) error {
	owners, err := s.repo.ComputersToMove(ctx)
	if err != nil {
		return err
	}
	at := s.now().UTC()
	for _, c := range owners {
		evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMoveRequested, Payload: MoveRequestedEvent{ComputerID: c.ID, UserID: c.UserID, MembersOnly: true}}
		if err := s.repo.RecordMoveNotice(ctx, c.UserID, at, evt); err != nil {
			return fmt.Errorf("announce the move to %s: %w", c.UserID, err)
		}
	}
	return nil
}

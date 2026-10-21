package pairing

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Publisher is the bus seam for ephemeral events that are never written to the outbox.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// Tunnels tears down what pairing through a tunnel left on the instance's Cloudflare (ADR 0062, retired by ADR 0146).
type Tunnels interface {
	// DeleteTunnel disconnects the computer, then deletes the tunnel, DNS record, and Access app; parts already gone count.
	DeleteTunnel(ctx context.Context, t ComputerTunnel) error
	// DeleteAccessServiceToken deletes the instance's one Access service token; with none stored it does nothing.
	DeleteAccessServiceToken(ctx context.Context) error
}

// RetireTunnels deletes every computer tunnel still left at Cloudflare, then the Access service token they shared. The
// computers keep their rows, links and setup; a refused teardown keeps its tunnel for the next boot to retry.
func (s *Service) RetireTunnels(ctx context.Context) error {
	cs, err := s.repo.ListTunnelComputers(ctx)
	if err != nil {
		return fmt.Errorf("list computers on a tunnel: %w", err)
	}
	left := 0
	for _, c := range cs {
		if !s.retireTunnel(ctx, c) {
			left++
		}
	}
	if left > 0 {
		return fmt.Errorf("%d computer tunnels are still at Cloudflare", left)
	}
	tunnels, err := s.tunnelSeam()
	if err != nil {
		return err
	}
	if err := tunnels.DeleteAccessServiceToken(ctx); err != nil {
		return fmt.Errorf("delete the Access service token: %w", err)
	}
	return nil
}

// DeleteComputer retires a computer's runner, revokes its MCP token, tears down any tunnel left, then removes it; a
// mismatched id is ErrNotFound.
func (s *Service) DeleteComputer(ctx context.Context, userID, id string) error {
	computer, err := s.ownComputer(ctx, userID, id)
	if err != nil {
		return err
	}
	// The runner goes first: a failed retire keeps the row, so removing the computer again retries it.
	if s.runners != nil {
		if err := s.runners.RetireComputerRunner(ctx, computer.ID); err != nil {
			return fmt.Errorf("retire runner of computer %s: %w", computer.ID, err)
		}
	}
	if err := s.revokeMCPToken(ctx, *computer); err != nil {
		return err
	}
	evts := []eventbus.OutboxEvent{{ID: ids.New(), Topic: TopicComputerRemoved, Payload: ComputerRemovedEvent{ComputerID: computer.ID, UserID: computer.UserID, MembersOnly: true}}}
	if computer.Tunnel != nil {
		tunnels, err := s.tunnelSeam()
		if err != nil {
			return err
		}
		// Cloudflare goes first: a failed teardown keeps the row, so removing the computer again retries it.
		if err := tunnels.DeleteTunnel(ctx, *computer.Tunnel); err != nil {
			return fmt.Errorf("remove tunnel for computer %s: %w", computer.ID, err)
		}
		evts = append(evts, tunnelEvent(TopicTunnelRemoved, *computer))
	}
	if err := s.repo.DeleteComputer(ctx, userID, computer.ID, evts...); err != nil {
		return fmt.Errorf("delete computer %s: %w", computer.ID, err)
	}
	s.notifyComputersChanged(userID)
	return nil
}

// retireTunnel deletes a computer's tunnel, DNS record and Access app, then forgets them; a failure keeps the tunnel,
// logged, for the next try.
func (s *Service) retireTunnel(ctx context.Context, c Computer) bool {
	log := logging.FromCtx(ctx)
	tunnels, err := s.tunnelSeam()
	if err != nil {
		log.Warn("computer keeps its tunnel", "computer_id", c.ID, "error", err)
		return false
	}
	if err := tunnels.DeleteTunnel(ctx, *c.Tunnel); err != nil {
		log.Warn("computer keeps its tunnel", "computer_id", c.ID, "error", err)
		return false
	}
	if err := s.repo.ClearTunnel(ctx, c.UserID, c.ID, s.now().UTC(), tunnelEvent(TopicTunnelRemoved, c)); err != nil {
		log.Warn("computer's tunnel deleted but not forgotten", "computer_id", c.ID, "error", err)
		return false
	}
	s.notifyComputersChanged(c.UserID)
	log.Info("computer's tunnel retired", "computer_id", c.ID, "tunnel_id", c.Tunnel.TunnelID)
	return true
}

func (s *Service) tunnelSeam() (Tunnels, error) {
	if s.tunnels == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: computer tunnels are not wired", apperrs.ErrFatal))
	}
	return s.tunnels, nil
}

func tunnelEvent(topic string, c Computer) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: TunnelChangedEvent{
		ComputerID: c.ID, UserID: c.UserID, TunnelID: c.Tunnel.TunnelID, Hostname: c.Tunnel.Hostname, MembersOnly: true,
	}}
}

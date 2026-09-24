package pairing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// TunnelLifecycle creates and removes a computer's tunnel on the instance's own Cloudflare (ADR 0062).
type TunnelLifecycle interface {
	// CreateTunnel closes the new hostname behind an Access app before its DNS record exists, routed to the loopback port.
	CreateTunnel(ctx context.Context, name string, port int) (*ComputerTunnel, error)
	// DeleteTunnel disconnects the computer, then deletes the tunnel, DNS record, and Access app; parts already gone count.
	DeleteTunnel(ctx context.Context, t ComputerTunnel) error
}

// TunnelReader reads a computer tunnel's live state from Cloudflare.
type TunnelReader interface {
	// TunnelStatus returns Cloudflare's connector status: inactive, healthy, degraded, or down.
	TunnelStatus(ctx context.Context, tunnelID string) (string, error)
	// TunnelToken returns the token the computer's cloudflared service runs with.
	TunnelToken(ctx context.Context, tunnelID string) (string, error)
}

// Tunnels is the whole computer-tunnel seam; the composition root adapts the dns domain to it.
type Tunnels interface {
	TunnelLifecycle
	TunnelReader
}

// CreateComputerTunnel adds a computer reached through its own tunnel; it stays unpaired until the harness pairs over the hostname.
func (s *Service) CreateComputerTunnel(ctx context.Context, userID string, kind harness.Kind, name string, port int) (*Computer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	if _, ok := s.harnesses[kind]; !ok {
		return nil, fmt.Errorf("%w: unsupported harness kind %q", apperrs.ErrInvalid, kind)
	}
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	if err := validatePort(port); err != nil {
		return nil, err
	}
	tunnels, err := s.tunnelSeam()
	if err != nil {
		return nil, err
	}
	t, err := tunnels.CreateTunnel(ctx, name, port)
	if err != nil {
		return nil, fmt.Errorf("create tunnel for %s: %w", name, err)
	}
	now := s.now().UTC()
	computer := Computer{
		ID: ids.New(), UserID: userID, Kind: kind, Name: name, ServerURL: "https://" + t.Hostname,
		CreatedAt: now, UpdatedAt: now, Tunnel: t,
	}
	if err := s.repo.SaveComputer(ctx, computer, tunnelEvent(TopicTunnelCreated, computer)); err != nil {
		// A tunnel with no stored computer could never be removed from Nexul, so it goes now.
		return nil, errors.Join(fmt.Errorf("save computer: %w", err), tunnels.DeleteTunnel(ctx, *t))
	}
	s.notifyComputersChanged(userID)
	s.watchTunnel(userID, computer.ID)
	return &computer, nil
}

// ComputerTunnelToken returns the connector token for the caller's computer, for its cloudflared install command.
func (s *Service) ComputerTunnelToken(ctx context.Context, userID, computerID string) (string, error) {
	computer, tunnels, err := s.tunnelComputer(ctx, userID, computerID)
	if err != nil {
		return "", err
	}
	token, err := tunnels.TunnelToken(ctx, computer.Tunnel.TunnelID)
	if err != nil {
		return "", fmt.Errorf("get tunnel token for computer %s: %w", computer.ID, err)
	}
	return token, nil
}

// ComputerTunnelStatus reports the tunnel's two checks and, until both pass, keeps pushing their changes live.
func (s *Service) ComputerTunnelStatus(ctx context.Context, userID, computerID string) (TunnelStatus, error) {
	status, err := s.tunnelStatus(ctx, userID, computerID)
	if err != nil {
		return TunnelStatus{}, err
	}
	if !status.Connected() {
		s.watchTunnel(userID, computerID)
	}
	return status, nil
}

// tunnelStatus reads Cloudflare's connector status, then probes the harness through the hostname.
func (s *Service) tunnelStatus(ctx context.Context, userID, computerID string) (TunnelStatus, error) {
	computer, tunnels, err := s.tunnelComputer(ctx, userID, computerID)
	if err != nil {
		return TunnelStatus{}, err
	}
	state, err := tunnels.TunnelStatus(ctx, computer.Tunnel.TunnelID)
	if err != nil {
		return TunnelStatus{}, fmt.Errorf("get tunnel status for computer %s: %w", computer.ID, err)
	}
	out := TunnelStatus{Tunnel: state}
	if state != "healthy" && state != "degraded" {
		return out, nil
	}
	client, err := s.client(computer.Kind)
	if err != nil {
		return TunnelStatus{}, err
	}
	// Healthy only proves cloudflared reached Cloudflare; the probe proves the harness answers behind it.
	version, err := client.Version(ctx, "https://"+computer.Tunnel.Hostname)
	if err != nil {
		logging.FromCtx(ctx).Debug("harness probe through computer tunnel failed", "computer_id", computer.ID, "error", err)
		return out, nil
	}
	out.HarnessReachable = true
	out.HarnessVersion = version
	return out, nil
}

// DeleteComputer revokes a paired computer's MCP token, tears down its tunnel, then removes it; a mismatched id is ErrNotFound.
func (s *Service) DeleteComputer(ctx context.Context, userID, id string) error {
	computer, err := s.ownComputer(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.revokeMCPToken(ctx, *computer); err != nil {
		return err
	}
	var evts []eventbus.OutboxEvent
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

// tunnelComputer returns the caller's own computer with its tunnel seam; a computer paired by URL has no tunnel.
func (s *Service) tunnelComputer(ctx context.Context, userID, computerID string) (*Computer, Tunnels, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, nil, err
	}
	if computer.Tunnel == nil {
		return nil, nil, fmt.Errorf("%w: computer %s is paired by URL and has no tunnel", apperrs.ErrInvalid, computer.ID)
	}
	tunnels, err := s.tunnelSeam()
	if err != nil {
		return nil, nil, err
	}
	return computer, tunnels, nil
}

func (s *Service) tunnelSeam() (Tunnels, error) {
	if s.tunnels == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: computer tunnels are not wired", apperrs.ErrFatal))
	}
	return s.tunnels, nil
}

func tunnelEvent(topic string, c Computer) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: TunnelChangedEvent{
		ComputerID: c.ID, UserID: c.UserID, TunnelID: c.Tunnel.TunnelID, Hostname: c.Tunnel.Hostname,
	}}
}

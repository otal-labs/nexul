package main

import (
	"context"
	"errors"

	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// pairingTunnels adapts dns's computer-tunnel use-cases to pairing's Tunnels seam (ADR 0017).
type pairingTunnels struct {
	dns *dns.Service
}

func (a pairingTunnels) CreateTunnel(ctx context.Context, name string, port int) (*pairing.ComputerTunnel, error) {
	t, err := a.dns.CreateComputerTunnel(ctx, name, port)
	if err != nil {
		return nil, tunnelPrerequisite(err)
	}
	out := pairing.ComputerTunnel(*t)
	return &out, nil
}

func (a pairingTunnels) DeleteTunnel(ctx context.Context, t pairing.ComputerTunnel) error {
	return tunnelPrerequisite(a.dns.DeleteComputerTunnel(ctx, dns.ComputerTunnel(t)))
}

func (a pairingTunnels) TunnelStatus(ctx context.Context, tunnelID string) (string, error) {
	status, err := a.dns.ComputerTunnelStatus(ctx, tunnelID)
	return status, tunnelPrerequisite(err)
}

func (a pairingTunnels) TunnelToken(ctx context.Context, tunnelID string) (string, error) {
	token, err := a.dns.ComputerTunnelToken(ctx, tunnelID)
	return token, tunnelPrerequisite(err)
}

// tunnelPrerequisite names a missing instance prerequisite in pairing's terms, so the dialog can show its fix.
func tunnelPrerequisite(err error) error {
	if errors.Is(err, dns.ErrCloudflareNotConnected) {
		return &pairing.PrerequisiteError{Reason: pairing.ReasonCloudflareNotConnected, Err: err}
	}
	if errors.Is(err, dns.ErrZeroTrustDisabled) {
		return &pairing.PrerequisiteError{Reason: pairing.ReasonZeroTrustDisabled, Err: err}
	}
	return err
}

// computerTunnelAccess hands out the Access service token for computer tunnel hostnames and nothing else.
type computerTunnelAccess struct {
	hosts *storage.PairingRepo
	dns   *dns.Service
}

func (a computerTunnelAccess) Credentials(ctx context.Context, host string) (string, string, bool, error) {
	ok, err := a.hosts.ComputerTunnelHostnameExists(ctx, host)
	if err != nil || !ok {
		return "", "", false, err
	}
	tok, err := a.dns.EnsureAccessServiceToken(ctx)
	if err != nil {
		return "", "", false, err
	}
	return tok.ClientID, tok.ClientSecret, true, nil
}

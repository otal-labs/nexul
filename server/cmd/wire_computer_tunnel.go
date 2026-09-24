package main

import (
	"context"

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
		return nil, err
	}
	out := pairing.ComputerTunnel(*t)
	return &out, nil
}

func (a pairingTunnels) DeleteTunnel(ctx context.Context, t pairing.ComputerTunnel) error {
	return a.dns.DeleteComputerTunnel(ctx, dns.ComputerTunnel(t))
}

func (a pairingTunnels) TunnelStatus(ctx context.Context, tunnelID string) (string, error) {
	return a.dns.ComputerTunnelStatus(ctx, tunnelID)
}

func (a pairingTunnels) TunnelToken(ctx context.Context, tunnelID string) (string, error) {
	return a.dns.ComputerTunnelToken(ctx, tunnelID)
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

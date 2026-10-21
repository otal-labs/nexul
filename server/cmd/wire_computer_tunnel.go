package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/pairing"
)

// pairingTunnels adapts dns's computer-tunnel teardown to pairing's Tunnels seam (ADR 0017).
type pairingTunnels struct {
	dns *dns.Service
}

func (a pairingTunnels) DeleteTunnel(ctx context.Context, t pairing.ComputerTunnel) error {
	return a.dns.DeleteComputerTunnel(ctx, dns.ComputerTunnel(t))
}

func (a pairingTunnels) DeleteAccessServiceToken(ctx context.Context) error {
	return a.dns.DeleteAccessServiceToken(ctx)
}

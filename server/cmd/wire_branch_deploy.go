package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// deployGatewayLookupAdapter adapts dns to deploy's GatewayLookup seam (deploy must never import dns).
type deployGatewayLookupAdapter struct {
	dns *dns.Service
}

func (a deployGatewayLookupAdapter) NetworkHasGateway(ctx context.Context, dockerNetwork string) (bool, error) {
	_, err := a.dns.GatewayForNetwork(ctx, dockerNetwork)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// deployExposureAdapter adapts dns to deploy's ExposureManager seam; EnsureExposure is idempotent, updating in place.
type deployExposureAdapter struct {
	dns *dns.Service
}

func (a deployExposureAdapter) EnsureExposure(ctx context.Context, hostname, service string, port int, dockerNetwork string) error {
	gw, err := a.dns.GatewayForNetwork(ctx, dockerNetwork)
	if err != nil {
		return fmt.Errorf("resolve gateway for network %s: %w", dockerNetwork, err)
	}
	existing, err := a.dns.GetExposureByService(ctx, service)
	switch {
	case err == nil:
		if existing.Hostname == hostname && existing.Port == port {
			return nil
		}
		if err := a.dns.DeleteExposure(ctx, existing.ID); err != nil {
			return fmt.Errorf("replace exposure for service %s: %w", service, err)
		}
	case !errors.Is(err, apperrs.ErrNotFound):
		return fmt.Errorf("check existing exposure for service %s: %w", service, err)
	}
	if _, err := a.dns.CreateExposure(ctx, dns.CreateExposureInput{
		GatewayID: gw.ID, Hostname: hostname, Service: service, Port: port, ZoneID: gw.ZoneID, Zone: gw.Zone,
	}); err != nil {
		return fmt.Errorf("create exposure for service %s: %w", service, err)
	}
	return nil
}

// deployGatewayJoinAdapter adapts dns to deploy's GatewayJoin seam (deploy must never import dns), so a
// deploy's last step can rejoin its gateway onto the stack's own network.
type deployGatewayJoinAdapter struct {
	dns *dns.Service
}

func (a deployGatewayJoinAdapter) GatewayForStackNetwork(ctx context.Context, network string) (string, bool, error) {
	return a.dns.GatewayForStackNetwork(ctx, network)
}

func (a deployExposureAdapter) RemoveExposures(ctx context.Context, stackName string, containerIDs []string) error {
	return a.dns.DeleteExposuresForStack(ctx, stackName, containerIDs)
}

// deployRequestSourcesAdapter adapts dns to deploy's RequestSources seam (deploy must never import dns): where
// hostnames are routed, and Cloudflare's per-minute counts for the tunnel ones.
type deployRequestSourcesAdapter struct {
	dns *dns.Service
}

func (a deployRequestSourcesAdapter) Routes(ctx context.Context) ([]deploy.RequestRoute, error) {
	routes, err := a.dns.RequestRoutes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]deploy.RequestRoute, 0, len(routes))
	for _, r := range routes {
		out = append(out, deploy.RequestRoute{
			Router: r.Router, Hostname: r.Hostname, ContainerID: r.ContainerID, ZoneID: r.ZoneID, Machine: r.Machine, Tunnel: r.Tunnel,
		})
	}
	return out, nil
}

func (a deployRequestSourcesAdapter) TunnelCounts(ctx context.Context, zoneID string, hostnames []string, from, to time.Time) ([]deploy.HostMinute, error) {
	counts, err := a.dns.HostnameRequests(ctx, zoneID, hostnames, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]deploy.HostMinute, 0, len(counts))
	for _, c := range counts {
		out = append(out, deploy.HostMinute{Hostname: c.Hostname, Minute: c.Minute, Requests: c.Requests, Errors: c.Errors})
	}
	return out, nil
}

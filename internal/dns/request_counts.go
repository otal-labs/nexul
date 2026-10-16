package dns

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// RouterName is the Traefik router an exposure's route names; the request counter's router label leads back to the
// exposure through it (Traefik appends the provider, as in nexul-<id>@file).
func RouterName(exposureID string) string { return "nexul-" + exposureID }

// RequestRoute is one exposure as request counting reads it: the container its hostname reaches and how the gateway
// carrying it is counted.
type RequestRoute struct {
	Router      string
	Hostname    string
	ContainerID string
	ZoneID      string
	Machine     string
	// Tunnel is true when Cloudflare counts the hostname, false when the machine's Traefik does.
	Tunnel bool
}

// HostMinute is one hostname's requests in one minute, and how many of them answered 5xx.
type HostMinute struct {
	Hostname string
	Minute   time.Time
	Requests int64
	Errors   int64
}

// AnalyticsProvider is what a DNS provider that reads per-hostname request counts adds to DNSProvider.
type AnalyticsProvider interface {
	// HostnameRequests counts the zone's requests for hostnames per minute over [from, to).
	HostnameRequests(ctx context.Context, zoneID string, hostnames []string, from, to time.Time) ([]HostMinute, error)
}

// RequestRoutes lists every exposure that reaches a container, for the background counting that has no actor.
func (s *Service) RequestRoutes(ctx context.Context) ([]RequestRoute, error) {
	exps, err := s.repo.ListExposures(ctx)
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	gateways, err := s.repo.ListGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("list gateways: %w", err)
	}
	byID := make(map[string]*Gateway, len(gateways))
	for _, g := range gateways {
		byID[g.ID] = g
	}
	routes := make([]RequestRoute, 0, len(exps))
	for _, e := range exps {
		g := byID[e.GatewayID]
		if g == nil || e.ServiceID == "" {
			continue
		}
		routes = append(routes, RequestRoute{
			Router: RouterName(e.ID), Hostname: e.Hostname, ContainerID: e.ServiceID, ZoneID: e.ZoneID,
			Machine: g.Machine, Tunnel: g.Kind == GatewayTunnel,
		})
	}
	return routes, nil
}

// HostnameRequests reads a zone's per-minute request counts for hostnames with the connected Cloudflare token.
func (s *Service) HostnameRequests(ctx context.Context, zoneID string, hostnames []string, from, to time.Time) ([]HostMinute, error) {
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	ap, ok := p.(AnalyticsProvider)
	if !ok {
		return nil, fmt.Errorf("%w: the dns provider reads no request counts", apperrs.ErrInvalid)
	}
	counts, err := ap.HostnameRequests(ctx, zoneID, hostnames, from, to)
	if err != nil {
		return nil, s.classify(err)
	}
	return counts, nil
}

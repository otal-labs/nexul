package dns

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// AdoptTunnelGatewayInput describes a cloudflared container found running on a machine: the tunnel its token
// connects, the adopted container row backing it, the docker networks it sits on (first is home), and the
// containers on that machine its ingress routes may target (container name → services.id).
type AdoptTunnelGatewayInput struct {
	TunnelID    string
	Machine     string
	ServiceID   string
	ServiceName string
	Networks    []string
	Targets     map[string]string
}

// AdoptedTunnelGateway is what adoption produced: the gateway, the hostnames now recorded as exposures, and
// the routed hostnames whose target is not a container Nexul tracks.
type AdoptedTunnelGateway struct {
	Gateway   *Gateway
	Exposed   []string
	Unmatched []string
}

// AdoptTunnelGateway takes a tunnel that already runs on a machine under Nexul's management without
// touching the provider's state: the tunnel row is filled from the provider (token included, encrypted at
// rest), the gateway backs onto the adopted cloudflared container, and every ingress route already pointing
// at a tracked container becomes an exposure so the canvas draws the hostname. Re-adoption is idempotent.
func (s *Service) AdoptTunnelGateway(ctx context.Context, in AdoptTunnelGatewayInput) (*AdoptedTunnelGateway, error) {
	if err := validateAdoptTunnelGatewayInput(in); err != nil {
		return nil, err
	}
	tp, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTunnelTracked(ctx, tp, in.TunnelID); err != nil {
		return nil, err
	}
	g, err := s.ensureAdoptedGateway(ctx, in)
	if err != nil {
		return nil, err
	}
	routes, err := tp.ListTunnelHostnames(ctx, in.TunnelID)
	if err != nil {
		return nil, s.classify(err)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	records, zoneNames, err := s.tunnelRecords(ctx, p, in.TunnelID)
	if err != nil {
		return nil, err
	}
	known, err := s.knownExposureHostnames(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	return s.reconcileAdoptedRoutes(ctx, g, in, routes, known, records, zoneNames)
}

// validateAdoptTunnelGatewayInput checks the fields AdoptTunnelGateway cannot proceed without.
func validateAdoptTunnelGatewayInput(in AdoptTunnelGatewayInput) error {
	if strings.TrimSpace(in.TunnelID) == "" || strings.TrimSpace(in.Machine) == "" || strings.TrimSpace(in.ServiceID) == "" {
		return fmt.Errorf("%w: tunnel id, machine and the backing container are required", apperrs.ErrInvalid)
	}
	if len(in.Networks) == 0 {
		return fmt.Errorf("%w: the gateway container is on no docker network", apperrs.ErrInvalid)
	}
	return nil
}

// knownExposureHostnames indexes gatewayID's existing exposures by hostname, so re-adoption can tell a
// route it already recorded from one it has not seen yet.
func (s *Service) knownExposureHostnames(ctx context.Context, gatewayID string) (map[string]bool, error) {
	existing, err := s.repo.ListExposuresByGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("list exposures for gateway %s: %w", gatewayID, err)
	}
	known := map[string]bool{}
	for _, e := range existing {
		known[e.Hostname] = true
	}
	return known, nil
}

// reconcileAdoptedRoutes walks the tunnel's live ingress routes: a route whose target Nexul does not track is
// unmatched, one already known is reported exposed as-is, and a new one is saved and reported exposed.
func (s *Service) reconcileAdoptedRoutes(ctx context.Context, g *Gateway, in AdoptTunnelGatewayInput, routes []TunnelRoute, known map[string]bool, records []Record, zoneNames map[string]string) (*AdoptedTunnelGateway, error) {
	out := &AdoptedTunnelGateway{Gateway: g, Exposed: []string{}, Unmatched: []string{}}
	for _, r := range routes {
		hostname := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(r.Hostname), "."))
		host, port := routeTarget(r.Service)
		serviceID, ok := in.Targets[host]
		if !ok {
			out.Unmatched = append(out.Unmatched, hostname)
			continue
		}
		if known[hostname] {
			out.Exposed = append(out.Exposed, hostname)
			continue
		}
		if err := s.saveAdoptedExposure(ctx, g, hostname, host, serviceID, port, records, zoneNames); err != nil {
			return nil, err
		}
		known[hostname] = true
		out.Exposed = append(out.Exposed, hostname)
	}
	return out, nil
}

// ensureTunnelTracked stores the tunnel row if this instance never created the tunnel, reading its token
// from the provider so a later rotate/route/delete works as for a tunnel created here.
func (s *Service) ensureTunnelTracked(ctx context.Context, tp TunnelProvider, tunnelID string) error {
	_, err := s.repo.GetTunnel(ctx, tunnelID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	live, err := tp.GetTunnel(ctx, tunnelID)
	if err != nil {
		return s.classify(err)
	}
	token, err := tp.TunnelToken(ctx, tunnelID)
	if err != nil {
		return s.classify(err)
	}
	enc, err := s.encryptTunnelToken(token)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	t := *live
	t.Token = enc
	t.CreatedAt = now
	t.UpdatedAt = now
	evt := s.tunnelEvent(TunnelChangedEvent{TunnelID: t.ID, Name: t.Name, Action: "created"})
	if err := s.repo.SaveTunnel(ctx, t, evt); err != nil {
		return fmt.Errorf("save adopted tunnel %s: %w", t.ID, err)
	}
	return nil
}

// ensureAdoptedGateway reuses the gateway already homed on the container's first network when it is this
// tunnel's, refuses another tunnel's, else creates one backed by the adopted container.
func (s *Service) ensureAdoptedGateway(ctx context.Context, in AdoptTunnelGatewayInput) (*Gateway, error) {
	home := in.Networks[0]
	now := s.now().UTC()
	g, err := s.repo.GetGatewayByNetwork(ctx, home)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("check gateway network %s: %w", home, err)
	}
	if err == nil {
		if g.TunnelID != in.TunnelID {
			return nil, fmt.Errorf("%w: network %q already has a gateway on another tunnel", apperrs.ErrConflict, home)
		}
		g.ServiceID, g.ServiceName, g.Machine, g.Networks, g.UpdatedAt = in.ServiceID, in.ServiceName, in.Machine, in.Networks, now
		evt := s.gatewayEvent(GatewayChangedEvent{GatewayID: g.ID, Kind: g.Kind, DockerNetwork: g.DockerNetwork, Action: "updated"})
		if err := s.repo.SaveGateway(ctx, *g, evt); err != nil {
			return nil, fmt.Errorf("save gateway %s: %w", g.ID, err)
		}
		return g, nil
	}
	g = &Gateway{
		ID: ids.New(), Kind: GatewayTunnel, DockerNetwork: home, Networks: in.Networks, Machine: in.Machine,
		ServiceID: in.ServiceID, ServiceName: in.ServiceName, TunnelID: in.TunnelID,
		CreatedAt: now, UpdatedAt: now,
	}
	evt := s.gatewayEvent(GatewayChangedEvent{GatewayID: g.ID, Kind: g.Kind, DockerNetwork: g.DockerNetwork, Action: "created"})
	if err := s.repo.SaveGateway(ctx, *g, evt); err != nil {
		return nil, fmt.Errorf("save gateway %s: %w", g.ID, err)
	}
	return g, nil
}

// saveAdoptedExposure records one already-routed hostname; its CNAME (when one exists) names the zone.
func (s *Service) saveAdoptedExposure(ctx context.Context, g *Gateway, hostname, service, serviceID string, port int, records []Record, zoneNames map[string]string) error {
	now := s.now().UTC()
	e := Exposure{
		ID: ids.New(), GatewayID: g.ID, Hostname: hostname, ServiceID: serviceID, Service: service, Port: port,
		CreatedAt: now, UpdatedAt: now,
	}
	for _, r := range records {
		if strings.EqualFold(r.Name, hostname) {
			e.ZoneID, e.Zone, e.RecordID = r.ZoneID, zoneNames[r.ZoneID], r.ID
			break
		}
	}
	evt := s.exposureEvent(ExposureChangedEvent{
		ExposureID: e.ID, GatewayID: g.ID, Hostname: hostname, Service: service, Port: port, Action: "created",
	})
	if err := s.repo.SaveExposure(ctx, e, evt); err != nil {
		return fmt.Errorf("save exposure %s: %w", e.ID, err)
	}
	return nil
}

// routeTarget splits an ingress service URL ("http://hello-api:3000") into the container name it addresses and
// the port, defaulting the port per scheme; a non-URL target (http_status:404) yields no host.
func routeTarget(service string) (string, int) {
	u, err := url.Parse(strings.TrimSpace(service))
	if err != nil || u.Hostname() == "" {
		return "", 0
	}
	if p, err := strconv.Atoi(u.Port()); err == nil {
		return u.Hostname(), p
	}
	if u.Scheme == "https" {
		return u.Hostname(), 443
	}
	return u.Hostname(), 80
}

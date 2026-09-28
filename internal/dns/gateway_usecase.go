package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// CreateGateway provisions a gateway's backing service, publishing gateway_changed; one gateway per network.
func (s *Service) CreateGateway(ctx context.Context, in CreateGatewayInput) (*Gateway, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	_, err := s.repo.GetGatewayByNetwork(ctx, in.DockerNetwork)
	if err == nil {
		return nil, fmt.Errorf("%w: network %q already has a gateway", apperrs.ErrConflict, in.DockerNetwork)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("check gateway network %s: %w", in.DockerNetwork, err)
	}

	name := strings.TrimSpace(in.Name)
	var serviceID string
	switch in.Kind {
	case GatewayTunnel:
		t, err := s.repo.GetTunnel(ctx, in.TunnelID)
		if err != nil {
			return nil, fmt.Errorf("get tunnel %s: %w", in.TunnelID, err)
		}
		if name == "" {
			name = "cloudflared-" + t.Name
		}
		provisioned, err := s.ProvisionTunnelAgent(ctx, in.TunnelID, AgentSpec{
			Target: in.Target, Name: name, Strategy: "run", DockerNetwork: in.DockerNetwork,
		})
		if err != nil {
			return nil, err
		}
		serviceID = provisioned.ServiceID
	case GatewayProxy:
		if name == "" {
			name = "traefik-" + ids.New()[:8]
		}
		spec := traefikSpec("", "", "")
		spec.Target, spec.Name, spec.DockerNetwork = in.Target, name, in.DockerNetwork
		provisioned, err := s.provisionProxy(ctx, spec)
		if err != nil {
			return nil, err
		}
		serviceID = provisioned.ServiceID
	}

	now := s.now().UTC()
	g := &Gateway{
		// Machine is Target: the backing service always deploys where the gateway itself runs.
		ID: ids.New(), Kind: in.Kind, DockerNetwork: in.DockerNetwork, Machine: in.Target,
		// The home network already counts as joined.
		Networks:  []string{in.DockerNetwork},
		ServiceID: serviceID, ServiceName: name, TunnelID: in.TunnelID,
		ZoneID: in.ZoneID, Zone: in.Zone, ServerAddress: in.ServerAddress,
		CreatedAt: now, UpdatedAt: now,
	}
	evt := s.gatewayEvent(GatewayChangedEvent{
		GatewayID: g.ID, Kind: g.Kind, DockerNetwork: g.DockerNetwork, Action: "created",
	})
	if err := s.repo.SaveGateway(ctx, *g, evt); err != nil {
		return nil, fmt.Errorf("save gateway %s: %w", g.ID, err)
	}
	return g, nil
}

// ListGateways returns every locally-tracked gateway, oldest first.
func (s *Service) ListGateways(ctx context.Context) ([]*Gateway, error) {
	gws, err := s.repo.ListGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("list gateways: %w", err)
	}
	return gws, nil
}

// GetGateway returns one locally-tracked gateway.
func (s *Service) GetGateway(ctx context.Context, gatewayID string) (*Gateway, error) {
	if strings.TrimSpace(gatewayID) == "" {
		return nil, fmt.Errorf("%w: gateway id is required", apperrs.ErrInvalid)
	}
	g, err := s.repo.GetGateway(ctx, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("get gateway %s: %w", gatewayID, err)
	}
	return g, nil
}

// GatewayForStackNetwork returns the container name of a gateway already connected to network (home or
// joined), so a stack's redeploy can rejoin it as the deploy's last step (spec §3); found is false when no
// gateway currently serves that network.
func (s *Service) GatewayForStackNetwork(ctx context.Context, network string) (containerName string, found bool, err error) {
	if strings.TrimSpace(network) == "" {
		return "", false, nil
	}
	gateways, err := s.repo.ListGateways(ctx)
	if err != nil {
		return "", false, fmt.Errorf("list gateways: %w", err)
	}
	for _, g := range gateways {
		if !g.hasNetwork(network) {
			continue
		}
		name := s.gatewayContainerName(ctx, g)
		if name == "" {
			continue
		}
		return name, true, nil
	}
	return "", false, nil
}

// GatewayForNetwork returns the gateway on a docker network, or ErrNotFound; deploy-side reads it to resolve routing.
func (s *Service) GatewayForNetwork(ctx context.Context, dockerNetwork string) (*Gateway, error) {
	if strings.TrimSpace(dockerNetwork) == "" {
		return nil, fmt.Errorf("%w: docker network is required", apperrs.ErrInvalid)
	}
	g, err := s.repo.GetGatewayByNetwork(ctx, dockerNetwork)
	if err != nil {
		return nil, fmt.Errorf("get gateway for network %s: %w", dockerNetwork, err)
	}
	return g, nil
}

// DeleteGateway deprovisions the backing service and drops the row; refuses a gateway that still has exposures.
func (s *Service) DeleteGateway(ctx context.Context, gatewayID string) error {
	if strings.TrimSpace(gatewayID) == "" {
		return fmt.Errorf("%w: gateway id is required", apperrs.ErrInvalid)
	}
	g, err := s.repo.GetGateway(ctx, gatewayID)
	if err != nil {
		return fmt.Errorf("get gateway %s: %w", gatewayID, err)
	}
	exposures, err := s.repo.ListExposuresByGateway(ctx, gatewayID)
	if err != nil {
		return fmt.Errorf("list exposures for gateway %s: %w", gatewayID, err)
	}
	if len(exposures) > 0 {
		return fmt.Errorf("%w: gateway %s still has %d exposure(s) — remove them first", apperrs.ErrConflict, gatewayID, len(exposures))
	}
	if g.ServiceID != "" {
		if s.provisioner == nil {
			return apperrs.Fatal(fmt.Errorf("%w: entry-path provisioning is not wired", apperrs.ErrInvalid))
		}
		if err := s.provisioner.Deprovision(ctx, g.ServiceID); err != nil {
			return fmt.Errorf("deprovision gateway %s: %w", gatewayID, err)
		}
	}
	evt := s.gatewayEvent(GatewayChangedEvent{
		GatewayID: g.ID, Kind: g.Kind, DockerNetwork: g.DockerNetwork, Action: "deleted",
	})
	if err := s.repo.DeleteGateway(ctx, gatewayID, evt); err != nil {
		return fmt.Errorf("delete gateway %s: %w", gatewayID, err)
	}
	return nil
}

func (s *Service) gatewayEvent(ev GatewayChangedEvent) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicGatewayChanged, Payload: ev}
}

// ProvisionInstanceProxy deploys the machine's one proxy gateway (reusing it when the machine has one) with Let's
// Encrypt and a route for the domain to this Nexul server; a retry redeploys the same stack.
func (s *Service) ProvisionInstanceProxy(ctx context.Context, in InstanceProxyInput) (*Gateway, error) {
	in, err := in.normalize()
	if err != nil {
		return nil, err
	}
	return s.deployProxyGateway(ctx, in)
}

// deployProxyGateway provisions Traefik for in (an empty domain routes nothing) and records it as a proxy gateway.
func (s *Service) deployProxyGateway(ctx context.Context, in InstanceProxyInput) (*Gateway, error) {
	if in.Domain != "" && s.origin == "" {
		return nil, apperrs.Fatal(fmt.Errorf("%w: the server's address for containers is not configured", apperrs.ErrInvalid))
	}
	machine, err := s.placementMachine(ctx, in.Target)
	if err != nil {
		return nil, err
	}
	g, err := s.proxyGatewayFor(ctx, machine, in.DockerNetwork)
	if err != nil {
		return nil, err
	}
	spec := traefikSpec(in.Domain, s.origin, in.Email)
	spec.Target, spec.Name, spec.DockerNetwork = machine, g.ServiceName, g.DockerNetwork
	provisioned, err := s.provisionProxy(ctx, spec)
	if err != nil {
		return nil, err
	}
	var evts []eventbus.OutboxEvent
	if g.ServiceID == "" {
		evts = append(evts, s.gatewayEvent(GatewayChangedEvent{GatewayID: g.ID, Kind: g.Kind, DockerNetwork: g.DockerNetwork, Action: "created"}))
	}
	g.ServiceID = provisioned.ServiceID
	if g.ServerAddress == "" && in.Domain != "" {
		g.ServerAddress = s.publicAddress(ctx, in.Domain)
	}
	g.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveGateway(ctx, *g, evts...); err != nil {
		return nil, fmt.Errorf("save gateway %s: %w", g.ID, err)
	}
	return g, nil
}

// proxyGatewayFor returns the machine's proxy gateway, or a new unsaved one homed on network (default nexul_proxy),
// so a machine never gets a second Traefik competing for ports 80 and 443.
func (s *Service) proxyGatewayFor(ctx context.Context, machine, network string) (*Gateway, error) {
	gws, err := s.repo.ListGatewaysByMachine(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("list gateways for machine %s: %w", machine, err)
	}
	for _, g := range gws {
		if g.Kind == GatewayProxy {
			return g, nil
		}
	}
	network = strings.TrimSpace(network)
	if network == "" {
		network = instanceProxyNetwork
	}
	_, err = s.repo.GetGatewayByNetwork(ctx, network)
	if err == nil {
		return nil, fmt.Errorf("%w: network %q already has a gateway; choose another docker network", apperrs.ErrConflict, network)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("check gateway network %s: %w", network, err)
	}
	now := s.now().UTC()
	return &Gateway{
		ID: ids.New(), Kind: GatewayProxy, DockerNetwork: network, Networks: []string{network},
		Machine: machine, ServiceName: instanceProxyStack, CreatedAt: now,
	}, nil
}

func (s *Service) placementMachine(ctx context.Context, target string) (string, error) {
	if t := strings.TrimSpace(target); t != "" {
		return t, nil
	}
	if s.placement == nil {
		return "", fmt.Errorf("%w: machine is required", apperrs.ErrInvalid)
	}
	machine, err := s.placement.InstanceMachine(ctx)
	if err != nil {
		return "", fmt.Errorf("find the instance machine: %w", err)
	}
	return machine, nil
}

func (s *Service) provisionProxy(ctx context.Context, spec AgentSpec) (*AgentProvisioned, error) {
	if s.provisioner == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: entry-path provisioning is not wired", apperrs.ErrInvalid))
	}
	provisioned, err := s.provisioner.Provision(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("provision reverse proxy: %w", err)
	}
	return provisioned, nil
}

// publicAddress is what the domain resolves to, IPv4 first, for the A/AAAA records later exposures create; empty
// when it does not resolve yet.
func (s *Service) publicAddress(ctx context.Context, domain string) string {
	addrs, err := s.ResolveHost(ctx, domain)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
			return a
		}
	}
	return addrs[0]
}

// resolveTimeout bounds one lookup, so a page polling resolve never hangs on a slow resolver.
const resolveTimeout = 5 * time.Second

// ResolveHost returns the sorted addresses host resolves to right now; a name that does not resolve yet is empty, not an error.
func (s *Service) ResolveHost(ctx context.Context, host string) ([]string, error) {
	domain, err := normalizeDomain(host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	addrs, err := s.resolver.LookupHost(ctx, domain)
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", domain, err)
	}
	slices.Sort(addrs)
	return slices.Compact(addrs), nil
}

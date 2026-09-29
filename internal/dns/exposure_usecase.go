package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// CreateExposure routes a hostname through a gateway to a container, publishing exposure_changed via the
// outbox. Without a gateway id, it reuses a gateway on the target's machine (preferring one already on
// one of its networks) or provisions one (spec §7). A retry converges: the same hostname and target return
// the saved exposure, and a DNS record an interrupted attempt left behind is reused.
func (s *Service) CreateExposure(ctx context.Context, in CreateExposureInput) (*Exposure, error) {
	if err := s.require(ctx, permissions.DNSWrite); err != nil {
		return nil, err
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	target, err := s.resolveExposureTarget(ctx, in)
	if err != nil {
		return nil, err
	}
	hostname := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Hostname), "."))
	saved, err := s.savedExposure(ctx, hostname, target.ContainerID, in.Port)
	if err != nil || saved != nil {
		return saved, err
	}
	g, err := s.resolveExposureGateway(ctx, in, target)
	if err != nil {
		return nil, err
	}
	if target.Machine != g.Machine {
		return nil, fmt.Errorf("%w: container %q runs on machine %q, gateway %s is on %q",
			apperrs.ErrInvalid, target.Name, target.Machine, g.ID, g.Machine)
	}
	if err := s.joinGatewayNetworks(ctx, g, target); err != nil {
		return nil, err
	}

	name, err := recordNameFor(hostname, in.Zone)
	if err != nil {
		return nil, err
	}
	dnsp, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}

	rec, err := s.createExposureRecord(ctx, g, in, target, hostname, name, dnsp)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	e := &Exposure{
		ID: ids.New(), GatewayID: g.ID, Hostname: hostname, ServiceID: target.ContainerID, Service: target.Name,
		Port: in.Port, ZoneID: in.ZoneID, Zone: in.Zone, RecordID: rec.ID, CreatedAt: now, UpdatedAt: now,
	}
	evt := s.exposureEvent(ExposureChangedEvent{
		ExposureID: e.ID, GatewayID: g.ID, Hostname: hostname, Service: target.Name, Port: in.Port, Action: "created",
	})
	if err := s.repo.SaveExposure(ctx, *e, evt); err != nil {
		return nil, fmt.Errorf("save exposure %s: %w", e.ID, err)
	}
	return e, nil
}

// savedExposure returns the exposure already routing hostname to this container and port, nil when the
// hostname is free, and a conflict when it routes somewhere else.
func (s *Service) savedExposure(ctx context.Context, hostname, containerID string, port int) (*Exposure, error) {
	exps, err := s.repo.ListExposures(ctx)
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	for _, e := range exps {
		if e.Hostname != hostname {
			continue
		}
		if e.ServiceID == containerID && e.Port == port {
			return e, nil
		}
		return nil, fmt.Errorf("%w: %s already routes to %s:%d; remove that exposure first", apperrs.ErrConflict, hostname, e.Service, e.Port)
	}
	return nil, nil
}

// resolveExposureTarget resolves the container an exposure targets: ServiceID directly, or the deprecated
// Service name as a stack lookup falling back to that stack's single container.
func (s *Service) resolveExposureTarget(ctx context.Context, in CreateExposureInput) (*ExposureTarget, error) {
	if s.containers == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: container lookup is not wired", apperrs.ErrInvalid))
	}
	if strings.TrimSpace(in.ServiceID) != "" {
		target, err := s.containers.ContainerByID(ctx, in.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("resolve container %s: %w", in.ServiceID, err)
		}
		return target, nil
	}
	target, err := s.containers.ContainerByStackName(ctx, in.Service)
	if err != nil {
		return nil, fmt.Errorf("resolve service %s: %w", in.Service, err)
	}
	return target, nil
}

// resolveExposureGateway honors an explicit gateway id, else reuses or provisions one on the target's machine.
func (s *Service) resolveExposureGateway(ctx context.Context, in CreateExposureInput, target *ExposureTarget) (*Gateway, error) {
	if strings.TrimSpace(in.GatewayID) != "" {
		g, err := s.repo.GetGateway(ctx, in.GatewayID)
		if err != nil {
			return nil, fmt.Errorf("get gateway %s: %w", in.GatewayID, err)
		}
		return g, nil
	}
	candidates, err := s.repo.ListGatewaysByMachine(ctx, target.Machine)
	if err != nil {
		return nil, fmt.Errorf("list gateways for machine %s: %w", target.Machine, err)
	}
	if g := pickPreferredGateway(candidates, target.Networks); g != nil {
		return g, nil
	}
	return s.provisionExposureGateway(ctx, in, target)
}

// pickPreferredGateway prefers a gateway already on one of the target's networks, else the machine's first.
func pickPreferredGateway(candidates []*Gateway, networks []string) *Gateway {
	if len(candidates) == 0 {
		return nil
	}
	for _, g := range candidates {
		for _, n := range networks {
			if g.hasNetwork(n) {
				return g
			}
		}
	}
	return candidates[0]
}

// provisionExposureGateway deploys a new gateway for the target's machine: the requested kind, or tunnel
// when Cloudflare is connected and a tunnel already exists, else proxy (spec §7).
func (s *Service) provisionExposureGateway(ctx context.Context, in CreateExposureInput, target *ExposureTarget) (*Gateway, error) {
	kind := in.Kind
	tunnelID := ""
	if kind == "" {
		kind, tunnelID = s.autoGatewayKind(ctx)
	}
	if kind == GatewayTunnel && tunnelID == "" {
		id, ok := s.defaultTunnelID(ctx)
		if !ok {
			return nil, fmt.Errorf("%w: no Cloudflare tunnel is set up to provision a tunnel gateway", apperrs.ErrInvalid)
		}
		tunnelID = id
	}
	if kind == GatewayTunnel {
		g, err := s.gatewayOnTunnelAgent(ctx, tunnelID, in, target.Machine)
		if err != nil || g != nil {
			return g, err
		}
	}
	serverAddress := ""
	if kind == GatewayProxy {
		addr, err := s.instanceServerAddress(ctx)
		if err != nil {
			return nil, err
		}
		serverAddress = addr
	}
	return s.CreateGateway(ctx, CreateGatewayInput{
		Kind: kind, DockerNetwork: firstNetwork(target.Networks),
		ZoneID: in.ZoneID, Zone: in.Zone, TunnelID: tunnelID, ServerAddress: serverAddress,
		Target: target.Machine,
	})
}

// gatewayOnTunnelAgent records the cloudflared the tunnel already runs on machine as the gateway, where
// provisioning would redeploy that agent and drop every request riding the tunnel, the browser's own included.
// Nil when the tunnel has no agent on machine.
func (s *Service) gatewayOnTunnelAgent(ctx context.Context, tunnelID string, in CreateExposureInput, machine string) (*Gateway, error) {
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	if t.AgentServiceID == "" || s.containers == nil {
		return nil, nil
	}
	agent, err := s.containers.ContainerByStackID(ctx, t.AgentServiceID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve tunnel agent %s: %w", t.AgentServiceID, err)
	}
	if agent.Machine != machine || len(agent.Networks) == 0 {
		return nil, nil
	}
	now := s.now().UTC()
	g := &Gateway{
		ID: ids.New(), Kind: GatewayTunnel, DockerNetwork: agent.Networks[0], Networks: agent.Networks, Machine: machine,
		ServiceID: t.AgentServiceID, ServiceName: agent.Name, TunnelID: tunnelID, ZoneID: in.ZoneID, Zone: in.Zone,
		CreatedAt: now, UpdatedAt: now,
	}
	evt := s.gatewayEvent(GatewayChangedEvent{GatewayID: g.ID, Kind: g.Kind, DockerNetwork: g.DockerNetwork, Action: "created"})
	if err := s.repo.SaveGateway(ctx, *g, evt); err != nil {
		return nil, fmt.Errorf("save gateway %s: %w", g.ID, err)
	}
	return g, nil
}

// autoGatewayKind defaults to tunnel when Cloudflare is connected and a tunnel already exists, else proxy.
func (s *Service) autoGatewayKind(ctx context.Context) (GatewayKind, string) {
	if s.cloudflareConnected(ctx) {
		if tunnelID, ok := s.defaultTunnelID(ctx); ok {
			return GatewayTunnel, tunnelID
		}
	}
	return GatewayProxy, ""
}

// defaultTunnelID picks the instance's tunnel to provision a new tunnel gateway against; there is normally
// exactly one, created during Cloudflare onboarding.
func (s *Service) defaultTunnelID(ctx context.Context) (string, bool) {
	tunnels, err := s.repo.ListTunnels(ctx)
	if err != nil || len(tunnels) == 0 {
		return "", false
	}
	return tunnels[0].ID, true
}

// instanceServerAddress defaults a new proxy gateway's server address to the instance's own configured host.
func (s *Service) instanceServerAddress(ctx context.Context) (string, error) {
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return "", err
	}
	return hostFromURL(instanceURL)
}

func firstNetwork(networks []string) string {
	if len(networks) == 0 {
		return ""
	}
	return networks[0]
}

// joinGatewayNetworks records target's networks into the gateway's Networks (set union) and, when the
// container is already running, asks a connected runner to join them immediately; a container that is not
// running yet is joined once its deploy reports healthy (ReconcileExposures).
func (s *Service) joinGatewayNetworks(ctx context.Context, g *Gateway, target *ExposureTarget) error {
	added := false
	for _, n := range target.Networks {
		if n != "" && !g.hasNetwork(n) {
			g.Networks = append(g.Networks, n)
			added = true
		}
	}
	if added {
		g.UpdatedAt = s.now().UTC()
		if err := s.repo.SaveGateway(ctx, *g); err != nil {
			return fmt.Errorf("record joined networks for gateway %s: %w", g.ID, err)
		}
	}
	if !target.Running || s.runnerJoin == nil || len(target.Networks) == 0 {
		return nil
	}
	gatewayContainer := s.gatewayContainerName(ctx, g)
	if gatewayContainer == "" {
		return nil
	}
	// Every network, recorded or not, since an earlier join may never have reached the runner; best-effort, as before.
	_ = s.runnerJoin.JoinNetworks(ctx, g.Machine, gatewayContainer, target.Networks)
	return nil
}

// ReconcileExposures settles the exposures of containers a deploy just reported running: the gateway joins
// their networks, and a tunnel route written before the container had a runtime name (the stack slug stood
// in) is re-routed to the name Docker gave it.
func (s *Service) ReconcileExposures(ctx context.Context, running []ExposureTarget) error {
	for i := range running {
		target := &running[i]
		exps, err := s.repo.ListExposuresByService(ctx, target.ContainerID)
		if err != nil {
			return fmt.Errorf("list exposures for container %s: %w", target.ContainerID, err)
		}
		for _, e := range exps {
			if err := s.reconcileExposure(ctx, e, target); err != nil {
				return fmt.Errorf("reconcile exposure %s: %w", e.Hostname, err)
			}
		}
	}
	return nil
}

func (s *Service) reconcileExposure(ctx context.Context, e *Exposure, target *ExposureTarget) error {
	g, err := s.repo.GetGateway(ctx, e.GatewayID)
	if err != nil {
		return fmt.Errorf("get gateway %s: %w", e.GatewayID, err)
	}
	if err := s.joinGatewayNetworks(ctx, g, target); err != nil {
		return err
	}
	if g.Kind != GatewayTunnel || target.Name == "" || e.Service == target.Name {
		return nil
	}
	tp, err := s.tunnelProviderFor(ctx, g.TunnelID)
	if err != nil {
		return err
	}
	if err := tp.RouteTunnelHostname(ctx, g.TunnelID, e.Hostname, tunnelOrigin(target.Name, e.Port)); err != nil {
		return s.classify(err)
	}
	e.Service, e.UpdatedAt = target.Name, s.now().UTC()
	if err := s.repo.SaveExposure(ctx, *e); err != nil {
		return fmt.Errorf("save exposure %s: %w", e.ID, err)
	}
	return nil
}

// gatewayContainerName resolves a gateway's own backing container name, for the join_networks request.
func (s *Service) gatewayContainerName(ctx context.Context, g *Gateway) string {
	if s.containers == nil || g.ServiceID == "" {
		return ""
	}
	container, err := s.containers.ContainerByStackID(ctx, g.ServiceID)
	if err != nil {
		return ""
	}
	return container.Name
}

// createExposureRecord creates the exposure's DNS record: a tunnel ingress rule + CNAME, or a proxy A/AAAA record.
func (s *Service) createExposureRecord(ctx context.Context, g *Gateway, in CreateExposureInput, target *ExposureTarget, hostname, name string, dnsp DNSProvider) (*Record, error) {
	switch g.Kind {
	case GatewayTunnel:
		if strings.TrimSpace(g.TunnelID) == "" {
			return nil, apperrs.Fatal(fmt.Errorf("%w: gateway %s has no tunnel reference", apperrs.ErrInvalid, g.ID))
		}
		tp, err := s.tunnelProviderFor(ctx, g.TunnelID)
		if err != nil {
			return nil, err
		}
		origin := tunnelOrigin(target.Name, in.Port)
		if err := tp.RouteTunnelHostname(ctx, g.TunnelID, hostname, origin); err != nil {
			return nil, s.classify(err)
		}
		return s.createOrReuseRecord(ctx, dnsp, in.ZoneID, hostname, RecordInput{
			Type: RecordCNAME, Name: name, Content: g.TunnelID + ".cfargotunnel.com", TTL: 1, Proxied: true,
		})
	case GatewayProxy:
		return s.createOrReuseRecord(ctx, dnsp, in.ZoneID, hostname, RecordInput{
			Type: aOrAAAA(g.ServerAddress), Name: name, Content: g.ServerAddress, TTL: 1,
		})
	default:
		return nil, apperrs.Fatal(fmt.Errorf("%w: unsupported gateway kind %q", apperrs.ErrInvalid, g.Kind))
	}
}

// createOrReuseRecord creates the record unless an interrupted attempt already left one with the same name,
// type and content; any other record on the name still fails with the provider's own message.
func (s *Service) createOrReuseRecord(ctx context.Context, dnsp DNSProvider, zoneID, hostname string, in RecordInput) (*Record, error) {
	records, err := dnsp.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, s.classify(err)
	}
	for i := range records {
		r := records[i]
		if strings.EqualFold(r.Name, hostname) && r.Type == in.Type && strings.EqualFold(r.Content, in.Content) {
			return &r, nil
		}
	}
	rec, err := dnsp.CreateRecord(ctx, zoneID, in)
	if err != nil {
		return nil, s.classify(err)
	}
	return rec, nil
}

// ListExposures returns every locally-tracked exposure, oldest first.
func (s *Service) ListExposures(ctx context.Context) ([]*Exposure, error) {
	if err := s.require(ctx, permissions.DNSRead); err != nil {
		return nil, err
	}
	exps, err := s.repo.ListExposures(ctx)
	if err != nil {
		return nil, fmt.Errorf("list exposures: %w", err)
	}
	return exps, nil
}

// GetExposure returns one locally-tracked exposure.
func (s *Service) GetExposure(ctx context.Context, exposureID string) (*Exposure, error) {
	if err := s.require(ctx, permissions.DNSRead); err != nil {
		return nil, err
	}
	if strings.TrimSpace(exposureID) == "" {
		return nil, fmt.Errorf("%w: exposure id is required", apperrs.ErrInvalid)
	}
	e, err := s.repo.GetExposure(ctx, exposureID)
	if err != nil {
		return nil, fmt.Errorf("get exposure %s: %w", exposureID, err)
	}
	return e, nil
}

// ExposuresForService returns the hostname+port of every exposure routed to a service by its legacy name,
// for the by-name alias route and the branch-deploy consumer.
func (s *Service) ExposuresForService(ctx context.Context, serviceName string) ([]ExposureSummary, error) {
	if err := s.require(ctx, permissions.DNSRead); err != nil {
		return nil, err
	}
	if strings.TrimSpace(serviceName) == "" {
		return nil, fmt.Errorf("%w: service is required", apperrs.ErrInvalid)
	}
	exps, err := s.repo.ListExposuresByServiceName(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("list exposures for service %s: %w", serviceName, err)
	}
	return exposureSummaries(exps), nil
}

// ExposuresForContainer returns the hostname+port of every exposure routed to a container by id.
func (s *Service) ExposuresForContainer(ctx context.Context, containerID string) ([]ExposureSummary, error) {
	if strings.TrimSpace(containerID) == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	exps, err := s.repo.ListExposuresByService(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("list exposures for container %s: %w", containerID, err)
	}
	return exposureSummaries(exps), nil
}

func exposureSummaries(exps []*Exposure) []ExposureSummary {
	out := make([]ExposureSummary, 0, len(exps))
	for _, e := range exps {
		out = append(out, ExposureSummary{Hostname: e.Hostname, Port: e.Port})
	}
	return out
}

// GetExposureByService returns a service's exposure by its legacy name, or ErrNotFound, so a repeated push
// can update, not duplicate (branch-deploy flow).
func (s *Service) GetExposureByService(ctx context.Context, serviceName string) (*Exposure, error) {
	if strings.TrimSpace(serviceName) == "" {
		return nil, fmt.Errorf("%w: service is required", apperrs.ErrInvalid)
	}
	exps, err := s.repo.ListExposuresByServiceName(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("get exposure for service %s: %w", serviceName, err)
	}
	if len(exps) == 0 {
		return nil, apperrs.ErrNotFound
	}
	return exps[0], nil
}

// DeleteExposure reverses CreateExposure: removes the ingress rule or A/AAAA record and drops the DNS record and row.
func (s *Service) DeleteExposure(ctx context.Context, exposureID string) error {
	if err := s.require(ctx, permissions.DNSDelete); err != nil {
		return err
	}
	return s.deleteExposure(ctx, exposureID)
}

func (s *Service) deleteExposure(ctx context.Context, exposureID string) error {
	if strings.TrimSpace(exposureID) == "" {
		return fmt.Errorf("%w: exposure id is required", apperrs.ErrInvalid)
	}
	e, err := s.repo.GetExposure(ctx, exposureID)
	if err != nil {
		return fmt.Errorf("get exposure %s: %w", exposureID, err)
	}
	g, err := s.repo.GetGateway(ctx, e.GatewayID)
	if err != nil {
		return fmt.Errorf("get gateway %s: %w", e.GatewayID, err)
	}
	if g.Kind == GatewayTunnel {
		tp, err := s.tunnelProviderFor(ctx, g.TunnelID)
		if err != nil {
			return err
		}
		if err := tp.RemoveTunnelHostname(ctx, g.TunnelID, e.Hostname); err != nil {
			return s.classify(err)
		}
	}
	if e.RecordID != "" {
		dnsp, err := s.providerFor(ctx)
		if err != nil {
			return err
		}
		if err := dnsp.DeleteRecord(ctx, e.ZoneID, e.RecordID); err != nil {
			return s.classify(err)
		}
	}
	evt := s.exposureEvent(ExposureChangedEvent{
		ExposureID: e.ID, GatewayID: e.GatewayID, Hostname: e.Hostname, Service: e.Service, Port: e.Port, Action: "deleted",
	})
	if err := s.repo.DeleteExposure(ctx, exposureID, evt); err != nil {
		return fmt.Errorf("delete exposure %s: %w", exposureID, err)
	}
	return nil
}

// DeleteExposuresForStack releases every hostname a stack holds before the stack itself is torn down. Both
// lookups are needed: a branch-deploy exposure still keys by the stack name (legacy), while a wizard/UI
// exposure keys by one of the stack's container ids (service_id) — the two spaces don't overlap by name alone.
func (s *Service) DeleteExposuresForStack(ctx context.Context, stackName string, containerIDs []string) error {
	byName, err := s.repo.ListExposuresByServiceName(ctx, stackName)
	if err != nil {
		return fmt.Errorf("list exposures for stack %s: %w", stackName, err)
	}
	seen := make(map[string]*Exposure, len(byName))
	for _, e := range byName {
		seen[e.ID] = e
	}
	for _, id := range containerIDs {
		exps, err := s.repo.ListExposuresByService(ctx, id)
		if err != nil {
			return fmt.Errorf("list exposures for container %s: %w", id, err)
		}
		for _, e := range exps {
			seen[e.ID] = e
		}
	}
	for _, e := range seen {
		if err := s.deleteExposure(ctx, e.ID); err != nil {
			return fmt.Errorf("release hostname %s: %w", e.Hostname, err)
		}
	}
	return nil
}

func (s *Service) exposureEvent(ev ExposureChangedEvent) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicExposureChanged, Payload: ev}
}

// aOrAAAA picks the record type for a server address: an IPv6 literal gets
// AAAA, everything else (IPv4, or a hostname the provider resolves) gets A.
func aOrAAAA(address string) RecordType {
	if ip := net.ParseIP(strings.TrimSpace(address)); ip != nil && ip.To4() == nil {
		return RecordAAAA
	}
	return RecordA
}

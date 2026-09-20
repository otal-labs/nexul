package dns

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
			ProjectID: in.ProjectID, Target: in.Target, Name: name,
			Strategy: "run", DockerNetwork: in.DockerNetwork,
		})
		if err != nil {
			return nil, err
		}
		serviceID = provisioned.ServiceID
	case GatewayProxy:
		if name == "" {
			name = "traefik-" + ids.New()[:8]
		}
		provisioned, err := s.ProvisionReverseProxy(ctx, AgentSpec{
			ProjectID: in.ProjectID, Target: in.Target, Name: name,
			Strategy: "run", DockerNetwork: in.DockerNetwork,
			Image:  traefikImage,
			Ports:  []string{"80:80", "443:443"},
			Mounts: []string{dockerSocketMount},
			Env: map[string]string{
				"TRAEFIK_PROVIDERS_DOCKER": "true",
				// exposedByDefault=false: Traefik sees every container on the machine's socket, so without this
				// one exposure would publish all of them; only a container an exposure labelled gets routed.
				"TRAEFIK_PROVIDERS_DOCKER_EXPOSEDBYDEFAULT": "false",
				"TRAEFIK_ENTRYPOINTS_WEB_ADDRESS":           ":80",
				"TRAEFIK_ENTRYPOINTS_WEBSECURE_ADDRESS":     ":443",
			},
		})
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

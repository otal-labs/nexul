package dns

import (
	"context"
	"encoding/json"
	"fmt"
)

// proxyRoutes is the file-provider fragment (http.routers and http.services) routing every exposure in exps through
// websecure to its container; the runner merges it into the gateway's own config.
func proxyRoutes(exps []*Exposure) string {
	routers := map[string]any{}
	services := map[string]any{}
	for _, e := range exps {
		name := RouterName(e.ID)
		routers[name] = map[string]any{
			"rule":        "Host(`" + e.Hostname + "`)",
			"entryPoints": []string{"websecure"},
			"service":     name,
			"tls":         map[string]string{"certResolver": certResolver},
		}
		services[name] = map[string]any{
			"loadBalancer": map[string]any{"servers": []map[string]string{{"url": tunnelOrigin(e.Service, e.Port)}}},
		}
	}
	// Maps of strings, string slices and ints cannot fail to marshal.
	b, _ := json.Marshal(map[string]any{"routers": routers, "services": services})
	return string(b)
}

// syncProxyRoutes sends a proxy gateway's current exposures to the machine's runner, which writes them into the running
// Traefik's file provider; a tunnel gateway, or one not deployed yet, has nothing to write.
func (s *Service) syncProxyRoutes(ctx context.Context, g *Gateway) error {
	if g.Kind != GatewayProxy {
		return nil
	}
	return s.sendProxyRoutes(ctx, g, s.gatewayContainerName(ctx, g))
}

// SyncMachineRoutes re-sends the routes of every proxy gateway on machine, for a runner that just connected: pushes sent
// while it was away reached nobody. A machine with only tunnel gateways gets nothing.
func (s *Service) SyncMachineRoutes(ctx context.Context, machine string) error {
	gateways, err := s.repo.ListGatewaysByMachine(ctx, machine)
	if err != nil {
		return fmt.Errorf("list gateways for machine %s: %w", machine, err)
	}
	for _, g := range gateways {
		if err := s.syncProxyRoutes(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

// syncGatewayOf re-sends the routes of the proxy gateway whose backing stack is target's, after a deploy replaced its
// container with one that carries only the base config; the container is the one the deploy just reported.
func (s *Service) syncGatewayOf(ctx context.Context, target *ExposureTarget) error {
	gateways, err := s.repo.ListGateways(ctx)
	if err != nil {
		return fmt.Errorf("list gateways: %w", err)
	}
	for _, g := range gateways {
		if g.Kind == GatewayProxy && g.ServiceID == target.StackID {
			return s.sendProxyRoutes(ctx, g, target.Name)
		}
	}
	return nil
}

func (s *Service) sendProxyRoutes(ctx context.Context, g *Gateway, container string) error {
	if s.runnerJoin == nil || container == "" {
		return nil
	}
	exps, err := s.repo.ListExposuresByGateway(ctx, g.ID)
	if err != nil {
		return fmt.Errorf("list exposures for gateway %s: %w", g.ID, err)
	}
	// A runner that is not connected is no error, as for the network join: the runner's next connect sends the routes again.
	_ = s.runnerJoin.SetRoutes(ctx, g.Machine, container, proxyRoutes(exps))
	return nil
}

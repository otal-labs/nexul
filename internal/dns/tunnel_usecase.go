package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// CreateTunnel creates a tunnel, stores it with its token encrypted at rest, publishing tunnel_changed.
func (s *Service) CreateTunnel(ctx context.Context, in CreateTunnelInput) (*Tunnel, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	// A retried onboarding step reuses the tunnel it already created instead of piling up duplicates at Cloudflare.
	existing, err := s.repo.ListTunnels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tunnels: %w", err)
	}
	for _, t := range existing {
		if t.Name == strings.TrimSpace(in.Name) {
			return t, nil
		}
	}
	p, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	t, err := p.CreateTunnel(ctx, strings.TrimSpace(in.Name))
	if err != nil {
		return nil, s.classify(err)
	}
	enc, err := s.encryptTunnelToken(t.Token)
	if err != nil {
		return nil, err
	}
	t.Token = enc
	t.CreatedAt = s.now().UTC()
	t.UpdatedAt = t.CreatedAt
	evt := s.tunnelEvent(TunnelChangedEvent{TunnelID: t.ID, Name: t.Name, Action: "created"})
	if err := s.repo.SaveTunnel(ctx, *t, evt); err != nil {
		return nil, fmt.Errorf("save tunnel %s: %w", t.ID, err)
	}
	return t, nil
}

// DescribeTunnel reads a tunnel the machine scan found running (its id came off the container's token) back
// from the provider: name and status, its routed hostnames, and every CNAME across the token's zones that
// targets it. Nothing is stored; Tracked says whether this instance already owns the tunnel.
func (s *Service) DescribeTunnel(ctx context.Context, tunnelID string) (*TunnelInfo, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return nil, fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	tp, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	t, err := tp.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, s.classify(err)
	}
	routes, err := tp.ListTunnelHostnames(ctx, tunnelID)
	if err != nil {
		return nil, s.classify(err)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	records, _, err := s.tunnelRecords(ctx, p, tunnelID)
	if err != nil {
		return nil, err
	}
	tracked := false
	if s.repo != nil {
		if _, err := s.repo.GetTunnel(ctx, tunnelID); err == nil {
			tracked = true
		}
	}
	if routes == nil {
		routes = []TunnelRoute{}
	}
	return &TunnelInfo{ID: t.ID, Name: t.Name, Status: t.Status, Tracked: tracked, Routes: routes, Records: records}, nil
}

// tunnelRecords finds every CNAME across the token's zones that targets the tunnel, plus zone id → name.
func (s *Service) tunnelRecords(ctx context.Context, p DNSProvider, tunnelID string) ([]Record, map[string]string, error) {
	zones, err := p.ListZones(ctx)
	if err != nil {
		return nil, nil, s.classify(err)
	}
	target := tunnelID + ".cfargotunnel.com"
	records := []Record{}
	names := make(map[string]string, len(zones))
	for _, z := range zones {
		names[z.ID] = z.Name
		recs, err := p.ListRecords(ctx, z.ID)
		if err != nil {
			return nil, nil, s.classify(err)
		}
		for _, r := range recs {
			if r.Type == RecordCNAME && r.Content == target {
				records = append(records, r)
			}
		}
	}
	return records, names, nil
}

// ListTunnels returns every locally-tracked tunnel, oldest first.
func (s *Service) ListTunnels(ctx context.Context) ([]*Tunnel, error) {
	tunnels, err := s.repo.ListTunnels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tunnels: %w", err)
	}
	return tunnels, nil
}

// GetTunnel returns one locally-tracked tunnel.
func (s *Service) GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return nil, fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	return t, nil
}

// RouteTunnelHostname adds the ingress rule and creates the CNAME pointing the hostname at the tunnel.
func (s *Service) RouteTunnelHostname(ctx context.Context, in RouteTunnelInput) (*Tunnel, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	p, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.RouteTunnelHostname(ctx, in.TunnelID, in.Hostname, in.Service); err != nil {
		return nil, s.classify(err)
	}
	dnsp, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	name, err := recordNameFor(in.Hostname, in.Zone)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.GetTunnel(ctx, in.TunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", in.TunnelID, err)
	}
	// A CNAME to *.cfargotunnel.com only resolves through Cloudflare's proxy; DNS-only leaves the hostname dead.
	input := RecordInput{Type: RecordCNAME, Name: name, Content: in.TunnelID + ".cfargotunnel.com", TTL: 1, Proxied: true}
	var rec *Record
	if t.RecordID != "" && t.ZoneID == in.ZoneID {
		// Re-pointing the same tunnel updates its record in place rather than colliding on the name.
		rec, err = dnsp.UpdateRecord(ctx, in.ZoneID, t.RecordID, input)
	}
	if t.RecordID == "" || t.ZoneID != in.ZoneID {
		rec, err = dnsp.CreateRecord(ctx, in.ZoneID, input)
	}
	if err != nil {
		return nil, s.classify(err)
	}
	t.Hostname = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Hostname), "."))
	t.ZoneID = in.ZoneID
	t.Zone = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Zone), "."))
	t.RecordID = rec.ID
	t.Service = in.Service
	t.UpdatedAt = s.now().UTC()
	evt := s.tunnelEvent(TunnelChangedEvent{
		TunnelID: t.ID, Name: t.Name, Action: "routed", Hostname: t.Hostname, Service: t.Service,
	})
	if err := s.repo.SaveTunnel(ctx, *t, evt); err != nil {
		return nil, fmt.Errorf("save tunnel route %s: %w", t.ID, err)
	}
	s.recordChanged(ctx, RecordChangedEvent{
		ZoneID: in.ZoneID, Zone: in.Zone, RecordID: rec.ID, Action: "created",
		Type: rec.Type, Name: rec.Name, Service: "tunnel:" + in.TunnelID,
	})
	return t, nil
}

// RotateTunnelCredentials issues a fresh encrypted token; cloudflared stays up until the agent redeploys.
func (s *Service) RotateTunnelCredentials(ctx context.Context, tunnelID string) (*Tunnel, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return nil, fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	p, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	token, err := p.RotateTunnelCredentials(ctx, tunnelID)
	if err != nil {
		return nil, s.classify(err)
	}
	enc, err := s.encryptTunnelToken(token)
	if err != nil {
		return nil, err
	}
	t.Token = enc
	t.UpdatedAt = s.now().UTC()
	evt := s.tunnelEvent(TunnelChangedEvent{TunnelID: t.ID, Name: t.Name, Action: "rotated"})
	if err := s.repo.SaveTunnel(ctx, *t, evt); err != nil {
		return nil, fmt.Errorf("save rotated tunnel %s: %w", t.ID, err)
	}
	return t, nil
}

// DeleteTunnel removes the tunnel at the provider (idempotent) and drops the
// local row, publishing dns.tunnel_changed (deleted).
func (s *Service) DeleteTunnel(ctx context.Context, tunnelID string) error {
	if strings.TrimSpace(tunnelID) == "" {
		return fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	p, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return err
	}
	if err := p.DeleteTunnel(ctx, tunnelID); err != nil {
		return s.classify(err)
	}
	evt := s.tunnelEvent(TunnelChangedEvent{TunnelID: t.ID, Name: t.Name, Action: "deleted"})
	if err := s.repo.DeleteTunnel(ctx, tunnelID, evt); err != nil {
		return fmt.Errorf("delete tunnel %s: %w", tunnelID, err)
	}
	return nil
}

// cloudflaredImage is the pinned tunnel-agent image; the token in env selects the tunnel.
const cloudflaredImage = "cloudflare/cloudflared:latest"

// cloudflaredHealthURL is cloudflared's readiness endpoint on the metrics port the command below opens.
const cloudflaredHealthURL = "http://localhost:20241/ready"

func cloudflaredCommand() []string {
	return []string{"tunnel", "--no-autoupdate", "--metrics", "0.0.0.0:20241", "run"}
}

// TunnelStatus asks the provider for the tunnel's live connector state ("healthy" once cloudflared is connected).
func (s *Service) TunnelStatus(ctx context.Context, tunnelID string) (*Tunnel, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return nil, fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	p, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	live, err := p.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, s.classify(err)
	}
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	t.Status = live.Status
	t.Token = ""
	return t, nil
}

// ProvisionTunnelAgent provisions the cloudflared service, feeding it the token, and records the service id.
func (s *Service) ProvisionTunnelAgent(ctx context.Context, tunnelID string, spec AgentSpec) (*AgentProvisioned, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return nil, fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	if s.provisioner == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: entry-path provisioning is not wired", apperrs.ErrInvalid))
	}
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	token, err := s.decryptTunnelToken(t.Token)
	if err != nil {
		return nil, err
	}
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		spec.Name = "cloudflared-" + t.Name
	}
	// Without an image nothing is ever deployed, and without `tunnel run` the image prints help and exits.
	if spec.Image == "" {
		spec.Image = cloudflaredImage
	}
	if len(spec.Command) == 0 {
		spec.Command = cloudflaredCommand()
	}
	if spec.Strategy == "" {
		spec.Strategy = "run"
	}
	// Every service definition needs a health URL; cloudflared's /ready answers once the tunnel is connected.
	if spec.HealthURL == "" {
		spec.HealthURL = cloudflaredHealthURL
	}
	if spec.Env == nil {
		spec.Env = map[string]string{}
	}
	spec.Env["TUNNEL_TOKEN"] = token
	provisioned, err := s.provisioner.Provision(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("provision tunnel agent %s: %w", tunnelID, err)
	}
	t.AgentServiceID = provisioned.ServiceID
	t.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveTunnel(ctx, *t); err != nil {
		return nil, fmt.Errorf("save tunnel agent %s: %w", tunnelID, err)
	}
	return provisioned, nil
}

// ProvisionReverseProxy provisions a reverse-proxy service definition (e.g. Traefik) as a Nexul service.
func (s *Service) ProvisionReverseProxy(ctx context.Context, spec AgentSpec) (*AgentProvisioned, error) {
	if s.provisioner == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: entry-path provisioning is not wired", apperrs.ErrInvalid))
	}
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		spec.Name = "reverse-proxy"
	}
	if spec.HealthURL == "" {
		spec.HealthURL = "http://localhost:80/"
	}
	provisioned, err := s.provisioner.Provision(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("provision reverse proxy: %w", err)
	}
	return provisioned, nil
}

// tunnelProviderFor resolves the tunnel provider; tunnel ops also need the account-level Tunnel permission.
func (s *Service) tunnelProviderFor(ctx context.Context) (TunnelProvider, error) {
	if s.tunnel != nil {
		return s.tunnel, nil
	}
	token, err := s.resolveToken(ctx)
	if err != nil {
		return nil, err
	}
	if s.newTunnel == nil {
		return nil, fmt.Errorf("%w: dns tunnel provider constructor is not wired", apperrs.ErrInvalid)
	}
	p, err := s.newTunnel(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("build dns tunnel provider: %w", err)
	}
	return p, nil
}

// encryptTunnelToken seals a tunnel token at rest.
func (s *Service) encryptTunnelToken(token string) (string, error) {
	if len(s.key) == 0 {
		return "", fmt.Errorf("%w: dns credentials encryption key is not configured", apperrs.ErrInvalid)
	}
	enc, err := crypto.Encrypt(s.key, []byte(token))
	if err != nil {
		return "", fmt.Errorf("encrypt tunnel token: %w", err)
	}
	return enc, nil
}

// decryptTunnelToken opens a stored tunnel token.
func (s *Service) decryptTunnelToken(token string) (string, error) {
	if len(s.key) == 0 {
		return "", apperrs.Fatal(fmt.Errorf("%w: dns credentials encryption key is not configured", apperrs.ErrFatal))
	}
	raw, err := crypto.Decrypt(s.key, token)
	if err != nil {
		return "", apperrs.Fatal(fmt.Errorf("%w: stored tunnel token cannot be decrypted (auth secret changed?)", apperrs.ErrFatal))
	}
	return string(raw), nil
}

func (s *Service) tunnelEvent(ev TunnelChangedEvent) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTunnelChanged, Payload: ev}
}

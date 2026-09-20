package dns

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// SettingsReader is the consumer-side view of the instance settings dns needs for the wizard hook (ADR 0017).
type SettingsReader interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// TokenProvider is the seam for the live Cloudflare token; dns must never import connectors (ADR 0017).
type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

// Config wires the dns use-cases.
type Config struct {
	// Repo persists tunnels and hostname associations.
	Repo Repo
	// Provider is a fixed provider for tests; when set, NewProvider is ignored.
	Provider DNSProvider
	// NewProvider builds a provider for a freshly-resolved access token; required unless Provider is set.
	NewProvider func(ctx context.Context, token string) (DNSProvider, error)
	// TunnelProvider is a fixed tunnel provider for tests; when set, NewTunnelProvider is ignored.
	TunnelProvider TunnelProvider
	// NewTunnelProvider builds a tunnel-capable provider for a freshly-resolved access token.
	NewTunnelProvider func(ctx context.Context, token string) (TunnelProvider, error)
	// Provisioner creates service definitions for entry-path agents; nil makes provisioning use-cases fail fatally.
	Provisioner ServiceProvisioner
	// Containers resolves the container an exposure targets, or a gateway's own backing container.
	Containers ContainerLookup
	// RunnerJoin asks a connected runner to join a gateway container onto networks immediately; nil skips
	// the immediate join and leaves it for the stack's next deploy.
	RunnerJoin RunnerJoiner
	// Tokens resolves the live Cloudflare token from connectors; required unless Provider/TunnelProvider are fixed.
	Tokens TokenProvider
	// EncryptionKey seals stored tunnel tokens at rest.
	EncryptionKey []byte
	// Settings feeds the wizard hook's instance record creation.
	Settings SettingsReader
	// Now overridable for tests.
	Now func() time.Time
}

// Service's Cloudflare token is resolved live via TokenProvider, never stored.
type Service struct {
	repo        Repo
	provider    DNSProvider
	newProvider func(ctx context.Context, token string) (DNSProvider, error)
	tunnel      TunnelProvider
	newTunnel   func(ctx context.Context, token string) (TunnelProvider, error)
	provisioner ServiceProvisioner
	containers  ContainerLookup
	runnerJoin  RunnerJoiner
	tokens      TokenProvider
	key         []byte
	settings    SettingsReader
	now         func() time.Time
}

// NewService wires the dns use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		repo:        cfg.Repo,
		provider:    cfg.Provider,
		newProvider: cfg.NewProvider,
		tunnel:      cfg.TunnelProvider,
		newTunnel:   cfg.NewTunnelProvider,
		provisioner: cfg.Provisioner,
		containers:  cfg.Containers,
		runnerJoin:  cfg.RunnerJoin,
		tokens:      cfg.Tokens,
		key:         cfg.EncryptionKey,
		settings:    cfg.Settings,
		now:         cfg.Now,
	}
}

// SetRunnerJoin wires the runner seam for an immediate network join on a running stack's exposure; unset,
// CreateExposure only records the join in gateways.networks and leaves the runner side to the next deploy.
func (s *Service) SetRunnerJoin(j RunnerJoiner) { s.runnerJoin = j }

// VerifyCredentials checks the current credentials against the provider,
// surfacing bad tokens as fatal.
func (s *Service) VerifyCredentials(ctx context.Context) error {
	p, err := s.providerFor(ctx)
	if err != nil {
		return err
	}
	if err := p.Verify(ctx); err != nil {
		return s.classify(err)
	}
	return nil
}

// ListZones returns the zones the connected credentials can edit.
func (s *Service) ListZones(ctx context.Context) ([]Zone, error) {
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	zones, err := p.ListZones(ctx)
	if err != nil {
		return nil, s.classify(err)
	}
	return zones, nil
}

// ListRecords returns the records in a zone.
func (s *Service) ListRecords(ctx context.Context, zoneID string) ([]Record, error) {
	if strings.TrimSpace(zoneID) == "" {
		return nil, fmt.Errorf("%w: zone id is required", apperrs.ErrInvalid)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	records, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, s.classify(err)
	}
	return records, nil
}

// CreateRecord creates a record at the provider and publishes dns.record_changed.
func (s *Service) CreateRecord(ctx context.Context, zoneID string, in RecordInput) (*Record, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(zoneID) == "" {
		return nil, fmt.Errorf("%w: zone id is required", apperrs.ErrInvalid)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := p.CreateRecord(ctx, zoneID, in)
	if err != nil {
		return nil, s.classify(err)
	}
	s.recordChanged(ctx, RecordChangedEvent{
		ZoneID: zoneID, RecordID: rec.ID, Action: "created", Type: rec.Type, Name: rec.Name,
	})
	return rec, nil
}

// UpdateRecord replaces a record's content at the provider.
func (s *Service) UpdateRecord(ctx context.Context, zoneID, recordID string, in RecordInput) (*Record, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(zoneID) == "" || strings.TrimSpace(recordID) == "" {
		return nil, fmt.Errorf("%w: zone id and record id are required", apperrs.ErrInvalid)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := p.UpdateRecord(ctx, zoneID, recordID, in)
	if err != nil {
		return nil, s.classify(err)
	}
	s.recordChanged(ctx, RecordChangedEvent{
		ZoneID: zoneID, RecordID: recordID, Action: "updated", Type: rec.Type, Name: rec.Name,
	})
	return rec, nil
}

// DeleteRecord removes a record at the provider (idempotent).
func (s *Service) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if strings.TrimSpace(zoneID) == "" || strings.TrimSpace(recordID) == "" {
		return fmt.Errorf("%w: zone id and record id are required", apperrs.ErrInvalid)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return err
	}
	if err := p.DeleteRecord(ctx, zoneID, recordID); err != nil {
		return s.classify(err)
	}
	s.recordChanged(ctx, RecordChangedEvent{ZoneID: zoneID, RecordID: recordID, Action: "deleted"})
	return nil
}

// CheckPropagation verifies a record has propagated to public DNS.
func (s *Service) CheckPropagation(ctx context.Context, zoneID string, rec Record) error {
	if strings.TrimSpace(zoneID) == "" || rec.ID == "" {
		return fmt.Errorf("%w: zone id and record are required", apperrs.ErrInvalid)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return err
	}
	if err := p.CheckPropagation(ctx, zoneID, rec); err != nil {
		return s.classify(err)
	}
	return nil
}

// CreateInstanceRecord creates the record from the saved URL; host/zone mismatch surfaces, never guessed.
func (s *Service) CreateInstanceRecord(ctx context.Context, zoneID, zone string, recType RecordType, target string) (*Record, error) {
	if strings.TrimSpace(zoneID) == "" || strings.TrimSpace(zone) == "" {
		return nil, fmt.Errorf("%w: zone is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("%w: target is required (server address or tunnel hostname)", apperrs.ErrInvalid)
	}
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return nil, err
	}
	host, err := hostFromURL(instanceURL)
	if err != nil {
		return nil, err
	}
	name, err := recordNameFor(host, zone)
	if err != nil {
		return nil, err
	}
	return s.CreateRecord(ctx, zoneID, RecordInput{Type: recType, Name: name, Content: target, TTL: 1})
}

// SetServiceHostname associates a hostname with a service, creating the provider record in one transaction.
func (s *Service) SetServiceHostname(ctx context.Context, in ServiceHostnameInput) (*ServiceHostname, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	in.Service = strings.TrimSpace(in.Service)
	name, err := recordNameFor(in.Hostname, in.Zone)
	if err != nil {
		return nil, err
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := p.CreateRecord(ctx, in.ZoneID, RecordInput{Type: in.Type, Name: name, Content: in.Target, TTL: 1})
	if err != nil {
		return nil, s.classify(err)
	}
	sh := &ServiceHostname{
		Service:   in.Service,
		Hostname:  strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Hostname), ".")),
		ZoneID:    in.ZoneID,
		Zone:      strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Zone), ".")),
		RecordID:  rec.ID,
		Type:      rec.Type,
		Content:   rec.Content,
		CreatedAt: s.now().UTC(),
	}
	evt := s.recordEvent(RecordChangedEvent{
		ZoneID: in.ZoneID, Zone: in.Zone, RecordID: rec.ID, Action: "created",
		Type: rec.Type, Name: rec.Name, Service: in.Service,
	})
	if err := s.repo.UpsertServiceHostname(ctx, *sh, evt); err != nil {
		return nil, fmt.Errorf("save service hostname: %w", err)
	}
	return sh, nil
}

// GetServiceHostname returns one service's hostname association.
func (s *Service) GetServiceHostname(ctx context.Context, service string) (*ServiceHostname, error) {
	if strings.TrimSpace(service) == "" {
		return nil, fmt.Errorf("%w: service is required", apperrs.ErrInvalid)
	}
	sh, err := s.repo.GetServiceHostname(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("get service hostname %s: %w", service, err)
	}
	return sh, nil
}

// ListServiceHostnames returns every hostname association, oldest first.
func (s *Service) ListServiceHostnames(ctx context.Context) ([]*ServiceHostname, error) {
	shs, err := s.repo.ListServiceHostnames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list service hostnames: %w", err)
	}
	return shs, nil
}

// RemoveServiceHostname deletes the provider record and the local association.
func (s *Service) RemoveServiceHostname(ctx context.Context, service string) error {
	if strings.TrimSpace(service) == "" {
		return fmt.Errorf("%w: service is required", apperrs.ErrInvalid)
	}
	sh, err := s.repo.GetServiceHostname(ctx, service)
	if err != nil {
		return fmt.Errorf("get service hostname %s: %w", service, err)
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return err
	}
	if err := p.DeleteRecord(ctx, sh.ZoneID, sh.RecordID); err != nil {
		return s.classify(err)
	}
	evt := s.recordEvent(RecordChangedEvent{
		ZoneID: sh.ZoneID, Zone: sh.Zone, RecordID: sh.RecordID, Action: "deleted", Service: service,
	})
	if err := s.repo.DeleteServiceHostname(ctx, service, evt); err != nil {
		return fmt.Errorf("delete service hostname %s: %w", service, err)
	}
	return nil
}

// providerFor treats a bad credential as a fatal config error, not retryable.
func (s *Service) providerFor(ctx context.Context) (DNSProvider, error) {
	if s.provider != nil {
		return s.provider, nil
	}
	token, err := s.resolveToken(ctx)
	if err != nil {
		return nil, err
	}
	if s.newProvider == nil {
		return nil, fmt.Errorf("%w: dns provider constructor is not wired", apperrs.ErrInvalid)
	}
	p, err := s.newProvider(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("build dns provider: %w", err)
	}
	return p, nil
}

// resolveToken returns the live Cloudflare token; a bad token is fatal, since a 401 would sign the user out.
func (s *Service) resolveToken(ctx context.Context) (string, error) {
	if s.tokens == nil {
		return "", apperrs.Fatal(fmt.Errorf("%w: dns provider token source is not wired", apperrs.ErrInvalid))
	}
	token, err := s.tokens.Token(ctx)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			// Not an auth failure: the workspace simply hasn't connected Cloudflare yet.
			return "", apperrs.Fatal(fmt.Errorf(
				"%w: Cloudflare is not connected — connect it in Settings", apperrs.ErrRetryable))
		}
		return "", apperrs.Fatal(fmt.Errorf("%w: resolve cloudflare access token: %v", apperrs.ErrRetryable, err))
	}
	return token, nil
}

// cloudflareConnected reports whether a Cloudflare token is resolvable, the signal CreateExposure uses to
// default an auto-provisioned gateway to tunnel (connected) or proxy (not connected).
func (s *Service) cloudflareConnected(ctx context.Context) bool {
	if s.tokens == nil {
		return false
	}
	_, err := s.tokens.Token(ctx)
	return err == nil
}

// classify maps a bad token or missing zone to fatal (never retried);
// provider API failures stay retryable.
func (s *Service) classify(err error) error {
	switch {
	case errors.Is(err, apperrs.ErrUnauthorized), errors.Is(err, apperrs.ErrForbidden), errors.Is(err, apperrs.ErrNotFound):
		return apperrs.Fatal(err)
	case errors.Is(err, apperrs.ErrRetryable):
		return err
	default:
		return err
	}
}

// recordChanged publishes a provider record change that has no local row via
// the outbox, best-effort.
func (s *Service) recordChanged(ctx context.Context, ev RecordChangedEvent) {
	if s.repo == nil {
		return
	}
	_ = s.repo.RecordChanged(ctx, s.recordEvent(ev))
}

func (s *Service) recordEvent(ev RecordChangedEvent) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicRecordChanged, Payload: ev}
}

func (s *Service) instanceURL(ctx context.Context) (string, error) {
	if s.settings == nil {
		return "", fmt.Errorf("%w: instance settings reader is not wired", apperrs.ErrInvalid)
	}
	u, err := s.settings.GetInstanceURL(ctx)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return "", fmt.Errorf("%w: set the instance URL (owner wizard) before creating the instance record", apperrs.ErrInvalid)
		}
		return "", fmt.Errorf("get instance url: %w", err)
	}
	return u, nil
}

// hostFromURL extracts the host from an instance URL.
func hostFromURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("%w: instance URL %q is not a valid http(s) URL", apperrs.ErrInvalid, raw)
	}
	return u.Hostname(), nil
}

package integrations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// tokenPrefix marks a scoped token so the gateway tells it from a session token or PAT without verifying all three.
const tokenPrefix = "dit_"

// OwnerGate resolves can_create_workspace; a consumer-side seam so this package never imports auth (ADR 0017).
type OwnerGate interface {
	CanCreateWorkspace(ctx context.Context, userID string) (bool, error)
}

// Config wires the integrations service.
type Config struct {
	Installs   InstallStore
	Tokens     TokenStore
	Subs       SubscriptionStore
	Deliveries DeliveryStore
	Schemas    SchemaStore
	Audit      AuditStore
	Owner      OwnerGate
	Now        func() time.Time
}

// Service's installs mint tokens; the authenticator gates the HTTP surface (ADR 0019).
type Service struct {
	cfg Config
}

// NewService wires the integrations use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{cfg: cfg}
}

// Install mints a scoped token for a new install (ADR 0043), returning the raw token and webhook secret exactly once.
func (s *Service) Install(ctx context.Context, createdBy, name string, tier TrustTier, webhookURL string, requested []Scope) (rawToken, webhookSecret string, install *Install, err error) {
	if err := s.requireOwner(ctx, createdBy); err != nil {
		return "", "", nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", nil, fmt.Errorf("%w: install name is required", apperrs.ErrInvalid)
	}
	switch tier {
	case TrustVerified, TrustCommunity:
	default:
		return "", "", nil, fmt.Errorf("%w: trust tier must be verified or community", apperrs.ErrInvalid)
	}
	if webhookURL != "" {
		if err := validateWebhookURL(webhookURL); err != nil {
			return "", "", nil, err
		}
	}
	scopes, err := ParseScopes(scopeStrings(requested))
	if err != nil {
		return "", "", nil, err
	}
	effective := ExpandScopes(scopes)

	now := s.cfg.Now().UTC()
	secret, err := newWebhookSecret()
	if err != nil {
		return "", "", nil, err
	}
	install = &Install{
		ID:            ids.New(),
		Name:          name,
		TrustTier:     tier,
		WebhookURL:    strings.TrimRight(webhookURL, "/"),
		WebhookSecret: secret,
		Scopes:        effective,
		CreatedBy:     createdBy,
		CreatedAt:     now,
	}
	if err := s.cfg.Installs.Create(ctx, install); err != nil {
		return "", "", nil, fmt.Errorf("create install: %w", err)
	}
	raw, err := newToken()
	if err != nil {
		return "", "", nil, err
	}
	token := &IntegrationToken{
		ID:        ids.New(),
		InstallID: install.ID,
		TokenHash: hashToken(raw),
		Prefix:    raw[len(raw)-6:],
		CreatedAt: now,
	}
	if err := s.cfg.Tokens.Create(ctx, token); err != nil {
		return "", "", nil, fmt.Errorf("create token: %w", err)
	}
	return raw, secret, install, nil
}

// ListInstalls returns all installs; owner-only, since installs are workspace configuration, not per-user records.
func (s *Service) ListInstalls(ctx context.Context, actorID string) ([]*Install, error) {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return nil, err
	}
	installs, err := s.cfg.Installs.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installs: %w", err)
	}
	return installs, nil
}

// GetInstall returns one install by id (owner-only read).
func (s *Service) GetInstall(ctx context.Context, actorID, id string) (*Install, error) {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return nil, err
	}
	install, err := s.cfg.Installs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get install %s: %w", id, err)
	}
	return install, nil
}

// RevokeInstall revokes an install immediately: its token stops authenticating and webhook fan-out skips it (ADR 0043).
func (s *Service) RevokeInstall(ctx context.Context, actorID, id string) error {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return err
	}
	if err := s.cfg.Installs.Revoke(ctx, id); err != nil {
		return fmt.Errorf("revoke install %s: %w", id, err)
	}
	if err := s.cfg.Tokens.RevokeByInstall(ctx, id); err != nil {
		return fmt.Errorf("revoke install %s tokens: %w", id, err)
	}
	return nil
}

// AuthenticateToken resolves a raw scoped token to its install (ADR 0043); a revoked or unknown token is ErrUnauthorized.
func (s *Service) AuthenticateToken(ctx context.Context, raw string) (*Install, *IntegrationToken, error) {
	if !strings.HasPrefix(raw, tokenPrefix) {
		return nil, nil, apperrs.ErrUnauthorized
	}
	token, err := s.cfg.Tokens.GetByHash(ctx, hashToken(raw))
	if err != nil {
		return nil, nil, apperrs.ErrUnauthorized
	}
	if token.RevokedAt != nil {
		return nil, nil, apperrs.ErrUnauthorized
	}
	install, err := s.cfg.Installs.GetByID(ctx, token.InstallID)
	if err != nil {
		return nil, nil, apperrs.ErrUnauthorized
	}
	if install.RevokedAt != nil {
		return nil, nil, apperrs.ErrUnauthorized
	}
	_ = s.cfg.Tokens.TouchLastUsed(ctx, token.ID)
	return install, token, nil
}

// Subscribe registers a webhook topic for an install (ADR 0043, per-topic v1).
func (s *Service) Subscribe(ctx context.Context, actorID, installID, topic string) error {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return err
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("%w: topic is required", apperrs.ErrInvalid)
	}
	if _, err := s.cfg.Installs.GetByID(ctx, installID); err != nil {
		return fmt.Errorf("get install %s: %w", installID, err)
	}
	if err := s.cfg.Subs.Add(ctx, Subscription{InstallID: installID, Topic: topic, CreatedAt: s.cfg.Now().UTC()}); err != nil {
		return fmt.Errorf("subscribe %s: %w", topic, err)
	}
	return nil
}

// Unsubscribe removes a webhook topic registration.
func (s *Service) Unsubscribe(ctx context.Context, actorID, installID, topic string) error {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return err
	}
	if err := s.cfg.Subs.Remove(ctx, installID, topic); err != nil {
		return fmt.Errorf("unsubscribe %s: %w", topic, err)
	}
	return nil
}

// ListSubscriptions returns the topics an install subscribes to.
func (s *Service) ListSubscriptions(ctx context.Context, actorID, installID string) ([]Subscription, error) {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return nil, err
	}
	subs, err := s.cfg.Subs.ListByInstall(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return subs, nil
}

// ListDeliveries returns the delivery history for an install.
func (s *Service) ListDeliveries(ctx context.Context, actorID, installID string) ([]*Delivery, error) {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return nil, err
	}
	deliveries, err := s.cfg.Deliveries.ListByInstall(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	return deliveries, nil
}

// Catalog returns every published schema version, newest first per topic.
func (s *Service) Catalog(ctx context.Context) ([]SchemaEntry, error) {
	entries, err := s.cfg.Schemas.Catalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("list schema catalog: %w", err)
	}
	return entries, nil
}

// ListAudit returns recent audit rows (owner-only read).
func (s *Service) ListAudit(ctx context.Context, actorID string, limit int) ([]AuditEntry, error) {
	if err := s.requireOwner(ctx, actorID); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	entries, err := s.cfg.Audit.List(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	return entries, nil
}

func (s *Service) requireOwner(ctx context.Context, actorID string) error {
	if s.cfg.Owner == nil {
		return fmt.Errorf("%w: owner gate is not configured", apperrs.ErrForbidden)
	}
	ok, err := s.cfg.Owner.CanCreateWorkspace(ctx, actorID)
	if err != nil {
		return fmt.Errorf("resolve owner: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: owner role required", apperrs.ErrForbidden)
	}
	return nil
}

// validateWebhookURL rejects non-http(s) URLs and embedded credentials, which would leak into the signed delivery.
func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: webhook URL must be an absolute http(s) URL", apperrs.ErrInvalid)
	}
	if u.User != nil {
		return fmt.Errorf("%w: webhook URL must not contain credentials", apperrs.ErrInvalid)
	}
	return nil
}

func scopeStrings(scopes []Scope) []string {
	out := make([]string, len(scopes))
	for i, sc := range scopes {
		out[i] = string(sc)
	}
	return out
}

// newToken mints a raw scoped token: tokenPrefix + 32 random bytes, base64url.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate integration token: %w", err)
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// newWebhookSecret mints the HMAC secret for signing outbound deliveries (ADR 0043); high entropy makes it unforgeable.
func newWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate webhook secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken derives the stored hash of a raw token (SHA-256, hex). Token
// hashes are high-entropy and long-lived, so a non-keyed hash is safe.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

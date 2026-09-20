package automations

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// tokenPrefix marks a scoped automation token so the gateway can tell token kinds apart.
const tokenPrefix = "dat_"

// Service is the automations use-case layer: identity, the token lifecycle, and the owner-edited fields.
type Service struct {
	repo  Repo
	perm  PermissionGate
	now   func() time.Time
	conns ConnectionRegistry
	// scopeAllows and owner back the HTTP gateway; nil until SetGateway wires them.
	scopeAllows   ScopeGate
	resolveScopes ScopeResolver
	owner         OwnerGate
}

// NewService wires the automations use-cases over the given repo and permission gate.
func NewService(repo Repo, perm PermissionGate) *Service {
	return &Service{repo: repo, perm: perm, now: time.Now}
}

// SetConnectionRegistry wires the dial-in registry after construction, closing a cycle the constructor can't.
func (s *Service) SetConnectionRegistry(r ConnectionRegistry) {
	s.conns = r
}

// AuthenticateToken resolves the automation dialing in with raw (ADR 0046); revocation kills access immediately.
func (s *Service) AuthenticateToken(ctx context.Context, raw string) (*Automation, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: token is required", apperrs.ErrUnauthorized)
	}
	a, err := s.repo.GetByTokenHash(ctx, hashToken(raw))
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil, fmt.Errorf("%w: invalid automation token", apperrs.ErrUnauthorized)
		}
		return nil, fmt.Errorf("authenticate automation token: %w", err)
	}
	if a.TokenRevokedAt != nil {
		return nil, fmt.Errorf("%w: automation token revoked", apperrs.ErrUnauthorized)
	}
	return a, nil
}

// Create mints a new Custom automation shell and token; the raw token is returned once, only its hash is stored.
func (s *Service) Create(ctx context.Context, actorID, name string, scopes []string) (*Automation, string, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsWrite); err != nil {
		return nil, "", err
	}
	normalized, err := normalizeScopes(scopes)
	if err != nil {
		return nil, "", err
	}
	if s.resolveScopes != nil {
		if normalized, err = s.resolveScopes(normalized); err != nil {
			return nil, "", err
		}
	}
	raw, hash, prefix, err := mintToken()
	if err != nil {
		return nil, "", err
	}
	now := s.now().UTC()
	a := &Automation{
		ID:           ids.New(),
		Name:         strings.TrimSpace(name),
		Kind:         KindCustom,
		Enabled:      false,
		ConfigSchema: json.RawMessage(`{}`),
		ConfigValues: json.RawMessage(`{}`),
		Scopes:       normalized,
		CreatedBy:    actorID,
		TokenHash:    hash,
		TokenPrefix:  prefix,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := a.Validate(); err != nil {
		return nil, "", err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, "", fmt.Errorf("create automation: %w", err)
	}
	return a, raw, nil
}

// List returns every automation, defaults included.
func (s *Service) List(ctx context.Context, actorID string) ([]Automation, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list automations: %w", err)
	}
	return list, nil
}

// Get returns a single automation by id.
func (s *Service) Get(ctx context.Context, actorID, id string) (*Automation, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	return s.getByID(ctx, id)
}

// SetEnabled toggles an automation; disabled means no event delivery and no worker.
func (s *Service) SetEnabled(ctx context.Context, actorID, id string, enabled bool) (*Automation, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsWrite); err != nil {
		return nil, err
	}
	a, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.Enabled == enabled {
		return a, nil
	}
	a.Enabled = enabled
	a.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("set automation %s enabled: %w", id, err)
	}
	return a, nil
}

// UpdateConfigValues' schema validation is the SDK's job, not this one's.
func (s *Service) UpdateConfigValues(ctx context.Context, actorID, id string, values json.RawMessage) (*Automation, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsWrite); err != nil {
		return nil, err
	}
	if !json.Valid(values) {
		return nil, fmt.Errorf("%w: config values must be valid JSON", apperrs.ErrInvalid)
	}
	a, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	a.ConfigValues = values
	a.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("update automation %s config: %w", id, err)
	}
	return a, nil
}

// Delete removes an automation permanently, revoking its token so it can never authenticate again.
func (s *Service) Delete(ctx context.Context, actorID, id string) error {
	if err := s.require(ctx, actorID, permissions.AutomationsDelete); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete automation %s: %w", id, err)
	}
	if s.conns != nil {
		s.conns.Disconnect(id, "automation deleted")
	}
	return nil
}

// MintToken rotates a token; the previous hash is overwritten, not kept, so it stops authenticating immediately.
func (s *Service) MintToken(ctx context.Context, actorID, id string) (*Automation, string, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsWrite); err != nil {
		return nil, "", err
	}
	a, err := s.getByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	raw, hash, prefix, err := mintToken()
	if err != nil {
		return nil, "", err
	}
	a.TokenHash = hash
	a.TokenPrefix = prefix
	a.TokenRevokedAt = nil
	a.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, "", fmt.Errorf("mint token for automation %s: %w", id, err)
	}
	// A rotation is usually a response to a leaked credential — a live dial-in on the old token must not outlive it.
	if s.conns != nil {
		s.conns.Disconnect(id, "token rotated")
	}
	return a, raw, nil
}

// RevokeToken invalidates an automation's current token immediately; minting a new one restores it.
func (s *Service) RevokeToken(ctx context.Context, actorID, id string) (*Automation, error) {
	if err := s.require(ctx, actorID, permissions.AutomationsWrite); err != nil {
		return nil, err
	}
	a, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.TokenRevokedAt != nil {
		return a, nil
	}
	now := s.now().UTC()
	a.TokenRevokedAt = &now
	a.UpdatedAt = now
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("revoke token for automation %s: %w", id, err)
	}
	if s.conns != nil {
		s.conns.Disconnect(id, "token revoked")
	}
	return a, nil
}

// SyncFromCode overwrites the code-declared surface at dial-in; ungated since the caller already is the automation.
func (s *Service) SyncFromCode(ctx context.Context, id, name, description string, subscriptions []string, configSchema json.RawMessage) (*Automation, error) {
	a, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if len(configSchema) == 0 {
		configSchema = a.ConfigSchema
	}
	// The host restarts a worker whenever updated_at moves; an unchanged announce must not write, or reconnects loop.
	if a.Name == name && a.Description == description && slices.Equal(a.Subscriptions, subscriptions) && bytes.Equal(a.ConfigSchema, configSchema) {
		return a, nil
	}
	a.Name = name
	a.Description = description
	a.Subscriptions = subscriptions
	a.ConfigSchema = configSchema
	a.UpdatedAt = s.now().UTC()
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("sync automation %s from code: %w", id, err)
	}
	return a, nil
}

func (s *Service) getByID(ctx context.Context, id string) (*Automation, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: automation id is required", apperrs.ErrInvalid)
	}
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get automation %s: %w", id, err)
	}
	return a, nil
}

func (s *Service) require(ctx context.Context, actorID string, action permissions.Action) error {
	return requirePermission(ctx, s.perm, actorID, action)
}

// requirePermission is the shared check every automations use-case enforces automation.* actions through.
func requirePermission(ctx context.Context, perm PermissionGate, actorID string, action permissions.Action) error {
	// RequireAutomation only reaches an automation's own subtree, so this read is a self-read needing no permission.
	if actor, ok := identity.ActorFromCtx(ctx); ok && actor.Automation != nil && action == permissions.AutomationsRead {
		return nil
	}
	if strings.TrimSpace(actorID) == "" {
		return fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if perm == nil || !perm.HasPermission(ctx, actorID, action) {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}

// mintToken generates a raw token, its hash, and a display prefix to recognize it without seeing it again.
func mintToken() (raw, hash, prefix string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generate automation token: %w", err)
	}
	raw = tokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw), raw[len(raw)-6:], nil
}

// hashToken ensures a leaked database row never yields a usable token.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

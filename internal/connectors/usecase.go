package connectors

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// OwnerGate gates the owner-only app-config write via can_create_workspace (CN3a, ADR 0017).
type OwnerGate interface {
	CanCreateWorkspace(ctx context.Context, userID string) (bool, error)
}

// SettingsReader is the consumer-side view of the instance settings this package needs (ADR 0017).
type SettingsReader interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// Config wires the connectors use-cases.
type Config struct {
	// Store persists connector credentials, encrypted at rest.
	Store CredentialsStore
	// AppConfigStore persists each connector's app-level OAuth registration (CN3a); nil disables SetAppConfig.
	AppConfigStore AppConfigStore
	// Owner resolves owner-only writes; nil disables SetAppConfig.
	Owner OwnerGate
	// Settings resolves the SPA's URL for the post-OAuth redirect; the provider's API host may not be browser-reachable.
	Settings SettingsReader
	// Registry overrides the package's static Registry() for tests.
	Registry []Connector
	// Now is overridable for tests.
	Now func() time.Time
}

// Service is the connectors use-case layer: OAuth connect/disconnect and status for every registry entry.
type Service struct {
	store          CredentialsStore
	appConfigStore AppConfigStore
	owner          OwnerGate
	settings       SettingsReader
	registry       map[string]Connector
	now            func() time.Time
}

// NewService wires the connectors use-cases.
func NewService(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	regs := cfg.Registry
	if regs == nil {
		regs = Registry()
	}
	byID := make(map[string]Connector, len(regs))
	for _, c := range regs {
		byID[c.ID] = c
	}
	return &Service{
		store:          cfg.Store,
		appConfigStore: cfg.AppConfigStore,
		owner:          cfg.Owner,
		settings:       cfg.Settings,
		registry:       byID,
		now:            cfg.Now,
	}
}

// InstanceURL returns the SPA-facing base URL, or "" when unreadable, so the callback falls back to the request host.
func (s *Service) InstanceURL(ctx context.Context) string {
	if s.settings == nil {
		return ""
	}
	u, err := s.settings.GetInstanceURL(ctx)
	if err != nil {
		return ""
	}
	return strings.TrimRight(u, "/")
}

// ConnectorStatus is one registry entry plus its connection state, what the frontend renders as a connector card.
type ConnectorStatus struct {
	Connector Connector        `json:"connector"`
	Status    CredentialStatus `json:"status"`
	// Available means a live OAuth implementation is wired up, regardless of Status.Configured; false is "coming soon".
	Available bool `json:"available"`
	// AppConfigured means the OAuth app registration (CN3a) is stored, so Connect can actually start the consent flow.
	AppConfigured bool `json:"app_configured"`
}

// List returns every registry entry with its current CredentialStatus, zero-valued for connectors never connected.
func (s *Service) List(ctx context.Context) ([]ConnectorStatus, error) {
	regs := make([]Connector, 0, len(s.registry))
	for _, c := range s.registry {
		regs = append(regs, c)
	}
	sort.Slice(regs, func(i, j int) bool { return regs[i].ID < regs[j].ID })

	out := make([]ConnectorStatus, 0, len(regs))
	for _, c := range regs {
		available := c.OAuth != nil || len(c.Manual) > 0
		appConfigured := c.OAuth != nil && c.OAuth.Configured()
		cred, err := s.store.GetCredentials(ctx, c.ID)
		if err != nil {
			if errors.Is(err, apperrs.ErrNotFound) {
				out = append(out, ConnectorStatus{Connector: c, Available: available, AppConfigured: appConfigured})
				continue
			}
			return nil, fmt.Errorf("get credentials %s: %w", c.ID, err)
		}
		out = append(out, ConnectorStatus{Connector: c, Status: cred.Status(), Available: available, AppConfigured: appConfigured})
	}
	return out, nil
}

// AuthorizeURL builds the provider consent URL for connectorID and a CSRF state token.
func (s *Service) AuthorizeURL(ctx context.Context, connectorID, state string) (string, error) {
	c, err := s.get(connectorID)
	if err != nil {
		return "", err
	}
	// Without this, the client renders a consent URL that 404s at the provider instead of telling the SPA what's missing.
	if !c.OAuth.Configured() {
		return "", fmt.Errorf("%w: %s app is not configured — an owner has to set up its app (client ID and secret) first", apperrs.ErrInvalid, c.Name)
	}
	return c.OAuth.AuthorizeURL(state), nil
}

// CompleteOAuth exchanges a code for tokens and stores them as the connector's credential, attributed to userID.
func (s *Service) CompleteOAuth(ctx context.Context, connectorID, code, userID string) error {
	c, err := s.get(connectorID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("%w: code is required", apperrs.ErrInvalid)
	}
	ts, err := c.OAuth.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange %s oauth code: %w", connectorID, err)
	}
	now := s.now().UTC()
	return s.store.SaveCredentials(ctx, Credentials{
		ConnectorID:  connectorID,
		AccessToken:  ts.AccessToken,
		RefreshToken: ts.RefreshToken,
		ExpiresAt:    now.Add(ts.ExpiresIn),
		ConnectedBy:  userID,
		ConnectedAt:  now,
	})
}

// Disconnect revokes and deletes the stored credential; the local row is deleted even if provider revocation fails.
func (s *Service) Disconnect(ctx context.Context, connectorID string) error {
	c, ok := s.registry[connectorID]
	if !ok {
		return fmt.Errorf("%w: unknown connector %q", apperrs.ErrNotFound, connectorID)
	}
	cred, err := s.store.GetCredentials(ctx, connectorID)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get credentials %s: %w", connectorID, err)
	}
	if c.OAuth != nil {
		_ = c.OAuth.Revoke(ctx, cred.AccessToken)
	}
	if err := s.store.DeleteCredentials(ctx, connectorID); err != nil {
		return fmt.Errorf("delete credentials %s: %w", connectorID, err)
	}
	return nil
}

// checkManual trims and requires every declared field, then runs the connector's live Verify; nothing is stored here.
func (s *Service) checkManual(ctx context.Context, connectorID string, fields map[string]string) (map[string]string, error) {
	c, ok := s.registry[connectorID]
	if !ok {
		return nil, fmt.Errorf("%w: unknown connector %q", apperrs.ErrNotFound, connectorID)
	}
	if len(c.Manual) == 0 {
		return nil, fmt.Errorf("%w: %s has no manual credential fields", apperrs.ErrInvalid, connectorID)
	}
	clean := make(map[string]string, len(c.Manual))
	for _, f := range c.Manual {
		v := strings.TrimSpace(fields[f.Key])
		if v == "" {
			return nil, fmt.Errorf("%w: %s is required", apperrs.ErrInvalid, f.Label)
		}
		clean[f.Key] = v
	}
	if c.Verify == nil {
		return nil, fmt.Errorf("%w: %s has no verifier wired, cannot confirm the credential works", apperrs.ErrInvalid, c.Name)
	}
	if err := c.Verify.Verify(ctx, clean); err != nil {
		return nil, asInvalid(err)
	}
	return clean, nil
}

// VerifyManualCredentials runs the live provider check without storing, so the UI can show a green light before Confirm.
func (s *Service) VerifyManualCredentials(ctx context.Context, connectorID string, fields map[string]string) error {
	_, err := s.checkManual(ctx, connectorID, fields)
	return err
}

// VerifyManualCheck runs one named permission check, so the dialog can fan the checks out in parallel and tick each row.
func (s *Service) VerifyManualCheck(ctx context.Context, connectorID string, fields map[string]string, key string) error {
	c, ok := s.registry[connectorID]
	if !ok {
		return fmt.Errorf("%w: unknown connector %q", apperrs.ErrNotFound, connectorID)
	}
	if !slices.ContainsFunc(c.Checks, func(ch CredentialCheck) bool { return ch.Key == key }) {
		return fmt.Errorf("%w: %s has no check %q", apperrs.ErrInvalid, connectorID, key)
	}
	cv, ok := c.Verify.(CheckVerifier)
	if !ok {
		return fmt.Errorf("%w: %s cannot verify checks one at a time", apperrs.ErrInvalid, c.Name)
	}
	clean := make(map[string]string, len(c.Manual))
	for _, f := range c.Manual {
		v := strings.TrimSpace(fields[f.Key])
		if v == "" {
			return fmt.Errorf("%w: %s is required", apperrs.ErrInvalid, f.Label)
		}
		clean[f.Key] = v
	}
	if err := cv.VerifyCheck(ctx, clean, key); err != nil {
		return asInvalid(err)
	}
	return nil
}

// asInvalid surfaces a provider's verdict as bad input without stacking a second "invalid:" prefix on one it already carries.
func asInvalid(err error) error {
	if errors.Is(err, apperrs.ErrInvalid) {
		return err
	}
	return fmt.Errorf("%w: %v", apperrs.ErrInvalid, err)
}

// SaveManualCredentials verifies again and stores, so a save can never bypass the live check.
func (s *Service) SaveManualCredentials(ctx context.Context, connectorID, userID string, fields map[string]string) (CredentialStatus, error) {
	clean, err := s.checkManual(ctx, connectorID, fields)
	if err != nil {
		return CredentialStatus{}, err
	}
	now := s.now().UTC()
	cred := Credentials{ConnectorID: connectorID, ManualFields: clean, ConnectedBy: userID, ConnectedAt: now}
	if err := s.store.SaveCredentials(ctx, cred); err != nil {
		return CredentialStatus{}, fmt.Errorf("save manual credentials %s: %w", connectorID, err)
	}
	return cred.Status(), nil
}

// ManualCredentials returns stored manual field values, letting other domains reuse the same DB-backed credential.
func (s *Service) ManualCredentials(ctx context.Context, connectorID string) (map[string]string, error) {
	cred, err := s.store.GetCredentials(ctx, connectorID)
	if err != nil {
		return nil, fmt.Errorf("get credentials %s: %w", connectorID, err)
	}
	if len(cred.ManualFields) == 0 {
		return nil, fmt.Errorf("%w: %s manual credentials not configured", apperrs.ErrNotFound, connectorID)
	}
	return cred.ManualFields, nil
}

// requireOwner enforces the instance-admin (can_create_workspace) bit on CN3a's app-config write.
func (s *Service) requireOwner(ctx context.Context, userID string) error {
	if userID == "" {
		return apperrs.ErrUnauthorized
	}
	if s.owner == nil {
		return fmt.Errorf("%w: owner gate is not configured", apperrs.ErrForbidden)
	}
	ok, err := s.owner.CanCreateWorkspace(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolve owner: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: owner role required", apperrs.ErrForbidden)
	}
	return nil
}

// SetAppConfig stores an app-level OAuth registration (CN3a), owner-only since it rotates an instance-wide credential.
func (s *Service) SetAppConfig(ctx context.Context, userID, connectorID, clientID, clientSecret, baseURL, appSlug string) (AppConfigStatus, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return AppConfigStatus{}, err
	}
	if _, ok := s.registry[connectorID]; !ok {
		return AppConfigStatus{}, fmt.Errorf("%w: unknown connector %q", apperrs.ErrNotFound, connectorID)
	}
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return AppConfigStatus{}, fmt.Errorf("%w: client_id and client_secret are required", apperrs.ErrInvalid)
	}
	if s.appConfigStore == nil {
		return AppConfigStatus{}, fmt.Errorf("%w: app config store is not configured", apperrs.ErrForbidden)
	}
	cfg := AppConfig{ConnectorID: connectorID, ClientID: clientID, ClientSecret: clientSecret, BaseURL: baseURL, AppSlug: appSlug}
	if v, ok := s.registry[connectorID].OAuth.(AppVerifier); ok {
		if err := v.VerifyApp(ctx, cfg); err != nil {
			return AppConfigStatus{}, err
		}
	}
	if err := s.appConfigStore.SetAppConfig(ctx, cfg); err != nil {
		return AppConfigStatus{}, fmt.Errorf("save app config %s: %w", connectorID, err)
	}
	return cfg.Status(), nil
}

// manualTokenField is the manual credential key that stands in for an OAuth access token on connectors carrying both.
const manualTokenField = "api_token"

// AccessToken returns a live bearer token for connectorID: a manual api_token as-is, else the OAuth token refreshed if expired.
func (s *Service) AccessToken(ctx context.Context, connectorID string) (string, error) {
	c, err := s.get(connectorID)
	if err != nil {
		return "", err
	}
	cred, err := s.store.GetCredentials(ctx, connectorID)
	if err != nil {
		return "", fmt.Errorf("get credentials %s: %w", connectorID, err)
	}
	if tok := cred.ManualFields[manualTokenField]; tok != "" {
		return tok, nil
	}
	if cred.ExpiresAt.After(s.now().UTC()) || cred.RefreshToken == "" {
		return cred.AccessToken, nil
	}
	ts, err := c.OAuth.Refresh(ctx, cred.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("refresh %s oauth token: %w", connectorID, err)
	}
	now := s.now().UTC()
	cred.AccessToken = ts.AccessToken
	cred.RefreshToken = ts.RefreshToken
	cred.ExpiresAt = now.Add(ts.ExpiresIn)
	if err := s.store.SaveCredentials(ctx, cred); err != nil {
		return "", fmt.Errorf("save refreshed credentials %s: %w", connectorID, err)
	}
	return cred.AccessToken, nil
}

// get looks up connectorID, erroring when unknown or not yet OAuth-connectable.
func (s *Service) get(connectorID string) (Connector, error) {
	c, ok := s.registry[connectorID]
	if !ok {
		return Connector{}, fmt.Errorf("%w: unknown connector %q", apperrs.ErrNotFound, connectorID)
	}
	if c.OAuth == nil {
		return Connector{}, fmt.Errorf("%w: %s is not connectable yet (coming soon)", apperrs.ErrInvalid, connectorID)
	}
	return c, nil
}

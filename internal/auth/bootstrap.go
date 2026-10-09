package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// GitHubAppVerifier is the live "does this App exist and accept this secret" check, adapted at the composition root.
type GitHubAppVerifier interface {
	VerifyGitHubApp(ctx context.Context, clientID, clientSecret, appSlug string) error
	// VerifyGitHubAppCheck runs one named half ("slug" or "secret") so /setup can light each row separately.
	VerifyGitHubAppCheck(ctx context.Context, check, clientID, clientSecret, appSlug string) error
}

// CompleteOwnerWizard binds the caller to the default workspace's Owner role; anyone but that Owner repeating it conflicts.
func (s *Service) CompleteOwnerWizard(ctx context.Context, userID, instanceURL string) error {
	if err := validateInstanceURL(instanceURL); err != nil {
		return err
	}
	user, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user %s: %w", userID, err)
	}
	owners, err := s.cfg.Users.ListActiveOwnerIDs(ctx)
	if err != nil {
		return fmt.Errorf("list owners: %w", err)
	}
	// Idempotent for the owner themself: a double-submitted finish re-runs instead of conflicting.
	if len(owners) > 0 && !slices.Contains(owners, user.ID) {
		return fmt.Errorf("%w: this instance already has an owner", apperrs.ErrConflict)
	}
	if _, err := s.cfg.Settings.Set(ctx, instanceURL); err != nil {
		return fmt.Errorf("set instance url: %w", err)
	}
	// Bind to the pre-seeded default workspace as its Owner-role member instead of creating a second, duplicate workspace.
	if err := s.cfg.DefaultWorkspace.BindDefaultWorkspaceOwner(ctx, user.ID); err != nil {
		return fmt.Errorf("bind default workspace owner: %w", err)
	}
	if err := s.cfg.Users.MarkFirstLoginDone(ctx, user.ID); err != nil {
		return fmt.Errorf("mark first login: %w", err)
	}
	return nil
}

// CompleteFirstLogin only completes the profile; identity stays provider-sourced.
func (s *Service) CompleteFirstLogin(ctx context.Context, userID string) error {
	return s.cfg.Users.MarkFirstLoginDone(ctx, userID)
}

// Bootstrap stores the GitHub App credentials, and the instance URL unless first run already stored it; setup pass only.
func (s *Service) Bootstrap(ctx context.Context, callerID, instanceURL, clientID, clientSecret, appSlug string) (Settings, error) {
	if err := requireSetupPass(callerID); err != nil {
		return Settings{}, err
	}
	instanceURL, stored, err := s.bootstrapInstanceURL(ctx, instanceURL)
	if err != nil {
		return Settings{}, err
	}
	if err := s.bootstrapAllowed(ctx, clientID, clientSecret, appSlug); err != nil {
		return Settings{}, err
	}
	if s.cfg.GitHubApp != nil {
		if err := s.cfg.GitHubApp.VerifyGitHubApp(ctx, clientID, clientSecret, appSlug); err != nil {
			return Settings{}, err
		}
	}
	if !stored {
		if _, err := s.cfg.Settings.Set(ctx, instanceURL); err != nil {
			return Settings{}, fmt.Errorf("set instance url: %w", err)
		}
	}
	st, err := s.cfg.Settings.SetGitHubOAuth(ctx, clientID, clientSecret)
	if err != nil {
		return Settings{}, fmt.Errorf("set github oauth: %w", err)
	}
	// Same GitHub App backs the connectors' install flow, so seed its config now for Connect without a Settings visit.
	if s.cfg.ConnectorApps != nil {
		if err := s.cfg.ConnectorApps.SeedGitHubApp(ctx, clientID, clientSecret, appSlug); err != nil {
			return Settings{}, fmt.Errorf("seed github connector app: %w", err)
		}
	}
	return st, nil
}

// BootstrapVerify runs one named GitHub App check without storing anything, under Bootstrap's own guards so a live instance never proxies GitHub for strangers.
func (s *Service) BootstrapVerify(ctx context.Context, callerID, check, clientID, clientSecret, appSlug string) error {
	if err := requireSetupPass(callerID); err != nil {
		return err
	}
	if err := s.bootstrapAllowed(ctx, clientID, clientSecret, appSlug); err != nil {
		return err
	}
	if s.cfg.GitHubApp == nil {
		return nil
	}
	return s.cfg.GitHubApp.VerifyGitHubAppCheck(ctx, check, clientID, clientSecret, appSlug)
}

// bootstrapAllowed is the shared gate for Bootstrap and BootstrapVerify: fields present, slug not the numeric App ID, and the instance still replaceable.
func (s *Service) bootstrapAllowed(ctx context.Context, clientID, clientSecret, appSlug string) error {
	if clientID == "" || clientSecret == "" || appSlug == "" {
		return fmt.Errorf("%w: client ID, client secret and app slug are required", apperrs.ErrInvalid)
	}
	// GitHub's install URL takes the URL slug, not the numeric "App ID"; a digits-only value is that ID pasted by mistake.
	if strings.Trim(appSlug, "0123456789") == "" {
		return fmt.Errorf("%w: that looks like the numeric App ID — the app slug is the name in your app's URL (github.com/apps/{slug})", apperrs.ErrInvalid)
	}
	return s.bootstrapReplaceable(ctx)
}

// bootstrapReplaceable fails ErrConflict once the instance is configured and a user has logged in.
func (s *Service) bootstrapReplaceable(ctx context.Context) error {
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	if !st.Configured() {
		return nil
	}
	open, err := s.SetupOpen(ctx)
	if err != nil {
		return err
	}
	if !open {
		return fmt.Errorf("%w: this instance is already configured", apperrs.ErrConflict)
	}
	return nil
}

// InstanceURL returns the configured instance URL, or "" when none is stored or Settings is unwired.
func (s *Service) InstanceURL(ctx context.Context) string {
	if s.cfg.Settings == nil {
		return ""
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return ""
	}
	return st.InstanceURL
}

func (s *Service) secureCookie(ctx context.Context, requestTLS bool) bool {
	if requestTLS {
		return true
	}
	if s.cfg.Settings == nil {
		return false
	}
	settings, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return false
	}
	parsed, err := url.Parse(settings.InstanceURL)
	return err == nil && strings.EqualFold(parsed.Scheme, "https")
}

// VerifyInstanceURL is the bootstrap ticker's reachability row: this server fetches its own bootstrap-status through
// the pasted URL, proving DNS, the proxy, and TLS before the URL is stored and every callback gets derived from it.
// Gated like the other checks, so a live instance never fetches URLs on a stranger's behalf.
func (s *Service) VerifyInstanceURL(ctx context.Context, instanceURL string) (err error) {
	if err := validateInstanceURL(instanceURL); err != nil {
		return err
	}
	if err := s.bootstrapReplaceable(ctx); err != nil {
		return err
	}
	target := strings.TrimSuffix(strings.TrimSpace(instanceURL), "/") + "/api/auth/bootstrap-status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", apperrs.ErrInvalid, err)
	}
	resp, err := s.probeClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: this server could not reach %s: %v", apperrs.ErrInvalid, instanceURL, err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	// ponytail: any Nexul answers this shape, not provably this one; add a per-process nonce if that ever matters.
	var body struct {
		Configured *bool `json:"configured"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body) != nil || body.Configured == nil {
		return fmt.Errorf("%w: %s answered, but not as a Nexul server (status %d) — check that the proxy forwards /api to it", apperrs.ErrInvalid, instanceURL, resp.StatusCode)
	}
	return nil
}

// VerifyBootstrapInstanceURL is the bootstrap ticker's reachability row for the URL Bootstrap would use; setup pass only.
func (s *Service) VerifyBootstrapInstanceURL(ctx context.Context, callerID, instanceURL string) error {
	if err := requireSetupPass(callerID); err != nil {
		return err
	}
	instanceURL, _, err := s.bootstrapInstanceURL(ctx, instanceURL)
	if err != nil {
		return err
	}
	return s.VerifyInstanceURL(ctx, instanceURL)
}

// bootstrapInstanceURL is the stored URL (given must match or be empty) when there is one, else the validated given.
func (s *Service) bootstrapInstanceURL(ctx context.Context, given string) (instanceURL string, stored bool, err error) {
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return "", false, fmt.Errorf("get settings: %w", err)
	}
	given = strings.TrimSpace(given)
	if st.InstanceURL == "" {
		return given, false, validateInstanceURL(given)
	}
	if given != "" && !strings.EqualFold(strings.TrimSuffix(given, "/"), strings.TrimSuffix(st.InstanceURL, "/")) {
		return "", false, fmt.Errorf("%w: the instance URL is already set to %s", apperrs.ErrInvalid, st.InstanceURL)
	}
	return st.InstanceURL, true, nil
}

// UpdateInstanceURL changes the instance URL (instance:write), bumping the version so new connection tokens carry it.
func (s *Service) UpdateInstanceURL(ctx context.Context, userID, instanceURL string) (Settings, error) {
	if err := validateInstanceURL(instanceURL); err != nil {
		return Settings{}, err
	}
	if err := s.requireAnywhere(ctx, userID, permissions.InstanceWrite); err != nil {
		return Settings{}, err
	}
	return s.cfg.Settings.Set(ctx, instanceURL)
}

// GenerateConnectionToken mints a connection token from the instance settings (instance:read); carries server info only.
func (s *Service) GenerateConnectionToken(ctx context.Context, userID string) (*ConnectionToken, error) {
	if err := s.requireAnywhere(ctx, userID, permissions.InstanceRead); err != nil {
		return nil, err
	}
	st, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	if st.InstanceURL == "" {
		return nil, fmt.Errorf("%w: set the instance URL (owner wizard) before generating a connection token", apperrs.ErrInvalid)
	}
	token, err := s.signConnectionToken(st)
	if err != nil {
		return nil, err
	}
	return &ConnectionToken{
		Token:           token,
		InstanceURL:     st.InstanceURL,
		MCPURL:          mcpURLFor(st.InstanceURL),
		SettingsVersion: st.SettingsVersion,
		ExpiresAt:       s.cfg.Now().Add(connectionTokenTTL),
	}, nil
}

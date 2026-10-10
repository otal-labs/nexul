package auth

import (
	"context"
	"crypto/hmac"
	"fmt"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

// GitHubAppCredentials stays inside the conversion and storage boundaries; no API returns it.
type GitHubAppCredentials struct {
	ClientID     string
	ClientSecret string `json:"-"`
	Slug         string
	PrivateKey   string `json:"-"`
}

// GitHubManifestConverter trades GitHub's one-use registration code for the new App's credentials.
type GitHubManifestConverter interface {
	Convert(ctx context.Context, code string) (GitHubAppCredentials, error)
}

// GitHubManifestStore consumes requests once and stores both sign-in and connector credentials atomically.
type GitHubManifestStore interface {
	Start(ctx context.Context, stateHash, initiatorHash string, expiresAt, now time.Time) error
	Consume(ctx context.Context, stateHash, initiatorHash string, now time.Time) error
	Save(ctx context.Context, app GitHubAppCredentials) error
}

const manifestTTL = 15 * time.Minute

// StartGitHubManifest binds registration to the exact setup pass, rather than the shared setup identity.
func (s *Service) StartGitHubManifest(ctx context.Context, callerID, pass string) (map[string]any, error) {
	if err := s.requireManifestSetup(ctx, callerID, pass); err != nil {
		return nil, err
	}
	settings, err := s.cfg.Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateInstanceURL(settings.InstanceURL); err != nil {
		return nil, err
	}
	if !s.cfg.Local && !strings.HasPrefix(settings.InstanceURL, "https://") {
		return nil, fmt.Errorf("%w: set the https instance URL first", apperrs.ErrInvalid)
	}
	nonce, _, err := hostcred.MintCredential("ghm_")
	if err != nil {
		return nil, err
	}
	state := nonce + "." + s.mac("github-manifest."+nonce)
	now := s.cfg.Now()
	if err := s.cfg.GitHubManifestStore.Start(ctx, hostcred.Hash(state), hostcred.Hash(pass), now.Add(manifestTTL), now); err != nil {
		return nil, err
	}
	base := strings.TrimRight(settings.InstanceURL, "/")
	return map[string]any{
		"url": "https://github.com/settings/apps/new?" + url.Values{"state": {state}}.Encode(),
		"manifest": map[string]any{
			"name": "Nexul", "url": base, "redirect_url": base + "/auth/github/manifest/callback",
			"callback_urls":   []string{base + "/auth/callback", base + "/auth/connectors/github/callback"},
			"hook_attributes": map[string]any{"url": base + "/hooks/github", "active": false},
			"public":          true, "request_oauth_on_install": true,
			"default_permissions": map[string]string{"contents": "read", "pull_requests": "write", "repository_hooks": "write"},
		},
	}, nil
}

func (s *Service) requireManifestSetup(ctx context.Context, callerID, pass string) error {
	if err := requireSetupPass(callerID); err != nil {
		return err
	}
	if err := s.verifySetupPass(pass); err != nil {
		return err
	}
	if err := s.bootstrapReplaceable(ctx); err != nil {
		return err
	}
	if s.cfg.GitHubManifestStore == nil || s.cfg.GitHubManifest == nil {
		return fmt.Errorf("%w: GitHub registration is unavailable", apperrs.ErrInvalid)
	}
	return nil
}

// CompleteGitHubManifest consumes state before the external exchange, so failures and concurrent callbacks cannot replay it.
func (s *Service) CompleteGitHubManifest(ctx context.Context, callerID, pass, state, code string) error {
	if err := s.requireManifestSetup(ctx, callerID, pass); err != nil {
		return err
	}
	nonce, mac, ok := strings.Cut(state, ".")
	if !ok || !strings.HasPrefix(nonce, "ghm_") || !hmac.Equal([]byte(mac), []byte(s.mac("github-manifest."+nonce))) {
		return fmt.Errorf("%w: invalid GitHub registration state", apperrs.ErrInvalid)
	}
	if code == "" || len(code) > 256 {
		return fmt.Errorf("%w: GitHub registration code is missing or invalid", apperrs.ErrInvalid)
	}
	if err := s.cfg.GitHubManifestStore.Consume(ctx, hostcred.Hash(state), hostcred.Hash(pass), s.cfg.Now()); err != nil {
		return err
	}
	app, err := s.cfg.GitHubManifest.Convert(ctx, code)
	if err != nil {
		return err
	}
	if app.ClientID == "" || app.ClientSecret == "" || app.Slug == "" || app.PrivateKey == "" {
		return fmt.Errorf("%w: GitHub returned incomplete App credentials", apperrs.ErrInvalid)
	}
	return s.cfg.GitHubManifestStore.Save(ctx, app)
}

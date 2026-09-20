// Package connectors holds credentials the instance uses to call out to third-party tools on its own behalf.
package connectors

import (
	"context"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// OAuthClient is the provider OAuth consent flow for one connector, redefined so this never imports internal/dns.
type OAuthClient interface {
	// Configured reports whether the OAuth app is registered.
	Configured() bool
	// AuthorizeURL builds the provider consent URL for a CSRF state token.
	AuthorizeURL(state string) string
	// Exchange trades an authorization code for tokens.
	Exchange(ctx context.Context, code string) (*TokenSet, error)
	// Refresh trades a refresh token for fresh tokens.
	Refresh(ctx context.Context, refreshToken string) (*TokenSet, error)
	// Revoke asks the provider to invalidate the access token; without one, return nil.
	Revoke(ctx context.Context, accessToken string) error
}

// AppVerifier is an optional OAuthClient extension that checks an app registration live before SetAppConfig stores it.
type AppVerifier interface {
	VerifyApp(ctx context.Context, cfg AppConfig) error
}

// CredentialField describes one manual form input, rendered by the frontend; values are never part of this type.
type CredentialField struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Secret bool   `json:"secret"`
	// Hint is one line under the input telling the human what to create on the provider side (scopes, permissions).
	Hint string `json:"hint,omitempty"`
}

// Verifier makes one live call against submitted manual field values before they are stored ("verify at save").
type Verifier interface {
	Verify(ctx context.Context, fields map[string]string) error
}

// CredentialCheck is one named permission the Connect dialog lists up front (with why) and verifies on its own row.
type CredentialCheck struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Why   string `json:"why,omitempty"`
}

// CheckVerifier is an optional Verifier extension that can run a single named check, so the dialog can fan them out in parallel.
type CheckVerifier interface {
	VerifyCheck(ctx context.Context, fields map[string]string, key string) error
}

// NotInstalledError is Exchange's "authorized but not installed" outcome, carrying the install URL to forward to.
type NotInstalledError struct {
	InstallURL string
}

func (e *NotInstalledError) Error() string {
	return "app was authorized but is not installed on any account or organization"
}

func (e *NotInstalledError) Unwrap() error { return apperrs.ErrInvalid }

// TokenSet is the result of an exchange or refresh.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// Connector is one static-registry entry; OAuth is nil when unimplemented ("coming soon").
type Connector struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	// Icon is an identifier string the frontend maps to an actual icon.
	Icon    string `json:"icon"`
	DocsURL string `json:"docs_url,omitempty"`
	// GitProvider marks that this connector backs gitprovider's consumers, unlike Category, which is cosmetic.
	GitProvider bool `json:"git_provider"`
	// Manual lists the manual-credential form fields; nil means OAuth-only. A connector may set both.
	Manual []CredentialField `json:"manual,omitempty"`
	// OAuth is never serialized: nil or a live client, never API-response data.
	OAuth OAuthClient `json:"-"`
	// Verify is never serialized; nil disables SaveManualCredentials rather than storing unverified fields.
	Verify Verifier `json:"-"`
	// Checks lists the named permissions Verify proves, rendered as rows in the Connect dialog; empty means one pass/fail.
	Checks []CredentialCheck `json:"checks,omitempty"`
}

// Credentials are the stored token data for a connection; never serialized to JSON, logged, or put on events.
type Credentials struct {
	ConnectorID  string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	ConnectedBy  string
	ConnectedAt  time.Time
	// ManualFields holds a manual connector's field values, stored as one encrypted JSON blob; empty for OAuth connectors.
	ManualFields map[string]string
}

// CredentialStatus is the safe, token-free view API responses return, never Credentials directly.
type CredentialStatus struct {
	Configured  bool      `json:"configured"`
	ConnectedBy string    `json:"connected_by,omitempty"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

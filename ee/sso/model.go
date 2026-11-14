// Package sso is single sign-on through the instance's one OIDC identity provider, an enterprise feature unlocked by the
// license key (ADR 0184, ADR 0186). The person's account, session and invitation paths stay auth's and tenancy's.
package sso

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Settings is the one provider the instance signs people in through; the client secret is as the Owner typed it, a
// secret reference included.
type Settings struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	DisplayName  string
	// JoinRoleID is the default workspace's role a person gets when their first sign-in makes their account.
	JoinRoleID string
	// Only hides every other sign-in method; the instance Owner's passkey still works.
	Only      bool
	UpdatedBy string
	UpdatedAt time.Time
}

// Configured reports whether every field a sign-in needs is set.
func (s Settings) Configured() bool {
	return s.IssuerURL != "" && s.ClientID != "" && s.ClientSecret != "" && s.DisplayName != "" && s.JoinRoleID != ""
}

// View is the settings as the Owner and MCP clients read them; the client secret never leaves the server.
type View struct {
	// Unlocked is whether the license key unlocks single sign-on right now; without it the settings stay and sign-in is off.
	Unlocked    bool   `json:"unlocked"`
	Configured  bool   `json:"configured"`
	IssuerURL   string `json:"issuer_url"`
	ClientID    string `json:"client_id"`
	SecretSet   bool   `json:"client_secret_set"`
	DisplayName string `json:"display_name"`
	JoinRoleID  string `json:"join_role_id"`
	Only        bool   `json:"only"`
	// RedirectURI is what the provider registers as this instance's callback.
	RedirectURI string `json:"redirect_uri"`
	// Active is Unlocked and Configured: the sign-in button shows.
	Active bool `json:"active"`
}

// Public is what the sign-in page reads before anyone is signed in.
type Public struct {
	Active bool   `json:"active"`
	Name   string `json:"name"`
	// Only is true while every other sign-in method is hidden.
	Only bool `json:"only"`
}

// Patch changes the settings; an omitted field keeps its value, and the secret is write-only.
type Patch struct {
	IssuerURL    *string `json:"issuer_url"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	DisplayName  *string `json:"display_name"`
	JoinRoleID   *string `json:"join_role_id"`
	Only         *bool   `json:"only"`
}

// Account is the user an identity resolved to.
type Account struct {
	UserID string
	Status string
}

// Identity is who the provider says signed in: Key is the issuer-scoped subject, Email is verified.
type Identity struct {
	Key       string
	Email     string
	Name      string
	AvatarURL string
}

// Claims are the ID token's claims a sign-in reads.
type Claims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

// Grant is what a code exchange returns: the verified claims and the refresh token, empty when the provider issues none.
type Grant struct {
	Claims       Claims
	RefreshToken string
}

// StoredGrant is a person's refresh token as the last sign-in left it.
type StoredGrant struct {
	UserID       string
	RefreshToken string
}

var (
	errNeedsLicense = fmt.Errorf("%w: single sign-on needs a license key that unlocks it; add one in Settings, Instance", apperrs.ErrForbidden)
	errUnverified   = fmt.Errorf("%w: the provider sent no verified email", apperrs.ErrUnauthorized)
	errDisabled     = fmt.Errorf("%w: account is disabled", apperrs.ErrUnauthorized)
	errRemoved      = fmt.Errorf("%w: account was removed", apperrs.ErrUnauthorized)
	errNotReady     = fmt.Errorf("%w: single sign-on is not set up to create accounts", apperrs.ErrUnauthorized)
)

// checkIssuer takes https issuers, and http only on a loopback address where a provider is run for development.
func checkIssuer(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%w: issuer URL must be an absolute https URL such as https://login.example.com", apperrs.ErrInvalid)
	}
	if u.Scheme == "https" {
		return nil
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		if u.Scheme == "http" {
			return nil
		}
	}
	return fmt.Errorf("%w: issuer URL must use https", apperrs.ErrInvalid)
}

func trim(p *string, keep string) string {
	if p == nil {
		return keep
	}
	return strings.TrimSpace(*p)
}

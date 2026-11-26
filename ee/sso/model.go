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
	// Protocol is the provider's, oidc or saml, which picks the route the button starts at; empty when none is set.
	Protocol string `json:"protocol"`
}

// The protocols a provider speaks; an instance holds one provider (ADR 0189).
const (
	ProtocolOIDC = "oidc"
	ProtocolSAML = "saml"
)

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

// Identity is who the provider says signed in: Provider is its protocol, Key the issuer-scoped subject, Email is the
// address the provider vouches for.
type Identity struct {
	Provider  string
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
	errLapsed       = fmt.Errorf("%w: single sign-on links and makes no accounts until the license key is renewed", apperrs.ErrUnauthorized)
)

// checkIssuer takes https issuers, and http only on a loopback address where a provider is run for development.
func checkIssuer(raw string) error {
	if u, err := url.Parse(raw); err == nil && u.RawQuery != "" {
		return fmt.Errorf("%w: issuer URL must be an absolute https URL such as https://login.example.com", apperrs.ErrInvalid)
	}
	return checkHTTPS("issuer URL", "https://login.example.com", raw)
}

// checkHTTPS takes an absolute https URL, and http only on a loopback address; label and example name it in the refusal.
func checkHTTPS(label, example, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("%w: %s must be an absolute https URL such as %s", apperrs.ErrInvalid, label, example)
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
	return fmt.Errorf("%w: %s must use https", apperrs.ErrInvalid, label)
}

func trim(p *string, keep string) string {
	if p == nil {
		return keep
	}
	return strings.TrimSpace(*p)
}

// MoveAccount is one account as moving sign-ins shows it: Email is the one its single sign-on brought, or the one an
// expected sign-in waits for.
type MoveAccount struct {
	ID        string    `json:"id"`
	Login     string    `json:"login"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// OldSignIn is someone still signing in through the other protocol, with the account a first sign-in through the saved
// one made under the same email, and the email an expected sign-in of theirs waits for, if either.
type OldSignIn struct {
	MoveAccount
	Match         *MoveAccount `json:"match"`
	ExpectedEmail string       `json:"expected_email"`
}

// ExpectedLink is an expected sign-in: the first sign-in through the provider that brings Account.Email lands on Account.
type ExpectedLink struct {
	Account   MoveAccount `json:"account"`
	CreatedBy string      `json:"created_by"`
	CreatedAt time.Time   `json:"created_at"`
}

// SignInMoves is what moving people from one protocol to the other reads: who is still on the other one, the accounts
// whose one sign-in is the saved provider's and match nobody, and the expected sign-ins waiting.
type SignInMoves struct {
	// Protocol and ProviderName are the saved provider's; empty when none is saved, and then the lists are too.
	Protocol     string         `json:"protocol"`
	ProviderName string         `json:"provider_name"`
	People       []OldSignIn    `json:"people"`
	Unmatched    []MoveAccount  `json:"unmatched"`
	Expected     []ExpectedLink `json:"expected"`
}

// MovePair is one sign-in to move: from the account a first sign-in made onto the person's own.
type MovePair struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
}

// MoveResult is one pair's outcome; Err is nil when it moved.
type MoveResult struct {
	MovePair
	Err error
}

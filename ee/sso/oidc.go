package sso

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// discoveryTTL is how long a provider's discovery document and key set serve sign-ins before they are fetched again; a
// token signed by a key the set has not seen fetches the keys at once.
const discoveryTTL = time.Hour

// errGrantRefused marks a refresh the provider refused, the person was removed or the grant revoked there, so their
// sessions end.
var errGrantRefused = errors.New("the provider refused the refresh token")

// errProviderUnavailable marks a refresh that says nothing about the person: the provider is down or rejects our client.
var errProviderUnavailable = errors.New("the provider could not answer the refresh")

// errClientRejected marks a code exchange the provider refused for its client credentials, the one case a re-read
// secret reference may fix.
var errClientRejected = errors.New("client credentials rejected")

// ConnectConfig is what reaching the provider takes.
type ConnectConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	// Fresh skips the cached discovery, for a check that must reach the provider now.
	Fresh bool
}

// IdP is the slice of the identity provider a sign-in calls, fakeable in tests.
type IdP interface {
	// AuthURL is where the browser goes to sign in; state is the CSRF token the callback must return.
	AuthURL(state string) string
	// Exchange trades the callback's code for the verified ID token's claims and the refresh token, when one is issued.
	Exchange(ctx context.Context, code, state string) (Grant, error)
	// Refresh trades a stored refresh token for the provider's answer and returns the token to keep, which differs when
	// the provider rotates it. errGrantRefused means the grant is gone; errProviderUnavailable means nothing is known.
	Refresh(ctx context.Context, refreshToken string) (string, error)
}

// Connect reaches the provider, which discovery does through its issuer URL.
type Connect func(ctx context.Context, c ConnectConfig) (IdP, error)

// dial discovers the provider from its issuer URL, reusing the last discovery of that issuer for discoveryTTL. The nonce
// and the PKCE verifier are derived from the state under the server's secret, so nothing but the state cookie has to
// survive the round trip.
func dial(hc *http.Client, secret []byte, now func() time.Time) Connect {
	d := &discoveries{hc: hc, now: now}
	return func(ctx context.Context, c ConnectConfig) (IdP, error) {
		p, err := d.provider(ctx, c.Issuer, c.Fresh)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot reach the provider at %s: %s", apperrs.ErrInvalid, c.Issuer, err)
		}
		scopes := []string{oidc.ScopeOpenID, "profile", "email"}
		var meta struct {
			ScopesSupported []string `json:"scopes_supported"`
		}
		if err := p.Claims(&meta); err == nil && slices.Contains(meta.ScopesSupported, oidc.ScopeOfflineAccess) {
			scopes = append(scopes, oidc.ScopeOfflineAccess)
		}
		return &provider{
			hc:       hc,
			secret:   secret,
			verifier: p.Verifier(&oidc.Config{ClientID: c.ClientID}),
			oauth: oauth2.Config{
				ClientID: c.ClientID, ClientSecret: c.ClientSecret, Endpoint: p.Endpoint(), RedirectURL: c.RedirectURI,
				Scopes: scopes,
			},
		}, nil
	}
}

// discoveries holds the one issuer's discovered provider, whose key set is fetched once and kept with it; the instance
// has one provider, so a changed issuer simply replaces it.
type discoveries struct {
	hc     *http.Client
	now    func() time.Time
	mu     sync.Mutex
	issuer string
	p      *oidc.Provider
	at     time.Time
}

func (d *discoveries) provider(ctx context.Context, issuer string, fresh bool) (*oidc.Provider, error) {
	d.mu.Lock()
	if !fresh && d.p != nil && d.issuer == issuer && d.now().Sub(d.at) < discoveryTTL {
		p := d.p
		d.mu.Unlock()
		return p, nil
	}
	d.mu.Unlock()
	p, err := oidc.NewProvider(oidc.ClientContext(ctx, d.hc), issuer)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	d.issuer, d.p, d.at = issuer, p, d.now()
	d.mu.Unlock()
	return p, nil
}

type provider struct {
	hc       *http.Client
	secret   []byte
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
}

// derive is a value only this server can compute from the state, base64url so it fits a nonce or a PKCE verifier.
func (p *provider) derive(label, state string) string {
	m := hmac.New(sha256.New, p.secret)
	_, _ = m.Write([]byte("sso-" + label + "." + state))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (p *provider) AuthURL(state string) string {
	return p.oauth.AuthCodeURL(state, oidc.Nonce(p.derive("nonce", state)), oauth2.S256ChallengeOption(p.derive("pkce", state)))
}

func (p *provider) Exchange(ctx context.Context, code, state string) (Grant, error) {
	ctx = oidc.ClientContext(ctx, p.hc)
	tok, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(p.derive("pkce", state)))
	var rejected *oauth2.RetrieveError
	if errors.As(err, &rejected) && (rejected.Response.StatusCode == http.StatusUnauthorized || rejected.ErrorCode == "invalid_client") {
		return Grant{}, fmt.Errorf("%w: %w: %s", apperrs.ErrUnauthorized, errClientRejected, rejected.ErrorCode)
	}
	if err != nil {
		return Grant{}, fmt.Errorf("%w: exchange the sign-in code: %s", apperrs.ErrUnauthorized, err)
	}
	raw, _ := tok.Extra("id_token").(string)
	if raw == "" {
		return Grant{}, fmt.Errorf("%w: the provider returned no ID token", apperrs.ErrUnauthorized)
	}
	id, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return Grant{}, fmt.Errorf("%w: verify the ID token: %s", apperrs.ErrUnauthorized, err)
	}
	if id.Nonce != p.derive("nonce", state) {
		return Grant{}, fmt.Errorf("%w: the ID token answers another sign-in", apperrs.ErrUnauthorized)
	}
	var c struct {
		Email             string `json:"email"`
		EmailVerified     any    `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := id.Claims(&c); err != nil {
		return Grant{}, fmt.Errorf("%w: read the ID token's claims: %s", apperrs.ErrUnauthorized, err)
	}
	name := c.Name
	if name == "" {
		name = c.PreferredUsername
	}
	return Grant{
		Claims:       Claims{Subject: id.Subject, Email: c.Email, EmailVerified: verified(c.EmailVerified), Name: name, Picture: c.Picture},
		RefreshToken: tok.RefreshToken,
	}, nil
}

func (p *provider) Refresh(ctx context.Context, refreshToken string) (string, error) {
	ctx = oidc.ClientContext(ctx, p.hc)
	tok, err := p.oauth.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	var refused *oauth2.RetrieveError
	if errors.As(err, &refused) && refused.ErrorCode == "invalid_grant" {
		return "", fmt.Errorf("%w: %s", errGrantRefused, refused.ErrorCode)
	}
	// Only invalid_grant says the person is gone: a bare 401 can be a broken client, which must sign nobody out.
	if err != nil {
		return "", fmt.Errorf("%w: %s", errProviderUnavailable, err)
	}
	return tok.RefreshToken, nil
}

// verified reads email_verified, which some providers send as the string "true".
func verified(v any) bool {
	switch v := v.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

package sso

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// errClientRejected marks a code exchange the provider refused for its client credentials, the one case a re-read
// secret reference may fix.
var errClientRejected = errors.New("client credentials rejected")

// ConnectConfig is what reaching the provider takes.
type ConnectConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// IdP is the slice of the identity provider a sign-in calls, fakeable in tests.
type IdP interface {
	// AuthURL is where the browser goes to sign in; state is the CSRF token the callback must return.
	AuthURL(state string) string
	// Exchange trades the callback's code for the verified ID token's claims.
	Exchange(ctx context.Context, code, state string) (Claims, error)
}

// Connect reaches the provider, which discovery does through its issuer URL.
type Connect func(ctx context.Context, c ConnectConfig) (IdP, error)

// dial discovers the provider from its issuer URL. The nonce and the PKCE verifier are derived from the state under the
// server's secret, so nothing but the state cookie has to survive the round trip.
// ponytail: discovery and the key set are fetched on every start and callback; cache the provider if its latency shows.
func dial(hc *http.Client, secret []byte) Connect {
	return func(ctx context.Context, c ConnectConfig) (IdP, error) {
		ctx = oidc.ClientContext(ctx, hc)
		p, err := oidc.NewProvider(ctx, c.Issuer)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot reach the provider at %s: %s", apperrs.ErrInvalid, c.Issuer, err)
		}
		return &provider{
			hc:       hc,
			secret:   secret,
			verifier: p.Verifier(&oidc.Config{ClientID: c.ClientID}),
			oauth: oauth2.Config{
				ClientID: c.ClientID, ClientSecret: c.ClientSecret, Endpoint: p.Endpoint(), RedirectURL: c.RedirectURI,
				Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
			},
		}, nil
	}
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

func (p *provider) Exchange(ctx context.Context, code, state string) (Claims, error) {
	ctx = oidc.ClientContext(ctx, p.hc)
	tok, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(p.derive("pkce", state)))
	var rejected *oauth2.RetrieveError
	if errors.As(err, &rejected) && (rejected.Response.StatusCode == http.StatusUnauthorized || rejected.ErrorCode == "invalid_client") {
		return Claims{}, fmt.Errorf("%w: %w: %s", apperrs.ErrUnauthorized, errClientRejected, rejected.ErrorCode)
	}
	if err != nil {
		return Claims{}, fmt.Errorf("%w: exchange the sign-in code: %s", apperrs.ErrUnauthorized, err)
	}
	raw, _ := tok.Extra("id_token").(string)
	if raw == "" {
		return Claims{}, fmt.Errorf("%w: the provider returned no ID token", apperrs.ErrUnauthorized)
	}
	id, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: verify the ID token: %s", apperrs.ErrUnauthorized, err)
	}
	if id.Nonce != p.derive("nonce", state) {
		return Claims{}, fmt.Errorf("%w: the ID token answers another sign-in", apperrs.ErrUnauthorized)
	}
	var c struct {
		Email             string `json:"email"`
		EmailVerified     any    `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := id.Claims(&c); err != nil {
		return Claims{}, fmt.Errorf("%w: read the ID token's claims: %s", apperrs.ErrUnauthorized, err)
	}
	name := c.Name
	if name == "" {
		name = c.PreferredUsername
	}
	return Claims{Subject: id.Subject, Email: c.Email, EmailVerified: verified(c.EmailVerified), Name: name, Picture: c.Picture}, nil
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

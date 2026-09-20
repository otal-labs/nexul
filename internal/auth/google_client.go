package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/oauthx"
)

const (
	googleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL     = "https://oauth2.googleapis.com/token"
	googleUserInfoURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

// HTTPGoogleClient talks to Google's OAuth + OpenID userinfo (ADR 0040); userinfo is trusted without verifying an id_token.
type HTTPGoogleClient struct {
	client       *http.Client
	clientID     string
	clientSecret string
	redirectURI  string
	tokenURL     string
	userURL      string
}

// NewHTTPGoogleClient wires a real Google client; redirectURI must match the authorize call, or Google rejects it.
func NewHTTPGoogleClient(clientID, clientSecret, redirectURI string, hc *http.Client) *HTTPGoogleClient {
	return &HTTPGoogleClient{
		client:       hc,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		tokenURL:     googleTokenURL,
		userURL:      googleUserInfoURL,
	}
}

// Exchange trades an authorization code for an access token.
func (c *HTTPGoogleClient) Exchange(ctx context.Context, code string) (string, error) {
	return exchangeCode(ctx, c.client, "google", c.tokenURL, c.clientID, c.clientSecret, c.redirectURI, code)
}

// exchangeCode is the standard authorization_code grant, shared by Google and Discord; GitHub has its own Exchange.
func exchangeCode(ctx context.Context, hc *http.Client, name, tokenURL, clientID, clientSecret, redirectURI, code string) (string, error) {
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("%w: %s OAuth is not configured", apperrs.ErrInvalid, name)
	}
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if _, err := oauthx.PostForm(ctx, hc, tokenURL, form, "", "", "application/json", &body); err != nil {
		return "", fmt.Errorf("exchange code: %w", apperrs.Retryable(err))
	}
	if body.Error != "" {
		return "", fmt.Errorf("%w: %s: %s", apperrs.ErrUnauthorized, name, body.Error)
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("%w: %s returned no access token", apperrs.ErrUnauthorized, name)
	}
	return body.AccessToken, nil
}

// FetchUser fetches the OpenID identity: sub is the stable key, lowercased email is what the allowlist matches.
func (c *HTTPGoogleClient) FetchUser(ctx context.Context, accessToken string) (user *ProviderUser, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", apperrs.Retryable(err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var body struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fetch user: %w", apperrs.Retryable(err))
	}
	if body.Sub == "" || body.Email == "" {
		return nil, fmt.Errorf("%w: google returned an incomplete user", apperrs.ErrUnauthorized)
	}
	if !body.EmailVerified {
		return nil, fmt.Errorf("%w: google account email is not verified", apperrs.ErrUnauthorized)
	}
	return &ProviderUser{
		ID:        body.Sub,
		Login:     strings.ToLower(body.Email),
		Name:      body.Name,
		AvatarURL: body.Picture,
	}, nil
}

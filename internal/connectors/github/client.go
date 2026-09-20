// Package github implements connectors.OAuthClient against a GitHub App, reading AppConfig fresh on every call.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
	"github.com/otal-labs/nexul/internal/platform/oauthx"
)

// var so tests can point at a stub server.
var (
	installBaseURL    = "https://github.com/apps"
	oauthAuthorizeURL = "https://github.com/login/oauth/authorize"
	oauthTokenURL     = "https://github.com/login/oauth/access_token"
	apiBaseURL        = "https://api.github.com"
)

// InstanceURLReader resolves the instance's public URL live, since redirect_uri must track the current setting (T5).
type InstanceURLReader interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// OAuthClient implements connectors.OAuthClient for one connector id against its currently stored AppConfig.
type OAuthClient struct {
	connectorID string
	store       connectors.AppConfigStore
	instanceURL InstanceURLReader
	httpc       *http.Client
}

// New wires the GitHub App OAuth client; hc defaults to a 15s-timeout client when nil.
func New(connectorID string, store connectors.AppConfigStore, instanceURL InstanceURLReader, hc *http.Client) *OAuthClient {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &OAuthClient{connectorID: connectorID, store: store, instanceURL: instanceURL, httpc: hc}
}

// VerifyApp implements connectors.AppVerifier: the slug must resolve to an app with this client ID that accepts the secret.
func (c *OAuthClient) VerifyApp(ctx context.Context, cfg connectors.AppConfig) error {
	apiBase := apiBaseURL
	if cfg.BaseURL != "" {
		apiBase = strings.TrimRight(cfg.BaseURL, "/") + "/api/v3"
	}
	return githubapp.Verify(ctx, c.httpc, apiBase, cfg.ClientID, cfg.ClientSecret, cfg.AppSlug)
}

// config reads the connector's current AppConfig fresh: no caching, no startup snapshot (T13a).
func (c *OAuthClient) config(ctx context.Context) (connectors.AppConfig, error) {
	return c.store.GetAppConfig(ctx, c.connectorID)
}

// callbackURL builds the registered callback from the current instance URL; inlined since this can't import auth (ADR 0017).
func (c *OAuthClient) callbackURL(ctx context.Context) (string, error) {
	u, err := c.instanceURL.GetInstanceURL(ctx)
	if err != nil {
		return "", fmt.Errorf("get instance url: %w", err)
	}
	return strings.TrimRight(u, "/") + "/auth/connectors/github/callback", nil
}

// Configured implements connectors.OAuthClient; AppSlug plus client id/secret are needed to build the install URL.
func (c *OAuthClient) Configured() bool {
	cfg, err := c.config(context.Background())
	if err != nil {
		return false
	}
	return cfg.Configured() && cfg.AppSlug != ""
}

// AuthorizeURL uses the classic authorize flow: on an installed app, the install URL dead-ends every reconnect.
func (c *OAuthClient) AuthorizeURL(state string) string {
	ctx := context.Background()
	cfg, err := c.config(ctx)
	if err != nil {
		return ""
	}
	q := url.Values{"client_id": {cfg.ClientID}, "state": {state}}
	if cb, err := c.callbackURL(ctx); err == nil && cb != "" {
		q.Set("redirect_uri", cb)
	}
	return oauthAuthorizeURL + "?" + q.Encode()
}

// installURL is used only when Exchange finds no installation for a first-run user.
func (c *OAuthClient) installURL(ctx context.Context, cfg connectors.AppConfig) string {
	u := installBaseURL + "/" + cfg.AppSlug + "/installations/new"
	if cb, err := c.callbackURL(ctx); err == nil && cb != "" {
		u += "?" + url.Values{"redirect_uri": {cb}}.Encode()
	}
	return u
}

// Exchange trades the code for a token, then confirms an installation exists; a token alone grants no repo access.
func (c *OAuthClient) Exchange(ctx context.Context, code string) (*connectors.TokenSet, error) {
	cfg, err := c.config(ctx)
	if err != nil {
		return nil, fmt.Errorf("get github app config: %w", err)
	}
	if !cfg.Configured() {
		return nil, fmt.Errorf("%w: github app is not configured", apperrs.ErrInvalid)
	}
	form := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
	}
	ts, err := c.token(ctx, form)
	if err != nil {
		return nil, err
	}
	if err := c.requireInstalled(ctx, cfg, ts.AccessToken); err != nil {
		return nil, err
	}
	return ts, nil
}

// Refresh implements connectors.OAuthClient.
func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*connectors.TokenSet, error) {
	cfg, err := c.config(ctx)
	if err != nil {
		return nil, fmt.Errorf("get github app config: %w", err)
	}
	if !cfg.Configured() {
		return nil, fmt.Errorf("%w: github app is not configured", apperrs.ErrInvalid)
	}
	form := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	return c.token(ctx, form)
}

// Revoke calls GitHub's documented DELETE /applications/{client_id}/token revocation endpoint.
func (c *OAuthClient) Revoke(ctx context.Context, accessToken string) (err error) {
	cfg, err := c.config(ctx)
	if err != nil {
		return fmt.Errorf("get github app config: %w", err)
	}
	if !cfg.Configured() {
		return nil // nothing registered to revoke against
	}
	payload, err := json.Marshal(map[string]string{"access_token": accessToken})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		apiBaseURL+"/applications/"+cfg.ClientID+"/token", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("github revoke token: %w", apperrs.Retryable(err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 500 {
		return apperrs.Retryable(fmt.Errorf("github revoke token: status %d", resp.StatusCode))
	}
	// 404 means the token is already invalid/gone — treat as already disconnected, not an error.
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("%w: github revoke token: status %d", apperrs.ErrUnauthorized, resp.StatusCode)
	}
	return nil
}

// requireInstalled implements the "authorizing != installing" check.
func (c *OAuthClient) requireInstalled(ctx context.Context, cfg connectors.AppConfig, accessToken string) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+"/user/installations", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("github list installations: %w", apperrs.Retryable(err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read github installations response: %w", apperrs.Retryable(err))
	}
	if resp.StatusCode >= 500 {
		return apperrs.Retryable(fmt.Errorf("github list installations: status %d", resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: github list installations: status %d", apperrs.ErrUnauthorized, resp.StatusCode)
	}
	var out struct {
		TotalCount int `json:"total_count"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decode github installations response: %w", apperrs.Retryable(err))
	}
	if out.TotalCount == 0 {
		return &connectors.NotInstalledError{InstallURL: c.installURL(ctx, cfg)}
	}
	return nil
}

// token is the shared token caller for Exchange and Refresh; Accept: application/json overrides GitHub's form default.
func (c *OAuthClient) token(ctx context.Context, form url.Values) (*connectors.TokenSet, error) {
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	status, err := oauthx.PostForm(ctx, c.httpc, oauthTokenURL, form, "", "", "application/json", &out)
	if err != nil {
		return nil, fmt.Errorf("github oauth token: %w", apperrs.Retryable(err))
	}
	if status >= 500 {
		return nil, apperrs.Retryable(fmt.Errorf("github oauth token: status %d", status))
	}
	if status >= 400 || out.Error != "" || out.AccessToken == "" {
		msg := out.Error
		if out.ErrorDesc != "" {
			msg += ": " + out.ErrorDesc
		}
		// A rejected grant is permanent, not retryable — the user must reconnect.
		return nil, fmt.Errorf("%w: github oauth: %s", apperrs.ErrUnauthorized, msg)
	}
	return &connectors.TokenSet{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresIn:    time.Duration(out.ExpiresIn) * time.Second,
	}, nil
}

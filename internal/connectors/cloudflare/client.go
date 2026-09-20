// Package cloudflare implements connectors.OAuthClient against Cloudflare's OAuth app, reading credentials live.
package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/oauthx"
)

// var so tests can point at a stub server.
var (
	oauthAuthorizeURL = "https://dash.cloudflare.com/oauth2/auth"
	oauthTokenURL     = "https://dash.cloudflare.com/oauth2/token"
)

// oauthScopes requests refresh tokens plus read+edit on DNS records; the OAuth client must offer these names.
const oauthScopes = "offline_access DNS Read DNS Write"

// InstanceURLReader resolves the instance's public URL live, since redirect_uri must track the current setting.
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

// New wires the Cloudflare OAuth client; hc defaults to a 15s-timeout client when nil.
func New(connectorID string, store connectors.AppConfigStore, instanceURL InstanceURLReader, hc *http.Client) *OAuthClient {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &OAuthClient{connectorID: connectorID, store: store, instanceURL: instanceURL, httpc: hc}
}

// config reads the connector's current AppConfig fresh: no caching, no startup snapshot (T13a).
func (c *OAuthClient) config(ctx context.Context) (connectors.AppConfig, error) {
	return c.store.GetAppConfig(ctx, c.connectorID)
}

// callbackURL builds this connector's registered callback from the current instance URL.
func (c *OAuthClient) callbackURL(ctx context.Context) (string, error) {
	u, err := c.instanceURL.GetInstanceURL(ctx)
	if err != nil {
		return "", fmt.Errorf("get instance url: %w", err)
	}
	return strings.TrimRight(u, "/") + "/auth/connectors/cloudflare/callback", nil
}

// Configured implements connectors.OAuthClient over a local encrypted-at-rest read, so Background() is fine.
func (c *OAuthClient) Configured() bool {
	cfg, err := c.config(context.Background())
	if err != nil {
		return false
	}
	return cfg.Configured()
}

// AuthorizeURL implements connectors.OAuthClient.
func (c *OAuthClient) AuthorizeURL(state string) string {
	ctx := context.Background()
	cfg, err := c.config(ctx)
	if err != nil {
		return ""
	}
	q := url.Values{
		"response_type": {"code"},
		"client_id":     {cfg.ClientID},
		"state":         {state},
		"scope":         {oauthScopes},
	}
	if cb, err := c.callbackURL(ctx); err == nil && cb != "" {
		q.Set("redirect_uri", cb)
	}
	return oauthAuthorizeURL + "?" + q.Encode()
}

// Exchange trades an authorization code for a token set (Authorization Code flow, client_secret_basic).
func (c *OAuthClient) Exchange(ctx context.Context, code string) (*connectors.TokenSet, error) {
	cfg, err := c.config(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cloudflare app config: %w", err)
	}
	if !cfg.Configured() {
		return nil, fmt.Errorf("%w: cloudflare app is not configured", apperrs.ErrInvalid)
	}
	cb, err := c.callbackURL(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"client_id":    {cfg.ClientID},
		"redirect_uri": {cb},
		"code":         {code},
	}
	return c.token(ctx, cfg, form)
}

// Refresh implements connectors.OAuthClient.
func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*connectors.TokenSet, error) {
	cfg, err := c.config(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cloudflare app config: %w", err)
	}
	if !cfg.Configured() {
		return nil, fmt.Errorf("%w: cloudflare app is not configured", apperrs.ErrInvalid)
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.ClientID},
		"refresh_token": {refreshToken},
	}
	return c.token(ctx, cfg, form)
}

// Revoke is a no-op: Cloudflare has no revocation endpoint; clearing local state is what stops this side using it.
func (c *OAuthClient) Revoke(_ context.Context, _ string) error {
	return nil
}

func (c *OAuthClient) token(ctx context.Context, cfg connectors.AppConfig, form url.Values) (*connectors.TokenSet, error) {
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
	}
	status, err := oauthx.PostForm(ctx, c.httpc, oauthTokenURL, form, cfg.ClientID, cfg.ClientSecret, "", &out)
	if err != nil {
		return nil, fmt.Errorf("cloudflare oauth token: %w", apperrs.Retryable(err))
	}
	if status >= 500 {
		return nil, apperrs.Retryable(fmt.Errorf("cloudflare oauth token: status %d", status))
	}
	if status >= 400 || out.Error != "" || out.AccessToken == "" {
		// A rejected grant is permanent — the user must reconnect.
		return nil, fmt.Errorf("%w: cloudflare oauth: %s", apperrs.ErrUnauthorized, out.Error)
	}
	return &connectors.TokenSet{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresIn:    time.Duration(out.ExpiresIn) * time.Second,
	}, nil
}

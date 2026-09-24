package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type serviceTokenEnvelope struct {
	ID           string `json:"id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// CreateAccessApp implements dns.AccessAppProvider; the 401 redirect gives a blocked API client a clean 401, not the login page.
func (c *Client) CreateAccessApp(ctx context.Context, hostname, serviceTokenID string) (string, error) {
	body := map[string]any{
		"type":                      "self_hosted",
		"name":                      hostname,
		"domain":                    hostname,
		"app_launcher_visible":      false,
		"service_auth_401_redirect": true,
		"policies": []map[string]any{{
			"name":     "Nexul server",
			"decision": "non_identity",
			"include":  []map[string]any{{"service_token": map[string]string{"token_id": serviceTokenID}}},
		}},
	}
	var out apiResponse
	if err := c.accessDo(ctx, http.MethodPost, "access/apps", body, &out); err != nil {
		return "", err
	}
	var app struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out.Result, &app); err != nil {
		return "", fmt.Errorf("decode access app: %w", err)
	}
	if app.ID == "" {
		return "", apperrs.Retryable(fmt.Errorf("create access app for %s: empty id in response", hostname))
	}
	return app.ID, nil
}

// DeleteAccessApp implements dns.AccessAppProvider.
func (c *Client) DeleteAccessApp(ctx context.Context, appID string) error {
	var out apiResponse
	return ignoreNotFound(c.accessDo(ctx, http.MethodDelete, "access/apps/"+url.PathEscape(appID), nil, &out))
}

// CreateServiceToken implements dns.ServiceTokenProvider; the token never expires, so no refresh schedule is needed.
func (c *Client) CreateServiceToken(ctx context.Context, name string) (*dns.ServiceToken, error) {
	var out apiResponse
	body := map[string]string{"name": name, "duration": "forever"}
	if err := c.accessDo(ctx, http.MethodPost, "access/service_tokens", body, &out); err != nil {
		return nil, err
	}
	return decodeServiceToken(out.Result)
}

// RotateServiceToken implements dns.ServiceTokenProvider; the old secret stops working at once, the id and client id stay.
func (c *Client) RotateServiceToken(ctx context.Context, tokenID string) (*dns.ServiceToken, error) {
	var out apiResponse
	path := "access/service_tokens/" + url.PathEscape(tokenID) + "/rotate"
	if err := c.accessDo(ctx, http.MethodPost, path, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return decodeServiceToken(out.Result)
}

// DeleteServiceToken implements dns.ServiceTokenProvider.
func (c *Client) DeleteServiceToken(ctx context.Context, tokenID string) error {
	var out apiResponse
	return ignoreNotFound(c.accessDo(ctx, http.MethodDelete, "access/service_tokens/"+url.PathEscape(tokenID), nil, &out))
}

// accessDo runs an account-scoped Access call, turning a missing Zero Trust organization into its own error.
func (c *Client) accessDo(ctx context.Context, method, path string, body any, out *apiResponse) error {
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	err = c.do(ctx, method, "accounts/"+url.PathEscape(acct)+"/"+path, body, out)
	if err == nil || !zeroTrustMissing(err.Error()) {
		return err
	}
	return fmt.Errorf("%w: %w: enable Zero Trust once in the Cloudflare dashboard (pick a team name and the Free plan), then try again", apperrs.ErrInvalid, dns.ErrZeroTrustDisabled)
}

// zeroTrustMissing matches Cloudflare's "no Access organization" errors by phrase; the exact wording is unverified live.
func zeroTrustMissing(msg string) bool {
	msg = strings.ToLower(msg)
	for _, phrase := range []string{"access organization", "organization not found", "organization_not_found", "not enabled"} {
		if strings.Contains(msg, phrase) {
			return true
		}
	}
	return false
}

func decodeServiceToken(raw json.RawMessage) (*dns.ServiceToken, error) {
	var e serviceTokenEnvelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("decode service token: %w", err)
	}
	if e.ID == "" || e.ClientID == "" || e.ClientSecret == "" {
		return nil, apperrs.Retryable(errors.New("service token response is missing its id, client id, or secret"))
	}
	return &dns.ServiceToken{ID: e.ID, ClientID: e.ClientID, ClientSecret: e.ClientSecret}, nil
}

func ignoreNotFound(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	return err
}

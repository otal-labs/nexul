package t3client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// wellKnownEnvironmentPath is T3's unauthenticated environment descriptor endpoint, used before any credential exists.
const wellKnownEnvironmentPath = "/.well-known/t3/environment"

// clientScopes mirrors T3's AuthStandardClientScopes; `t3 pair` grants exactly this set.
const clientScopes = "orchestration:read orchestration:operate terminal:operate review:write relay:read"

const maxResponseBytes = 1 << 20 // 1MiB: generous for a token/JSON response, small enough to bound a hostile/broken server.

// exchangeResult is the bearer session returned by the RFC 8693 token exchange.
type exchangeResult struct {
	BearerToken string
	ExpiresIn   time.Duration
}

type tokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Pair implements harness.Client: the `t3 pair` one-time token is traded for a bearer session, then the version is read.
func (h *Harness) Pair(ctx context.Context, serverURL, secret string) (harness.PairResult, error) {
	ex, err := h.exchange(ctx, serverURL, secret)
	if err != nil {
		return harness.PairResult{}, fmt.Errorf("exchange pairing token: %w", err)
	}
	version, err := h.Version(ctx, serverURL)
	if err != nil {
		return harness.PairResult{}, fmt.Errorf("read T3 version: %w", err)
	}
	return harness.PairResult{BearerToken: ex.BearerToken, ExpiresIn: ex.ExpiresIn, Version: version}, nil
}

// exchange trades a `t3 pair` one-time token for a bearer session (RFC 8693 token-exchange grant).
func (h *Harness) exchange(ctx context.Context, serverURL, oneTimeToken string) (exchangeResult, error) {
	form := url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":        {oneTimeToken},
		"subject_token_type":   {"urn:t3:params:oauth:token-type:environment-bootstrap"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"scope":                {clientScopes},
		// Presentation hints for T3's authorized-clients UI; without them the row is labeled "one-time-token".
		"client_label":       {"Nexul"},
		"client_device_type": {"desktop"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return exchangeResult{}, fmt.Errorf("build token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := h.do(req)
	if err != nil {
		return exchangeResult{}, err
	}
	var out tokenExchangeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return exchangeResult{}, fmt.Errorf("decode T3 token exchange response: %w", err)
	}
	if out.AccessToken == "" {
		return exchangeResult{}, fmt.Errorf("%w: T3 token exchange returned no access token", apperrs.ErrInvalid)
	}
	expiresIn := time.Duration(out.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 30 * 24 * time.Hour // T3's DEFAULT_SESSION_TTL: plain bearer sessions are 30 days.
	}
	return exchangeResult{BearerToken: out.AccessToken, ExpiresIn: expiresIn}, nil
}

type environmentDescriptor struct {
	ServerVersion string `json:"serverVersion"`
}

// Version implements harness.Client via the unauthenticated well-known probe.
func (h *Harness) Version(ctx context.Context, serverURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(serverURL, "/")+wellKnownEnvironmentPath, nil)
	if err != nil {
		return "", fmt.Errorf("build version probe request: %w", err)
	}
	body, err := h.do(req)
	if err != nil {
		return "", err
	}
	var out environmentDescriptor
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode T3 environment descriptor: %w", err)
	}
	if out.ServerVersion == "" {
		return "", fmt.Errorf("%w: T3 environment descriptor has no serverVersion", apperrs.ErrInvalid)
	}
	return out.ServerVersion, nil
}

// do maps unreachable-server failures to ErrRetryable and non-200 responses to ErrInvalid (retrying won't help).
func (h *Harness) do(req *http.Request) (body []byte, err error) {
	resp, err := h.httpClient().Do(req)
	if err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("reach T3 server: %w", err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	body, err = io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read T3 response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: T3 server responded %d: %s", apperrs.ErrInvalid, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (h *Harness) httpClient() *http.Client {
	if h.Options.HTTPClient == nil {
		return http.DefaultClient
	}
	return h.Options.HTTPClient
}

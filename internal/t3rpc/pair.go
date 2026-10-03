package t3rpc

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

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// wellKnownEnvironmentPath is T3's unauthenticated environment descriptor endpoint, used before any credential exists.
const wellKnownEnvironmentPath = "/.well-known/t3/environment"

// clientScopes mirrors T3's AuthStandardClientScopes; `t3 pair` grants exactly this set.
const clientScopes = "orchestration:read orchestration:operate terminal:operate review:write relay:read"

const maxResponseBytes = 1 << 20 // 1MiB: generous for a token/JSON response, small enough to bound a hostile/broken server.

type tokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Exchange trades a `t3 pair` one-time token for a bearer session (RFC 8693 token-exchange grant).
func Exchange(ctx context.Context, client *http.Client, serverURL, oneTimeToken string) (bearerToken string, expiresIn time.Duration, err error) {
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
		return "", 0, fmt.Errorf("build token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := do(client, req)
	if err != nil {
		return "", 0, err
	}
	var out tokenExchangeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", 0, fmt.Errorf("decode T3 token exchange response: %w", err)
	}
	if out.AccessToken == "" {
		return "", 0, fmt.Errorf("%w: T3 token exchange returned no access token", apperrs.ErrInvalid)
	}
	expiresIn = time.Duration(out.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 30 * 24 * time.Hour // T3's DEFAULT_SESSION_TTL: plain bearer sessions are 30 days.
	}
	return out.AccessToken, expiresIn, nil
}

// Descriptor is what T3's environment descriptor says about the server; getConfig's environment carries the same.
type Descriptor struct {
	ServerVersion string `json:"serverVersion"`
	Protocol      int    `json:"orchestrationProtocolVersion"`
}

// Describe reads the unauthenticated well-known descriptor, the one probe that works before any credential exists.
func Describe(ctx context.Context, client *http.Client, serverURL string) (Descriptor, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(serverURL, "/")+wellKnownEnvironmentPath, nil)
	if err != nil {
		return Descriptor{}, fmt.Errorf("build version probe request: %w", err)
	}
	body, err := do(client, req)
	if err != nil {
		return Descriptor{}, err
	}
	var out Descriptor
	if err := json.Unmarshal(body, &out); err != nil {
		return Descriptor{}, fmt.Errorf("decode T3 environment descriptor: %w", err)
	}
	if out.ServerVersion == "" {
		return Descriptor{}, fmt.Errorf("%w: T3 environment descriptor has no serverVersion", apperrs.ErrInvalid)
	}
	out.Protocol = protocolOrOne(out.Protocol)
	return out, nil
}

// protocolOrOne reads an absent orchestration protocol as 1: T3 before protocol negotiation sends none.
func protocolOrOne(protocol int) int {
	if protocol == 0 {
		return 1
	}
	return protocol
}

// do maps unreachable-server failures to ErrRetryable and non-200 responses to ErrInvalid (retrying won't help).
func do(client *http.Client, req *http.Request) (body []byte, err error) {
	resp, err := httpClient(client).Do(req)
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

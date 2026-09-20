package cloudflare

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

// var so tests can point at a stub server.
var apiBaseURL = "https://api.cloudflare.com/client/v4"

// TokenVerifier implements connectors.Verifier for the manual api_token field: token active, then each permission DNS and tunnels need.
type TokenVerifier struct {
	httpc *http.Client
}

// NewTokenVerifier builds the verifier; hc defaults to a 15s-timeout client when nil.
func NewTokenVerifier(hc *http.Client) *TokenVerifier {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &TokenVerifier{httpc: hc}
}

// Verify walks the calls the product will make later, so a token missing a permission fails here, not on the first deploy.
func (v *TokenVerifier) Verify(ctx context.Context, fields map[string]string) error {
	token := fields["api_token"]
	if err := v.verifyActive(ctx, token); err != nil {
		return err
	}
	zone, err := v.firstZone(ctx, token)
	if err != nil {
		return err
	}
	if err := v.requireWrite(ctx, token, "zones/"+url.PathEscape(zone.ID)+"/dns_records", "Zone → DNS: Edit"); err != nil {
		return err
	}
	return v.requireWrite(ctx, token, "accounts/"+url.PathEscape(zone.Account.ID)+"/cfd_tunnel", "Account → Cloudflare Tunnel: Edit")
}

// VerifyCheck implements connectors.CheckVerifier; every permission check first needs the zone it runs against.
func (v *TokenVerifier) VerifyCheck(ctx context.Context, fields map[string]string, key string) error {
	token := fields["api_token"]
	if key == "token" {
		return v.verifyActive(ctx, token)
	}
	zone, err := v.firstZone(ctx, token)
	if err != nil {
		return err
	}
	switch key {
	case "zone_read":
		return nil
	case "dns_edit":
		return v.requireWrite(ctx, token, "zones/"+url.PathEscape(zone.ID)+"/dns_records", "Zone → DNS: Edit")
	case "tunnel_edit":
		return v.requireWrite(ctx, token, "accounts/"+url.PathEscape(zone.Account.ID)+"/cfd_tunnel", "Account → Cloudflare Tunnel: Edit")
	default:
		return fmt.Errorf("%w: unknown check %q", apperrs.ErrInvalid, key)
	}
}

type verifyZone struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Account struct {
		ID string `json:"id"`
	} `json:"account"`
}

// verifyActive rejects anything Cloudflare does not report as an active token.
func (v *TokenVerifier) verifyActive(ctx context.Context, token string) error {
	var out struct {
		Status string `json:"status"`
	}
	status, err := v.get(ctx, token, "user/tokens/verify", &out)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("%w: Cloudflare rejected the token", apperrs.ErrInvalid)
	}
	if out.Status != "active" {
		return fmt.Errorf("%w: token is %s, not active", apperrs.ErrInvalid, out.Status)
	}
	return nil
}

// firstZone proves "Zone → Zone: Read" and yields the zone and account the other checks run against.
func (v *TokenVerifier) firstZone(ctx context.Context, token string) (verifyZone, error) {
	var zones []verifyZone
	status, err := v.get(ctx, token, "zones?per_page=1", &zones)
	if err != nil {
		return verifyZone{}, err
	}
	if status == http.StatusForbidden || status == http.StatusUnauthorized {
		return verifyZone{}, fmt.Errorf("%w: the token cannot list zones — add Zone → Zone: Read", apperrs.ErrInvalid)
	}
	if status != http.StatusOK {
		return verifyZone{}, fmt.Errorf("cloudflare zones: status %d", status)
	}
	if len(zones) == 0 {
		return verifyZone{}, fmt.Errorf("%w: the token can see no zone — include your zone under Zone → Zone: Read", apperrs.ErrInvalid)
	}
	return zones[0], nil
}

// requireWrite proves an Edit permission without changing anything: Cloudflare authorises before it validates, so an
// empty POST body answers 400 (allowed, rejected as malformed) when the permission is there and 401/403 when it is not.
func (v *TokenVerifier) requireWrite(ctx context.Context, token, path, permission string) error {
	status, err := v.call(ctx, http.MethodPost, token, path, nil)
	if err != nil {
		return err
	}
	switch {
	case status == http.StatusForbidden || status == http.StatusUnauthorized:
		return fmt.Errorf("%w: the token is missing %s", apperrs.ErrInvalid, permission)
	case status == http.StatusBadRequest, status < 300:
		return nil
	default:
		return fmt.Errorf("cloudflare %s: status %d", path, status)
	}
}

// get performs one authenticated GET and decodes the envelope's result when the body is readable.
func (v *TokenVerifier) get(ctx context.Context, token, path string, result any) (int, error) {
	return v.call(ctx, http.MethodGet, token, path, result)
}

// call performs one authenticated request; a success:false envelope on 200 is reported as 403, the way Cloudflare means it.
func (v *TokenVerifier) call(ctx context.Context, method, token, path string, result any) (status int, err error) {
	var body io.Reader
	if method != http.MethodGet {
		body = strings.NewReader("{}")
	}
	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+"/"+path, body)
	if err != nil {
		return 0, fmt.Errorf("build cloudflare request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.httpc.Do(req)
	if err != nil {
		return 0, fmt.Errorf("reach cloudflare: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var envelope struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return resp.StatusCode, nil
	}
	if !envelope.Success && resp.StatusCode == http.StatusOK {
		return http.StatusForbidden, nil
	}
	if !envelope.Success && resp.StatusCode == http.StatusBadRequest {
		return http.StatusBadRequest, nil
	}
	if len(envelope.Result) > 0 && result != nil {
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return resp.StatusCode, fmt.Errorf("decode cloudflare %s: %w", path, err)
		}
	}
	return resp.StatusCode, nil
}

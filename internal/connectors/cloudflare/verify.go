package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
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

// Verify walks the calls DNS and tunnels will make, so a token missing one fails here, not on the first deploy; the advisory
// Access checks run only through VerifyCheck, so they never block saving.
func (v *TokenVerifier) Verify(ctx context.Context, fields map[string]string) error {
	token := fields["api_token"]
	if err := v.verifyActive(ctx, token); err != nil {
		return err
	}
	zones, err := v.zones(ctx, token)
	if err != nil {
		return err
	}
	if _, err := v.dnsEdit(ctx, token, zones); err != nil {
		return err
	}
	return v.requireWrite(ctx, token, "accounts/"+url.PathEscape(zones[0].Account.ID)+"/cfd_tunnel", "Account → Cloudflare Tunnel: Edit")
}

// VerifyCheck implements connectors.CheckVerifier; every permission check first needs the zones it runs against.
func (v *TokenVerifier) VerifyCheck(ctx context.Context, fields map[string]string, key string) (string, error) {
	token := fields["api_token"]
	if key == "token" {
		return "", v.verifyActive(ctx, token)
	}
	zones, err := v.zones(ctx, token)
	if err != nil {
		return "", err
	}
	account := zones[0].Account.ID
	switch key {
	case "zone_read":
		return "", nil
	case "dns_edit":
		return v.dnsEdit(ctx, token, zones)
	case "tunnel_edit":
		return "", v.requireWrite(ctx, token, "accounts/"+url.PathEscape(account)+"/cfd_tunnel", "Account → Cloudflare Tunnel: Edit")
	case "access_apps_edit":
		return "", v.requireAccessEdit(ctx, token, http.MethodPost, "accounts/"+url.PathEscape(account)+"/access/apps", accessAppsPermission)
	case "access_tokens_edit":
		return "", v.requireAccessEdit(ctx, token, http.MethodDelete, "accounts/"+url.PathEscape(account)+"/access/service_tokens/"+nilUUID, accessTokensPermission)
	default:
		return "", fmt.Errorf("%w: unknown check %q", apperrs.ErrInvalid, key)
	}
}

// dnsEdit probes DNS Edit on every zone in parallel and names the domains the token can edit, and any it only reads.
func (v *TokenVerifier) dnsEdit(ctx context.Context, token string, zones []verifyZone) (string, error) {
	errs := make([]error, len(zones))
	var wg sync.WaitGroup
	for i, z := range zones {
		wg.Go(func() {
			errs[i] = v.requireWrite(ctx, token, "zones/"+url.PathEscape(z.ID)+"/dns_records", dnsEditPermission)
		})
	}
	wg.Wait()
	var editable, readOnly []string
	for i, err := range errs {
		if err == nil {
			editable = append(editable, zones[i].Name)
			continue
		}
		if !errors.Is(err, apperrs.ErrInvalid) {
			return "", err
		}
		readOnly = append(readOnly, zones[i].Name)
	}
	if len(editable) == 0 {
		return "", fmt.Errorf("%w: the token is missing %s on %s", apperrs.ErrInvalid, dnsEditPermission, strings.Join(readOnly, ", "))
	}
	detail := "Can edit DNS on " + strings.Join(editable, ", ")
	if len(readOnly) > 0 {
		detail += "; read-only on " + strings.Join(readOnly, ", ")
	}
	return detail, nil
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

// zones proves "Zone → Zone: Read" and lists the token's first 50 zones (one page); the first names the checks' account.
func (v *TokenVerifier) zones(ctx context.Context, token string) ([]verifyZone, error) {
	var zones []verifyZone
	status, err := v.get(ctx, token, "zones?per_page=50", &zones)
	if err != nil {
		return nil, err
	}
	if status == http.StatusForbidden || status == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: the token cannot list zones — add Zone → Zone: Read", apperrs.ErrInvalid)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("cloudflare zones: status %d", status)
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("%w: the token can see no zone — include your zone under Zone → Zone: Read", apperrs.ErrInvalid)
	}
	return zones, nil
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

const (
	dnsEditPermission      = "Zone → DNS: Edit"
	accessAppsPermission   = "Account → Access: Apps and Policies: Edit"
	accessTokensPermission = "Account → Access: Service Tokens: Edit"
	// nilUUID names no real service token, so the delete probe cannot change or mint anything.
	nilUUID = "00000000-0000-0000-0000-000000000000"
)

// requireAccessEdit probes Access Edit with a request Cloudflare authorises before validating (not an app delete, which 404s unpermitted).
func (v *TokenVerifier) requireAccessEdit(ctx context.Context, token, method, path, permission string) error {
	status, env, err := v.send(ctx, method, token, path)
	if err != nil {
		return err
	}
	if env != nil && slices.ContainsFunc(env.Errors, func(e apiError) bool { return zeroTrustMissing(e.Message) }) {
		return fmt.Errorf("%w: Zero Trust is not enabled on this Cloudflare account, which pairing computers by tunnel needs — enable it once in the Cloudflare dashboard (pick a team name and the Free plan), then verify again", apperrs.ErrInvalid)
	}
	denied := env != nil && !env.Success && status == http.StatusOK
	switch {
	case denied || status == http.StatusForbidden || status == http.StatusUnauthorized:
		return fmt.Errorf("%w: the token is missing %s, needed only to pair computers by tunnel", apperrs.ErrInvalid, permission)
	case status == http.StatusNotFound, status == http.StatusBadRequest, status < 300:
		return nil
	default:
		return fmt.Errorf("cloudflare %s: status %d", path, status)
	}
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

// get performs one authenticated GET and decodes the envelope's result when the body is readable.
func (v *TokenVerifier) get(ctx context.Context, token, path string, result any) (int, error) {
	return v.call(ctx, http.MethodGet, token, path, result)
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []apiError      `json:"errors"`
}

// call performs one authenticated request; a success:false envelope on 200 is reported as 403, the way Cloudflare means it.
func (v *TokenVerifier) call(ctx context.Context, method, token, path string, result any) (int, error) {
	status, envelope, err := v.send(ctx, method, token, path)
	if err != nil || envelope == nil {
		return status, err
	}
	if !envelope.Success && status == http.StatusOK {
		return http.StatusForbidden, nil
	}
	if len(envelope.Result) > 0 && result != nil {
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return status, fmt.Errorf("decode cloudflare %s: %w", path, err)
		}
	}
	return status, nil
}

// send performs one authenticated request; the envelope is nil when the body is unreadable, leaving the status to speak.
func (v *TokenVerifier) send(ctx context.Context, method, token, path string) (status int, envelope *apiEnvelope, err error) {
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader("{}")
	}
	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+"/"+path, body)
	if err != nil {
		return 0, nil, fmt.Errorf("build cloudflare request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.httpc.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("reach cloudflare: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var env apiEnvelope
	if json.NewDecoder(resp.Body).Decode(&env) != nil {
		return resp.StatusCode, nil, nil
	}
	return resp.StatusCode, &env, nil
}

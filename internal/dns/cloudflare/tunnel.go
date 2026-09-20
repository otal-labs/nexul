package cloudflare

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Tunnel operations are account-scoped; account id comes from WithAccountID, else the first readable account, cached.
type tunnelState struct {
	mu        sync.Mutex
	accountID string
}

func (c *Client) accountID(ctx context.Context) (string, error) {
	if c.tunnel.accountID != "" {
		return c.tunnel.accountID, nil
	}
	c.tunnel.mu.Lock()
	defer c.tunnel.mu.Unlock()
	if c.tunnel.accountID != "" {
		return c.tunnel.accountID, nil
	}
	// The zone names the account its tunnels must live in; /accounts may list several (or none without Account Settings: Read).
	id, err := c.zoneAccountID(ctx)
	if err != nil || id == "" {
		id, err = c.firstAccountID(ctx)
	}
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("%w: the Cloudflare API token can see no account or zone — give it Zone → Zone: Read on your zone (or Account → Account Settings: Read)", apperrs.ErrInvalid)
	}
	c.tunnel.accountID = id
	return id, nil
}

// firstAccountID lists the accounts the token may read; empty when the token has no account-level read.
func (c *Client) firstAccountID(ctx context.Context) (string, error) {
	var out apiResponse
	if err := c.do(ctx, http.MethodGet, "accounts?per_page=50", nil, &out); err != nil {
		return "", err
	}
	var accounts []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out.Result, &accounts); err != nil {
		return "", fmt.Errorf("decode accounts: %w", err)
	}
	if len(accounts) == 0 {
		return "", nil
	}
	return accounts[0].ID, nil
}

// zoneAccountID reads the owning account off the first zone the token can see.
func (c *Client) zoneAccountID(ctx context.Context) (string, error) {
	var out apiResponse
	if err := c.do(ctx, http.MethodGet, "zones?per_page=1", nil, &out); err != nil {
		return "", err
	}
	var zones []struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	if err := json.Unmarshal(out.Result, &zones); err != nil {
		return "", fmt.Errorf("decode zones: %w", err)
	}
	if len(zones) == 0 {
		return "", nil
	}
	return zones[0].Account.ID, nil
}

// tunnelEnvelope is the account-scoped tunnel object returned by the v4 API.
type tunnelEnvelope struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AccountTag string `json:"account_tag"`
	Status     string `json:"status"`
	Token      string `json:"token"`
}

func toTunnel(e tunnelEnvelope) *dns.Tunnel {
	return &dns.Tunnel{ID: e.ID, Name: e.Name, AccountID: e.AccountTag, Status: e.Status, Token: e.Token}
}

// CreateTunnel implements dns.TunnelProvider, creating a remotely-managed tunnel and returning it with its token.
func (c *Client) CreateTunnel(ctx context.Context, name string) (*dns.Tunnel, error) {
	acct, err := c.accountID(ctx)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"name": name, "config_src": "cloudflare"}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel", acct)
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	var e tunnelEnvelope
	if err := json.Unmarshal(out.Result, &e); err != nil {
		return nil, fmt.Errorf("decode tunnel: %w", err)
	}
	return toTunnel(e), nil
}

// ListTunnels implements dns.TunnelProvider.
func (c *Client) ListTunnels(ctx context.Context) ([]dns.Tunnel, error) {
	acct, err := c.accountID(ctx)
	if err != nil {
		return nil, err
	}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel?per_page=50", acct)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var list []tunnelEnvelope
	if err := json.Unmarshal(out.Result, &list); err != nil {
		return nil, fmt.Errorf("decode tunnels: %w", err)
	}
	outTunnels := make([]dns.Tunnel, 0, len(list))
	for _, e := range list {
		outTunnels = append(outTunnels, *toTunnel(e))
	}
	return outTunnels, nil
}

// GetTunnel implements dns.TunnelProvider.
func (c *Client) GetTunnel(ctx context.Context, tunnelID string) (*dns.Tunnel, error) {
	acct, err := c.accountID(ctx)
	if err != nil {
		return nil, err
	}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel/%s", acct, tunnelID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var e tunnelEnvelope
	if err := json.Unmarshal(out.Result, &e); err != nil {
		return nil, fmt.Errorf("decode tunnel: %w", err)
	}
	return toTunnel(e), nil
}

// DeleteTunnel implements dns.TunnelProvider; a 404 for an already-absent tunnel is a no-op success (idempotent).
func (c *Client) DeleteTunnel(ctx context.Context, tunnelID string) error {
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel/%s", acct, tunnelID)
	if err := c.do(ctx, http.MethodDelete, path, nil, &out); err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// ingressRule is one ingress-config entry; a rule with no Hostname is the trailing catch-all Cloudflare requires.
type ingressRule struct {
	Hostname      string         `json:"hostname,omitempty"`
	Service       string         `json:"service"`
	OriginRequest map[string]any `json:"originRequest,omitempty"`
}

// tunnelIngress fetches the rules minus the trailing catch-all, so routing never clobbers hostnames routed earlier.
func (c *Client) tunnelIngress(ctx context.Context, acct, tunnelID string) ([]ingressRule, error) {
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel/%s/configurations", acct, tunnelID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var cfg struct {
		Config struct {
			Ingress []ingressRule `json:"ingress"`
		} `json:"config"`
	}
	if err := json.Unmarshal(out.Result, &cfg); err != nil {
		return nil, fmt.Errorf("decode tunnel configuration: %w", err)
	}
	rules := make([]ingressRule, 0, len(cfg.Config.Ingress))
	for _, r := range cfg.Config.Ingress {
		if r.Hostname == "" {
			continue // drop the catch-all; callers re-append it
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// putTunnelIngress writes the ingress rules plus the trailing catch-all.
func (c *Client) putTunnelIngress(ctx context.Context, acct, tunnelID string, rules []ingressRule) error {
	body := map[string]any{"config": map[string]any{
		"ingress": append(append([]ingressRule{}, rules...), ingressRule{Service: "http_status:404"}),
	}}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel/%s/configurations", acct, tunnelID)
	return c.do(ctx, http.MethodPut, path, body, &out)
}

// RouteTunnelHostname upserts the rule for this hostname, writing the list back so a second doesn't clobber the first.
func (c *Client) RouteTunnelHostname(ctx context.Context, tunnelID, hostname, service string) error {
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	existing, err := c.tunnelIngress(ctx, acct, tunnelID)
	if err != nil {
		return err
	}
	rule := ingressRule{Hostname: hostname, Service: service, OriginRequest: map[string]any{}}
	rules := make([]ingressRule, 0, len(existing)+1)
	replaced := false
	for _, r := range existing {
		if r.Hostname == hostname {
			rules = append(rules, rule)
			replaced = true
			continue
		}
		rules = append(rules, r)
	}
	if !replaced {
		rules = append(rules, rule)
	}
	return c.putTunnelIngress(ctx, acct, tunnelID, rules)
}

// ListTunnelHostnames implements dns.TunnelProvider: the routed hostnames, catch-all already dropped by tunnelIngress.
func (c *Client) ListTunnelHostnames(ctx context.Context, tunnelID string) ([]dns.TunnelRoute, error) {
	acct, err := c.accountID(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := c.tunnelIngress(ctx, acct, tunnelID)
	if err != nil {
		return nil, err
	}
	routes := make([]dns.TunnelRoute, 0, len(rules))
	for _, r := range rules {
		routes = append(routes, dns.TunnelRoute{Hostname: r.Hostname, Service: r.Service})
	}
	return routes, nil
}

// RemoveTunnelHostname drops the rule for this hostname and writes the rest back; other routed hostnames survive.
func (c *Client) RemoveTunnelHostname(ctx context.Context, tunnelID, hostname string) error {
	acct, err := c.accountID(ctx)
	if err != nil {
		return err
	}
	existing, err := c.tunnelIngress(ctx, acct, tunnelID)
	if err != nil {
		return err
	}
	rules := make([]ingressRule, 0, len(existing))
	for _, r := range existing {
		if r.Hostname == hostname {
			continue
		}
		rules = append(rules, r)
	}
	return c.putTunnelIngress(ctx, acct, tunnelID, rules)
}

// RotateTunnelCredentials issues a fresh secret and token, force-disconnecting connectors still on the old one.
func (c *Client) RotateTunnelCredentials(ctx context.Context, tunnelID string) (string, error) {
	acct, err := c.accountID(ctx)
	if err != nil {
		return "", err
	}
	secret := newTunnelSecret()
	body := map[string]any{"tunnel_secret": secret}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel/%s", acct, tunnelID)
	if err := c.do(ctx, http.MethodPatch, path, body, &out); err != nil {
		return "", err
	}
	var e tunnelEnvelope
	if err := json.Unmarshal(out.Result, &e); err != nil {
		return "", fmt.Errorf("decode rotated tunnel: %w", err)
	}
	if e.Token == "" {
		return "", apperrs.Retryable(fmt.Errorf("rotate tunnel %s: empty token in response", tunnelID))
	}
	// Force-disconnect so the old token cannot establish new connections.
	var conns apiResponse
	connPath := fmt.Sprintf("accounts/%s/cfd_tunnel/%s/connections", acct, tunnelID)
	if err := c.do(ctx, http.MethodDelete, connPath, nil, &conns); err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		// Rotation itself succeeded; a failed disconnect is surfaced as a
		// non-fatal error so the caller can note the caveat.
		return e.Token, fmt.Errorf("%w: tunnel token rotated but connectors not disconnected: %v", apperrs.ErrRetryable, err)
	}
	return e.Token, nil
}

// TunnelToken implements dns.TunnelProvider: it returns the current token.
func (c *Client) TunnelToken(ctx context.Context, tunnelID string) (string, error) {
	acct, err := c.accountID(ctx)
	if err != nil {
		return "", err
	}
	var out apiResponse
	path := fmt.Sprintf("accounts/%s/cfd_tunnel/%s/token", acct, tunnelID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return "", err
	}
	var token string
	if err := json.Unmarshal(out.Result, &token); err != nil {
		return "", fmt.Errorf("decode tunnel token: %w", err)
	}
	if token == "" {
		return "", apperrs.Retryable(fmt.Errorf("tunnel %s: empty token", tunnelID))
	}
	return token, nil
}

// newTunnelSecret generates a base64 32-byte tunnel secret (minimum size for
// the Cloudflare API is 32 bytes).
func newTunnelSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

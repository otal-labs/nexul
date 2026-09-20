// Package cloudflare implements dns.DNSProvider against Cloudflare's v4 API.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// Client is the Cloudflare DNSProvider implementation.
type Client struct {
	base    *url.URL
	token   string
	httpc   *http.Client
	resolve dnsResolver
	tunnel  tunnelState
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at a different API root (tests, GHE-style
// mirrors). The path must end in /client/v4.
func WithBaseURL(u *url.URL) Option {
	return func(c *Client) {
		if u != nil {
			base := *u
			if !strings.HasSuffix(base.Path, "/") {
				base.Path += "/"
			}
			c.base = &base
		}
	}
}

// WithHTTPClient overrides the HTTP client (tests, custom timeouts).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpc = hc
		}
	}
}

// WithResolver overrides the DNS resolver used by CheckPropagation (tests).
func WithResolver(r dnsResolver) Option {
	return func(c *Client) {
		if r != nil {
			c.resolve = r
		}
	}
}

// WithAccountID pins the account for tunnel operations; absent, resolves the first account the token can read.
func WithAccountID(accountID string) Option {
	return func(c *Client) {
		if accountID != "" {
			c.tunnel.accountID = accountID
		}
	}
}

// New builds a Cloudflare provider authenticated with an API token (OAuth
// access tokens and API tokens both work as Bearer credentials).
func New(token string, opts ...Option) *Client {
	c := &Client{
		base:    mustParse(defaultBaseURL),
		token:   token,
		httpc:   &http.Client{Timeout: 20 * time.Second},
		resolve: net.DefaultResolver,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// dnsResolver is the slice of net.Resolver CheckPropagation needs.
type dnsResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// Verify implements dns.DNSProvider: it checks the token is valid and active.
func (c *Client) Verify(ctx context.Context) error {
	var out apiResponse
	if err := c.do(ctx, http.MethodGet, "user/tokens/verify", nil, &out); err != nil {
		return err
	}
	return nil
}

// ListZones implements dns.DNSProvider.
func (c *Client) ListZones(ctx context.Context) ([]dns.Zone, error) {
	zones, err := c.listZones(ctx, 1)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (c *Client) listZones(ctx context.Context, page int) ([]dns.Zone, error) {
	var out apiResponse
	path := fmt.Sprintf("zones?per_page=50&page=%d", page)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var zones []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Result, &zones); err != nil {
		return nil, fmt.Errorf("decode zones: %w", err)
	}
	var outZones []dns.Zone
	for _, z := range zones {
		outZones = append(outZones, dns.Zone{ID: z.ID, Name: z.Name, Status: z.Status})
	}
	if len(outZones) == 50 {
		next, err := c.listZones(ctx, page+1)
		if err != nil {
			return nil, err
		}
		outZones = append(outZones, next...)
	}
	return outZones, nil
}

// ListRecords implements dns.DNSProvider.
func (c *Client) ListRecords(ctx context.Context, zoneID string) ([]dns.Record, error) {
	records, err := c.listRecords(ctx, zoneID, 1)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (c *Client) listRecords(ctx context.Context, zoneID string, page int) ([]dns.Record, error) {
	var out apiResponse
	path := fmt.Sprintf("zones/%s/dns_records?per_page=100&page=%d", url.PathEscape(zoneID), page)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var records []struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Name    string `json:"name"`
		TTL     int    `json:"ttl"`
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
	}
	if err := json.Unmarshal(out.Result, &records); err != nil {
		return nil, fmt.Errorf("decode records: %w", err)
	}
	var outRecords []dns.Record
	for _, r := range records {
		outRecords = append(outRecords, dns.Record{
			ID: r.ID, ZoneID: zoneID, Type: dns.RecordType(r.Type), Name: r.Name, Content: r.Content, TTL: r.TTL, Proxied: r.Proxied,
		})
	}
	if len(outRecords) == 100 {
		next, err := c.listRecords(ctx, zoneID, page+1)
		if err != nil {
			return nil, err
		}
		outRecords = append(outRecords, next...)
	}
	return outRecords, nil
}

// CreateRecord implements dns.DNSProvider.
func (c *Client) CreateRecord(ctx context.Context, zoneID string, in dns.RecordInput) (*dns.Record, error) {
	body := recordBody{Type: string(in.Type), Name: in.Name, Content: in.Content, TTL: in.TTL, Proxied: in.Proxied}
	var out apiResponse
	path := fmt.Sprintf("zones/%s/dns_records", url.PathEscape(zoneID))
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return decodeRecord(zoneID, out.Result)
}

// UpdateRecord implements dns.DNSProvider.
func (c *Client) UpdateRecord(ctx context.Context, zoneID, recordID string, in dns.RecordInput) (*dns.Record, error) {
	body := recordBody{Type: string(in.Type), Name: in.Name, Content: in.Content, TTL: in.TTL, Proxied: in.Proxied}
	var out apiResponse
	path := fmt.Sprintf("zones/%s/dns_records/%s", url.PathEscape(zoneID), url.PathEscape(recordID))
	if err := c.do(ctx, http.MethodPatch, path, body, &out); err != nil {
		return nil, err
	}
	return decodeRecord(zoneID, out.Result)
}

// DeleteRecord implements dns.DNSProvider. Cloudflare returns 404 for an
// already-absent record; that is a no-op success (idempotent delete).
func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	var out apiResponse
	path := fmt.Sprintf("zones/%s/dns_records/%s", url.PathEscape(zoneID), url.PathEscape(recordID))
	if err := c.do(ctx, http.MethodDelete, path, nil, &out); err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// CheckPropagation resolves the record name through public DNS, retryable until the expected content shows up.
func (c *Client) CheckPropagation(ctx context.Context, zoneID string, rec dns.Record) error {
	name := ensureTrailingDot(rec.Name)
	switch rec.Type {
	case dns.RecordA, dns.RecordAAAA:
		addrs, err := c.resolve.LookupHost(ctx, name)
		if err != nil {
			return apperrs.Retryable(fmt.Errorf("resolve %s: %w", rec.Name, err))
		}
		for _, a := range addrs {
			if strings.EqualFold(a, rec.Content) {
				return nil
			}
		}
		return apperrs.Retryable(fmt.Errorf("%s not propagated: resolved %v, want %s", rec.Name, addrs, rec.Content))
	case dns.RecordCNAME:
		target, err := c.resolve.LookupCNAME(ctx, name)
		if err != nil {
			return apperrs.Retryable(fmt.Errorf("resolve %s: %w", rec.Name, err))
		}
		if strings.EqualFold(strings.TrimSuffix(target, "."), strings.TrimSuffix(rec.Content, ".")) {
			return nil
		}
		return apperrs.Retryable(fmt.Errorf("%s not propagated: resolved %s, want %s", rec.Name, target, rec.Content))
	case dns.RecordTXT:
		values, err := c.resolve.LookupTXT(ctx, name)
		if err != nil {
			return apperrs.Retryable(fmt.Errorf("resolve %s: %w", rec.Name, err))
		}
		for _, v := range values {
			if v == rec.Content {
				return nil
			}
		}
		return apperrs.Retryable(fmt.Errorf("%s not propagated: %v", rec.Name, values))
	default:
		return fmt.Errorf("%w: unsupported record type %q", apperrs.ErrInvalid, rec.Type)
	}
}

func decodeRecord(zoneID string, raw json.RawMessage) (*dns.Record, error) {
	var r struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}
	return &dns.Record{
		ID: r.ID, ZoneID: zoneID, Type: dns.RecordType(r.Type), Name: r.Name, Content: r.Content, TTL: r.TTL,
	}, nil
}

type recordBody struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// apiResponse is the Cloudflare v4 envelope: success plus either an errors
// array or a result payload.
type apiResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// do performs an authenticated request and maps Cloudflare errors to the platform sentinels (unauthorized/not found/retryable).
func (c *Client) do(ctx context.Context, method, path string, body any, out *apiResponse) (err error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	u := *c.base
	pathPart, queryPart, _ := strings.Cut(strings.TrimPrefix(path, "/"), "?")
	u.Path = strings.TrimSuffix(c.base.Path, "/") + "/" + pathPart
	u.RawQuery = queryPart
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return apperrs.Retryable(fmt.Errorf("cloudflare %s %s: %w", method, path, err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return apperrs.Retryable(fmt.Errorf("read cloudflare response: %w", err))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode cloudflare response: %w", err)
	}
	if !out.Success {
		return fmt.Errorf("%w (%s %s)", mapAPIError(resp.StatusCode, out.Errors), method, strings.SplitN(path, "?", 2)[0])
	}
	if resp.StatusCode >= 300 {
		return apperrs.Retryable(fmt.Errorf("cloudflare %s %s: unexpected status %d", method, path, resp.StatusCode))
	}
	return nil
}

func mapAPIError(status int, errs []struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}) error {
	msg := "cloudflare api error"
	if len(errs) > 0 {
		msg = fmt.Sprintf("cloudflare: %s", errs[0].Message)
	}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		// Not ErrUnauthorized: that maps to HTTP 401, which the web client treats as an expired session and logs out.
		return fmt.Errorf("%w: Cloudflare rejected the connected token — %s", apperrs.ErrInvalid, msg)
	case status == http.StatusNotFound:
		return fmt.Errorf("%w: %s", apperrs.ErrNotFound, msg)
	case status == http.StatusBadRequest:
		return fmt.Errorf("%w: %s", apperrs.ErrInvalid, msg)
	case status == http.StatusConflict:
		return fmt.Errorf("%w: %s", apperrs.ErrConflict, msg)
	case status >= 500:
		return apperrs.Retryable(fmt.Errorf("%s", msg))
	default:
		return apperrs.Retryable(fmt.Errorf("%s", msg))
	}
}

func ensureTrailingDot(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

func mustParse(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

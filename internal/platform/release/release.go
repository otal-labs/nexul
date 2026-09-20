// Package release is a small GitHub Releases client for the Nexul repo, shared by the runner download
// proxy and the /api/version handler so both hit the same 5-minute cache.
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// repo is the GitHub repository releases are published to; calls carry a token when one is available.
const repo = "otal-labs/nexul"

// latestTTL bounds how long a channel's resolved latest release is served before re-checking GitHub.
const latestTTL = 5 * time.Minute

// Config wires the client to GitHub.
type Config struct {
	// TokenSource resolves the GitHub connector's token; nil or erroring means unauthenticated.
	TokenSource func(ctx context.Context) (string, error)
	// APIBase overrides the GitHub API root; defaults to https://api.github.com.
	APIBase string
	// HTTP performs the GitHub calls; defaults to a 2-minute timeout since binaries can be tens of MB.
	HTTP *http.Client
}

// Release is one GitHub release.
type Release struct {
	Tag        string
	URL        string
	Prerelease bool
	Assets     []Asset
}

// Asset is one file attached to a Release.
type Asset struct {
	Name string
	URL  string
	Size int64
}

// Client looks up releases and streams their assets, caching the resolved "latest" release per channel.
type Client struct {
	cfg Config
	// now is the clock Latest checks the cache against; overridden only in tests to exercise the 5-minute expiry
	// without sleeping.
	now func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	release *Release
	at      time.Time
}

// New wires a Client with cfg's defaults filled in.
func New(cfg Config) *Client {
	if cfg.APIBase == "" {
		cfg.APIBase = "https://api.github.com"
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 2 * time.Minute}
	}
	return &Client{cfg: cfg, now: time.Now, cache: make(map[string]cacheEntry)}
}

// Latest resolves channel's newest release: "stable" and "dev" both GET releases/latest; "beta" lists releases
// and picks the first (newest) non-draft prerelease tag (one with a dash after the semver). Cached per channel for 5 minutes;
// errors are never cached.
func (c *Client) Latest(ctx context.Context, channel string) (*Release, error) {
	if rel, ok := c.cacheGet(channel); ok {
		return rel, nil
	}
	rel, err := c.fetchLatest(ctx, channel)
	if err != nil {
		return nil, err
	}
	c.cacheSet(channel, rel)
	return rel, nil
}

func (c *Client) fetchLatest(ctx context.Context, channel string) (*Release, error) {
	if channel == "beta" {
		return c.latestBeta(ctx)
	}
	return c.fetchRelease(ctx, "/repos/"+repo+"/releases/latest")
}

// ByTag fetches the release tagged tag.
func (c *Client) ByTag(ctx context.Context, tag string) (*Release, error) {
	return c.fetchRelease(ctx, "/repos/"+repo+"/releases/tags/"+url.PathEscape(tag))
}

// Download streams the asset named name from rel; callers must close the returned reader.
func (c *Client) Download(ctx context.Context, rel *Release, name string) (io.ReadCloser, int64, error) {
	asset := findAsset(rel, name)
	if asset == nil {
		return nil, 0, fmt.Errorf("release asset %q: %w", name, apperrs.ErrNotFound)
	}
	body, err := c.fetchAssetBody(ctx, asset.URL)
	if err != nil {
		return nil, 0, err
	}
	return body, asset.Size, nil
}

// Checksums downloads rel's checksums.txt and parses its "<sha256>  <name>" lines into a map keyed by name.
// Returns an empty map, no error, when the release has no checksums.txt asset.
func (c *Client) Checksums(ctx context.Context, rel *Release) (map[string]string, error) {
	asset := findAsset(rel, "checksums.txt")
	if asset == nil {
		return map[string]string{}, nil
	}
	body, err := c.fetchAssetBody(ctx, asset.URL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := body.Close(); cerr != nil {
			slog.Default().Debug("close checksums.txt body", "error", cerr)
		}
	}()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read checksums.txt: %w", err)
	}
	return parseChecksums(string(data)), nil
}

func parseChecksums(data string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		out[fields[len(fields)-1]] = fields[0]
	}
	return out
}

func findAsset(rel *Release, name string) *Asset {
	for i := range rel.Assets {
		if rel.Assets[i].Name == name {
			return &rel.Assets[i]
		}
	}
	return nil
}

func (c *Client) cacheGet(channel string) (*Release, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.cache[channel]
	if !ok || c.now().Sub(entry.at) >= latestTTL {
		return nil, false
	}
	return entry.release, true
}

func (c *Client) cacheSet(channel string, rel *Release) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[channel] = cacheEntry{release: rel, at: c.now()}
}

// ghRelease is the subset of GitHub's release API response this package needs.
type ghRelease struct {
	TagName    string    `json:"tag_name"`
	HTMLURL    string    `json:"html_url"`
	Prerelease bool      `json:"prerelease"`
	Draft      bool      `json:"draft"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

func (r ghRelease) toRelease() *Release {
	assets := make([]Asset, len(r.Assets))
	for i, a := range r.Assets {
		assets[i] = Asset(a)
	}
	return &Release{Tag: r.TagName, URL: r.HTMLURL, Prerelease: r.Prerelease, Assets: assets}
}

// latestBeta lists the newest 30 releases and picks the first non-draft prerelease tag (v0.2.0-beta-003, not v0.2.0).
func (c *Client) latestBeta(ctx context.Context) (_ *Release, err error) {
	req, err := c.newRequest(ctx, c.cfg.APIBase+"/repos/"+repo+"/releases?per_page=30")
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req, "list releases")
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var releases []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}
	for _, r := range releases {
		if r.Draft {
			continue
		}
		if strings.Contains(r.TagName, "-") {
			return r.toRelease(), nil
		}
	}
	return nil, fmt.Errorf("no beta release found: %w", apperrs.ErrNotFound)
}

func (c *Client) fetchRelease(ctx context.Context, path string) (_ *Release, err error) {
	req, err := c.newRequest(ctx, c.cfg.APIBase+path)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req, "fetch release")
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return rel.toRelease(), nil
}

// fetchAssetBody GETs an asset's download URL; the caller owns the body. Go rightly drops Authorization on a 302.
func (c *Client) fetchAssetBody(ctx context.Context, assetURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build asset request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if token := c.token(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release asset: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		statusErr := statusError(resp.StatusCode, "fetch release asset")
		return nil, errors.Join(statusErr, resp.Body.Close())
	}
	return resp.Body, nil
}

func (c *Client) newRequest(ctx context.Context, rawURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := c.token(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func (c *Client) do(req *http.Request, action string) (*http.Response, error) {
	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	if resp.StatusCode != http.StatusOK {
		statusErr := statusError(resp.StatusCode, action)
		return nil, errors.Join(statusErr, resp.Body.Close())
	}
	return resp, nil
}

// token falls back to unauthenticated when the connector source fails (GitHub not connected yet).
func (c *Client) token(ctx context.Context) string {
	if c.cfg.TokenSource == nil {
		return ""
	}
	token, err := c.cfg.TokenSource(ctx)
	if err != nil {
		slog.Default().Debug("release token source unavailable; calling GitHub unauthenticated", "error", err)
		return ""
	}
	return token
}

// statusError names the status; a 404 reports that the release or asset is missing.
func statusError(status int, action string) error {
	if status == http.StatusNotFound {
		return fmt.Errorf("%s: status %d, release or asset not found: %w", action, status, apperrs.ErrNotFound)
	}
	return fmt.Errorf("%s: status %d: %w", action, status, apperrs.ErrRetryable)
}

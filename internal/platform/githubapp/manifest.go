package githubapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// ManifestApp is the secret-bearing conversion result, kept server-side.
type ManifestApp struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Slug         string `json:"slug"`
	PrivateKey   string `json:"pem"`
}

// ManifestClient exchanges registration codes only with GitHub's API, without forwarding setup credentials.
type ManifestClient struct {
	Client *http.Client
}

func (c ManifestClient) Convert(ctx context.Context, code string) (ManifestApp, error) {
	var out ManifestApp
	hc := c.Client
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	client := *hc
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, DefaultAPIBase+"/app-manifests/"+url.PathEscape(code)+"/conversions", nil)
	if err != nil {
		return out, fmt.Errorf("%w: cannot build GitHub registration request", apperrs.ErrInvalid)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return out, fmt.Errorf("%w: GitHub registration is unavailable", apperrs.ErrRetryable)
	}
	defer func() { _ = resp.Body.Close() }() // A read-only response has no pending write to lose on close.
	if resp.StatusCode != http.StatusCreated {
		return out, fmt.Errorf("%w: GitHub could not complete registration, start again (status %d)", apperrs.ErrInvalid, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 128<<10)).Decode(&out); err != nil {
		return ManifestApp{}, fmt.Errorf("%w: GitHub returned an unreadable registration", apperrs.ErrInvalid)
	}
	if out.ClientID == "" || out.ClientSecret == "" || out.Slug == "" {
		return ManifestApp{}, fmt.Errorf("%w: GitHub returned incomplete App credentials", apperrs.ErrInvalid)
	}
	if _, err := ParsePrivateKey(out.PrivateKey); err != nil {
		return ManifestApp{}, fmt.Errorf("%w: GitHub returned an invalid private key", apperrs.ErrInvalid)
	}
	return out, nil
}

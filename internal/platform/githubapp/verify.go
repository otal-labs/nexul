// Package githubapp checks a GitHub App registration live, so a typo never gets stored as this instance's login app.
package githubapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// DefaultAPIBase is api.github.com; GHE callers pass their own /api/v3 root.
const DefaultAPIBase = "https://api.github.com"

// Verify confirms slug resolves to an app whose client_id is clientID, and that GitHub accepts clientSecret for it.
func Verify(ctx context.Context, hc *http.Client, apiBase, clientID, clientSecret, slug string) error {
	if apiBase == "" {
		apiBase = DefaultAPIBase
	}
	if err := verifySlug(ctx, hc, apiBase, clientID, slug); err != nil {
		return err
	}
	return verifySecret(ctx, hc, apiBase, clientID, clientSecret)
}

// VerifyCheck runs one named half of Verify ("slug" or "secret") so a setup form can tick each row on its own.
func VerifyCheck(ctx context.Context, hc *http.Client, apiBase, check, clientID, clientSecret, slug string) error {
	if apiBase == "" {
		apiBase = DefaultAPIBase
	}
	switch check {
	case "slug":
		return verifySlug(ctx, hc, apiBase, clientID, slug)
	case "secret":
		return verifySecret(ctx, hc, apiBase, clientID, clientSecret)
	default:
		return fmt.Errorf("%w: unknown check %q", apperrs.ErrInvalid, check)
	}
}

// verifySlug uses the public /apps/{slug} endpoint, which also reports the app's client_id.
func verifySlug(ctx context.Context, hc *http.Client, apiBase, clientID, slug string) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/apps/"+url.PathEscape(slug), nil)
	if err != nil {
		return fmt.Errorf("build apps request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("reach github: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: no GitHub App with slug %q — it is the name in your app's URL (github.com/apps/<slug>)", apperrs.ErrInvalid, slug)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github apps lookup: status %d", resp.StatusCode)
	}
	var app struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return fmt.Errorf("decode github app: %w", err)
	}
	if app.ClientID != "" && app.ClientID != clientID {
		return fmt.Errorf("%w: GitHub App %q has client ID %s, not %s — check you copied the ID from the same app", apperrs.ErrInvalid, slug, app.ClientID, clientID)
	}
	return nil
}

// verifySecret checks a throwaway token via /applications/{client_id}/token: 401 means bad client credentials, 404 means the credentials were accepted and the token merely unknown.
func verifySecret(ctx context.Context, hc *http.Client, apiBase, clientID, clientSecret string) (err error) {
	body, _ := json.Marshal(map[string]string{"access_token": "nexul-verify"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/applications/"+url.PathEscape(clientID)+"/token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build token check request: %w", err)
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("reach github: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: GitHub rejected the client secret for client ID %s", apperrs.ErrInvalid, clientID)
	case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusUnprocessableEntity, resp.StatusCode < 300:
		return nil
	default:
		return fmt.Errorf("github token check: status %d", resp.StatusCode)
	}
}

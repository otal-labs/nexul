package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// LookupMembers works only through GitHub, since only it exposes a user-search API (ADR 0040).
func (s *Service) LookupMembers(ctx context.Context, userID, q string) (matches []LoginMatch, err error) {
	if err := s.requireCanCreateWorkspace(ctx, userID); err != nil {
		return nil, err
	}
	q = strings.TrimSpace(q)
	if len(q) < 2 || strings.Contains(q, "@") {
		return []LoginMatch{}, nil
	}
	req, err := s.newSearchRequest(ctx, q)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search github users: %w", apperrs.Retryable(err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// ponytail: a rate-limit hit (403/429) just empties the typeahead; add a cache if that bites in practice.
		return []LoginMatch{}, nil
	}
	var body struct {
		Items []struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode github search response: %w", apperrs.Retryable(err))
	}
	matches = make([]LoginMatch, 0, len(body.Items))
	for _, item := range body.Items {
		matches = append(matches, LoginMatch{Login: item.Login, AvatarURL: item.AvatarURL})
	}
	return matches, nil
}

// newSearchRequest builds the user-search request, sending stored OAuth credentials as Basic auth when configured.
func (s *Service) newSearchRequest(ctx context.Context, q string) (*http.Request, error) {
	params := url.Values{
		"q":        {q + " in:login"},
		"per_page": {"5"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "nexul")
	if s.cfg.Settings != nil {
		if st, err := s.cfg.Settings.Get(ctx); err == nil {
			if id, secret := st.OAuthCredentials(ProviderGitHub); id != "" && secret != "" {
				req.SetBasicAuth(id, secret)
			}
		}
	}
	return req, nil
}

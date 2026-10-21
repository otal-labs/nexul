package t3rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
)

const (
	sessionPath = "/api/auth/session"

	// sourceControlWrite is the scope T3's projectClone.start requires.
	sourceControlWrite = "source-control:write"
)

// CanClone reports whether T3 granted the bearer session the scope a clone needs; a T3 that does not say counts as yes.
func CanClone(ctx context.Context, client *http.Client, serverURL, bearerToken string) (bool, error) {
	scopes, err := sessionScopes(ctx, client, serverURL, bearerToken)
	if err != nil {
		return false, err
	}
	return scopes == nil || slices.Contains(scopes, sourceControlWrite), nil
}

// sessionScopes reads the scopes of the bearer session: nil when T3 names none, empty when it names an empty grant.
func sessionScopes(ctx context.Context, client *http.Client, serverURL, bearerToken string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(serverURL, "/")+sessionPath, nil)
	if err != nil {
		return nil, fmt.Errorf("build session request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	body, status, err := doStatus(client, req)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("%w (%d): %s", harness.ErrSessionRejected, status, strings.TrimSpace(string(body)))
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return nil, nil
	default:
		return nil, fmt.Errorf("T3 session responded %d: %s", status, Snippet(body))
	}
	var out struct {
		Permissions *[]string `json:"permissions"`
		Scopes      *[]string `json:"scopes"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode T3 session: %w", err)
	}
	for _, granted := range []*[]string{out.Permissions, out.Scopes} {
		if granted != nil {
			return append([]string{}, *granted...), nil
		}
	}
	return nil, nil
}

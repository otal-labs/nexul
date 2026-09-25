package mcp

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// originGuard answers 403 to a request whose Origin is not the configured instance URL's, the specification's
// DNS-rebinding rule; a request without Origin (every non-browser client) passes to the bearer check.
func originGuard(instanceURL func(context.Context) (string, error), next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && !allowedOrigin(r.Context(), instanceURL, origin) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOrigin(ctx context.Context, instanceURL func(context.Context) (string, error), origin string) bool {
	if instanceURL == nil {
		return false
	}
	raw, err := instanceURL(ctx)
	if err != nil {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return strings.EqualFold(origin, u.Scheme+"://"+u.Host)
}

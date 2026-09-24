package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// AccessCredentials returns the Access service-token pair for host; ok is false for every host that is not a computer tunnel.
type AccessCredentials func(ctx context.Context, host string) (clientID, clientSecret string, ok bool, err error)

// AccessTransport sends the Access headers only to computer tunnel hostnames, WebSocket upgrades included, never to URL-paired machines.
type AccessTransport struct {
	// Base sends the request; nil means http.DefaultTransport.
	Base        http.RoundTripper
	Credentials AccessCredentials
}

// RoundTrip implements http.RoundTripper; the caller's request is never modified.
func (t *AccessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.ToLower(req.URL.Hostname())
	clientID, clientSecret, ok, err := t.Credentials(req.Context(), host)
	if err != nil {
		if req.Body != nil {
			_ = req.Body.Close() // RoundTrip must close the body even on error; the credential error is the one to report
		}
		return nil, fmt.Errorf("resolve access credentials for %s: %w", host, err)
	}
	if !ok {
		return t.base().RoundTrip(req)
	}
	out := req.Clone(req.Context())
	out.Header.Set("CF-Access-Client-Id", clientID)
	out.Header.Set("CF-Access-Client-Secret", clientSecret)
	return t.base().RoundTrip(out)
}

func (t *AccessTransport) base() http.RoundTripper {
	if t.Base == nil {
		return http.DefaultTransport
	}
	return t.Base
}

// Package freshdns resolves names through a public resolver before the host's, so a record created seconds ago is
// not hidden behind the negative answer the host's resolver cached while it did not exist yet (Cloudflare zones
// cache "no such name" for 30 minutes).
package freshdns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// publicServers answer from the authoritative zone without anyone's stale negative cache.
var publicServers = []string{"1.1.1.1:53", "1.0.0.1:53"}

// Resolver looks names up through the public servers first, then through the host's own resolver, which is what
// answers for names only a private DNS knows.
type Resolver struct {
	Public, System interface {
		LookupHost(ctx context.Context, host string) ([]string, error)
	}
}

// New returns a Resolver over publicServers and the host's resolver.
func New() *Resolver {
	public := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			var errs []error
			for _, server := range publicServers {
				conn, err := d.DialContext(ctx, network, server)
				if err == nil {
					return conn, nil
				}
				errs = append(errs, err)
			}
			return nil, errors.Join(errs...)
		},
	}
	return &Resolver{Public: public, System: net.DefaultResolver}
}

// LookupHost returns host's addresses from the first resolver that has any.
func (r *Resolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	addrs, err := r.Public.LookupHost(ctx, host)
	if err == nil && len(addrs) > 0 {
		return addrs, nil
	}
	return r.System.LookupHost(ctx, host)
}

// DialContext resolves addr's host with LookupHost and connects to the first address that answers.
func (r *Resolver) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split %s: %w", addr, err)
	}
	if net.ParseIP(host) != nil {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	}
	addrs, err := r.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}
	var d net.Dialer
	var errs []error
	for _, ip := range addrs {
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			return conn, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}

// Client returns an HTTP client whose connections resolve through r, for checking a hostname that was just created.
func (r *Resolver) Client(timeout time.Duration) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = r.DialContext
	return &http.Client{Timeout: timeout, Transport: tr}
}

package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/otal-labs/nexul/internal/dns/cloudflare"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/runner"
)

// computerHostSuffix names a computer reached through its personal runner (ADR 0146); .invalid never resolves.
const computerHostSuffix = ".nexul-computer.invalid"

// computerDialer opens a connection to a computer's T3 Code through its runner.
type computerDialer interface {
	DialComputer(ctx context.Context, computerID string) (net.Conn, error)
}

// runnerComputers reaches the runner handler, which exists only once the workers start after the harness client.
type runnerComputers struct {
	handler atomic.Pointer[runner.Handler]
}

func (r *runnerComputers) DialComputer(ctx context.Context, computerID string) (net.Conn, error) {
	h := r.handler.Load()
	if h == nil {
		return nil, apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	return h.DialComputer(ctx, computerID)
}

// harnessHTTPClient is both T3 clients' client: a computer's address dials through its runner, the rest as before.
func harnessHTTPClient(computers computerDialer, access cloudflare.AccessCredentials) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	dial, proxy := base.DialContext, base.Proxy
	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if id, ok := strings.CutSuffix(host, computerHostSuffix); ok && err == nil {
			return computers.DialComputer(ctx, id)
		}
		return dial(ctx, network, addr)
	}
	base.Proxy = func(req *http.Request) (*url.URL, error) {
		if isComputerHost(req) {
			return nil, nil
		}
		return proxy(req)
	}
	return &http.Client{Transport: &cloudflare.AccessTransport{Base: asLoopback{base}, Credentials: access}}
}

// asLoopback sends a computer's request with the Host a local client sends T3 Code.
type asLoopback struct{ next http.RoundTripper }

func (t asLoopback) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isComputerHost(req) {
		return t.next.RoundTrip(req)
	}
	out := req.Clone(req.Context())
	out.Host = "127.0.0.1"
	return t.next.RoundTrip(out)
}

func isComputerHost(req *http.Request) bool {
	return strings.HasSuffix(req.URL.Hostname(), computerHostSuffix)
}

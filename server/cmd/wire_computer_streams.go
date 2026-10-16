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
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/runner"
)

// computerDialer opens a connection to a computer's T3 Code through its runner.
type computerDialer interface {
	DialComputer(ctx context.Context, computerID string) (net.Conn, error)
}

// runnerComputers reaches the runner handler and service, which exist only once the workers start after the harness
// client and the pairing domain.
type runnerComputers struct {
	handler atomic.Pointer[runner.Handler]
	svc     atomic.Pointer[runner.Service]
}

func (r *runnerComputers) service() (*runner.Service, error) {
	s := r.svc.Load()
	if s == nil {
		return nil, apperrs.Retryable(errors.New("the runner service has not started"))
	}
	return s, nil
}

// EnrollComputer is pairing's seam for minting a computer's personal runner code.
func (r *runnerComputers) EnrollComputer(ctx context.Context, userID, computerID string) (pairing.Enrollment, error) {
	s, err := r.service()
	if err != nil {
		return pairing.Enrollment{}, err
	}
	e, err := s.CreatePersonalEnrollment(ctx, userID, computerID)
	if err != nil {
		return pairing.Enrollment{}, err
	}
	return pairing.Enrollment{Token: e.Token, ExpiresAt: e.ExpiresAt, Commands: pairing.InstallCommands{Unix: e.Commands.Unix, Windows: e.Commands.Windows}}, nil
}

// ComputerRunner is pairing's seam for reading the runner that reaches a computer.
func (r *runnerComputers) ComputerRunner(ctx context.Context, computerID string) (pairing.ComputerRunner, error) {
	s, err := r.service()
	if err != nil {
		return pairing.ComputerRunner{}, err
	}
	rn, err := s.ComputerRunner(ctx, computerID)
	if err != nil {
		return pairing.ComputerRunner{}, err
	}
	return pairing.ComputerRunner{Connected: rn.Connected, LastSeen: rn.LastSeen}, nil
}

// PairingToken is pairing's seam for a one-time T3 Code pairing token minted on the computer by its runner.
func (r *runnerComputers) PairingToken(ctx context.Context, computerID string) (string, error) {
	h := r.handler.Load()
	if h == nil {
		return "", apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	return h.PairingToken(ctx, computerID)
}

// RetireComputerRunner is pairing's seam for revoking a removed computer's runner.
func (r *runnerComputers) RetireComputerRunner(ctx context.Context, computerID string) error {
	s, err := r.service()
	if err != nil {
		return err
	}
	return s.RetireComputerRunner(ctx, computerID)
}

// HandleAccountClosed revokes a closed account's personal runners, once the runner service has started.
func (r *runnerComputers) HandleAccountClosed(ctx context.Context, ev eventbus.Event) error {
	s, err := r.service()
	if err != nil {
		return err
	}
	return s.HandleAccountClosed(ctx, ev)
}

func (r *runnerComputers) DialComputer(ctx context.Context, computerID string) (net.Conn, error) {
	h := r.handler.Load()
	if h == nil {
		return nil, apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	return h.DialComputer(ctx, computerID)
}

// ComputerContainers is pairing's seam for the containers on a computer, listed by its runner.
func (r *runnerComputers) ComputerContainers(ctx context.Context, computerID string) ([]pairing.Container, error) {
	h := r.handler.Load()
	if h == nil {
		return nil, apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	found, err := h.ComputerContainers(ctx, computerID)
	if err != nil {
		return nil, err
	}
	out := make([]pairing.Container, 0, len(found))
	for _, c := range found {
		out = append(out, pairing.Container{Name: c.Name, Image: c.Image, Status: c.Status})
	}
	return out, nil
}

// OpenComputerLogs is pairing's seam for one container's logs on a computer, streamed by its runner.
func (r *runnerComputers) OpenComputerLogs(ctx context.Context, computerID, container string, tail int, follow, mask bool) (containerlog.Feed, error) {
	h := r.handler.Load()
	if h == nil {
		return nil, apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	feed, err := h.OpenComputerLogs(ctx, computerID, container, tail, follow, mask)
	if err != nil {
		return nil, err
	}
	return runnerLogFeed{feed: feed}, nil
}

// OpenRunnerLogs is pairing's seam for a computer's runner's own log, streamed by that runner.
func (r *runnerComputers) OpenRunnerLogs(ctx context.Context, computerID string, tail int, follow bool) (containerlog.Feed, error) {
	h := r.handler.Load()
	if h == nil {
		return nil, apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	feed, err := h.OpenComputerRunnerLogs(ctx, computerID, tail, follow)
	if err != nil {
		return nil, err
	}
	return runnerLogFeed{feed: feed}, nil
}

// harnessHTTPClient is both T3 clients' client: a computer's address dials through its runner, the rest as before.
func harnessHTTPClient(computers computerDialer, access cloudflare.AccessCredentials) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	dial, proxy := base.DialContext, base.Proxy
	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if id, ok := strings.CutSuffix(host, pairing.ComputerHostSuffix); ok && err == nil {
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
	return strings.HasSuffix(req.URL.Hostname(), pairing.ComputerHostSuffix)
}

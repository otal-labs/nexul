package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
	tokens  cloneTokens
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

// CloneRepository is pairing's seam for cloning onto a computer through its runner; without a token for the repository,
// or with a runner too old to clone, the clone is left to T3 Code.
func (r *runnerComputers) CloneRepository(ctx context.Context, computerID, repository, path string) (string, error) {
	h := r.handler.Load()
	if h == nil {
		return "", apperrs.Retryable(errors.New("the runner handler has not started"))
	}
	token, err := r.tokens.token(ctx, repository)
	if err != nil {
		logging.FromCtx(ctx).Info("clone left to T3 Code", "computer_id", computerID, "repo", repository, "reason", err)
		return "", pairing.ErrCloneThroughT3
	}
	dir, err := h.CloneRepository(ctx, computerID, repository, path, token)
	if errors.Is(err, runner.ErrCannotClone) {
		return "", pairing.ErrCloneThroughT3
	}
	return dir, err
}

// cloneTokens mints the token a person's computer clones with: one for that repository alone, only when it belongs to
// a project the person may open, and never the instance's own connector token.
type cloneTokens struct {
	repos  gitRepoResolver
	access pairing.ProjectGate
	git    runner.GitTokenLookup
}

func (c cloneTokens) token(ctx context.Context, repository string) (string, error) {
	if c.git == nil {
		return "", errors.New("no git tokens")
	}
	owner, name, _ := strings.Cut(repository, "/")
	ref, err := c.repos.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return "", err
	}
	if err := c.access.RequireProject(ctx, ref.ProjectID, permissions.Member); err != nil {
		return "", err
	}
	credentials, err := c.git.RepoToken(ctx, repository)
	if err != nil {
		return "", err
	}
	if credentials.AllowFallback || credentials.Token == "" {
		return "", errors.New("no token for this repository alone")
	}
	return credentials.Token, nil
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

// errNotThroughRunner refuses an address that is not a computer's runner, which is every computer paired before the
// Nexul app until it moves (ADR 0146).
var errNotThroughRunner = fmt.Errorf("%w: Nexul reaches T3 Code only through the Nexul app on the computer; add this computer with the Nexul app", apperrs.ErrInvalid)

// harnessHTTPClient is both T3 clients' client: it dials a computer's T3 Code through its runner and nothing else.
func harnessHTTPClient(computers computerDialer) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DialContext = func(ctx context.Context, _, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if id, ok := strings.CutSuffix(host, pairing.ComputerHostSuffix); ok && err == nil {
			return computers.DialComputer(ctx, id)
		}
		return nil, errNotThroughRunner
	}
	base.Proxy = nil
	return &http.Client{Transport: asLoopback{base}}
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

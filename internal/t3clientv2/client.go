// Package t3clientv2 is the harness.Client for T3 Code servers on orchestration protocol 2, over the t3rpc transport.
package t3clientv2

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

// protocol is the orchestration protocol this client speaks.
const protocol = 2

// errTurnsNotYet answers the turn calls this client cannot make yet.
var errTurnsNotYet = fmt.Errorf("%w: stopping or answering a turn on this T3 Code version needs a newer Nexul", apperrs.ErrInvalid)

// Harness is the harness.Client for T3 Code servers running T3's orchestrator V2.
type Harness struct {
	Options t3rpc.Options
}

// NewHarness wires a Harness dialing real T3 servers.
func NewHarness(opts t3rpc.Options) *Harness {
	return &Harness{Options: opts}
}

// Kind implements harness.Client.
func (h *Harness) Kind() harness.Kind { return harness.KindT3CodeV2 }

// Pair implements harness.Client; it reads the descriptor first, so a protocol-1 server never spends the one-time token.
func (h *Harness) Pair(ctx context.Context, serverURL, secret string) (harness.PairResult, error) {
	serverVersion, err := h.Version(ctx, serverURL)
	if err != nil {
		return harness.PairResult{}, err
	}
	token, expiresIn, err := t3rpc.Exchange(ctx, h.Options.HTTPClient, serverURL, secret)
	if err != nil {
		return harness.PairResult{}, fmt.Errorf("exchange pairing token: %w", err)
	}
	return harness.PairResult{BearerToken: token, ExpiresIn: expiresIn, Version: serverVersion, Kind: harness.KindT3CodeV2}, nil
}

// Version implements harness.Client via the unauthenticated well-known probe, refusing any protocol but 2.
func (h *Harness) Version(ctx context.Context, serverURL string) (string, error) {
	d, err := t3rpc.Describe(ctx, h.Options.HTTPClient, serverURL)
	if err != nil {
		return "", fmt.Errorf("read T3 version: %w", err)
	}
	if err := refusal(hostOf(serverURL), d.Protocol); err != nil {
		return "", err
	}
	return d.ServerVersion, nil
}

// ListProjects implements harness.Client: connect, read the registry, close.
func (h *Harness) ListProjects(ctx context.Context, s harness.Session) (projects []harness.Project, err error) {
	c, err := h.connect(ctx, s)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	return c.ListProjects(ctx)
}

// ListProviders implements harness.Client: connect, parse the handshake config, close.
func (h *Harness) ListProviders(ctx context.Context, s harness.Session) (providers []harness.Provider, err error) {
	c, err := h.connect(ctx, s)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	return c.Providers()
}

// Hold implements harness.Client: the WebSocket itself is the presence signal T3 shows for a paired client.
func (h *Harness) Hold(ctx context.Context, s harness.Session) (harness.Conn, error) {
	c, err := h.connect(ctx, s)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Interrupt implements harness.Client.
func (h *Harness) Interrupt(context.Context, harness.Target) error {
	return errTurnsNotYet
}

// Answer implements harness.Client.
func (h *Harness) Answer(context.Context, harness.Target, string, harness.QuestionAnswer) error {
	return errTurnsNotYet
}

// settleCommand is protocol 1's thread.settle shape, which protocol 2 kept; settledAt is optional and left out.
type settleCommand struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	ThreadID  string `json:"threadId"`
}

// Settle implements harness.Client; handed-off child threads are hidden by T3 itself, so only target's thread settles.
func (h *Harness) Settle(ctx context.Context, target harness.Target) (err error) {
	if target.SessionID == "" {
		return fmt.Errorf("%w: no session to settle", apperrs.ErrInvalid)
	}
	c, err := h.connect(ctx, target.Session)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	_, err = c.Call(ctx, "orchestration.dispatchCommand", settleCommand{Type: "thread.settle", CommandID: ids.New(), ThreadID: target.SessionID})
	if err != nil {
		return fmt.Errorf("settle t3 thread: %w", err)
	}
	return nil
}

// connect refuses on the handshake's protocol, because a protocol-1 T3 accepts a protocol-2 dial and speaks protocol 1.
func (h *Harness) connect(ctx context.Context, s harness.Session) (*t3rpc.Conn, error) {
	opts := h.Options
	opts.Query = url.Values{"orchestrationProtocol": {strconv.Itoa(protocol)}, "clientAppVersion": {"nexul/" + version.Version}}
	where := computerName(s)
	c, err := t3rpc.Connect(ctx, s, opts)
	var mismatch *t3rpc.ProtocolMismatchError
	if errors.As(err, &mismatch) {
		// Only a server past protocol 2 answers a protocol-2 dial with 426 Upgrade Required.
		return nil, newerNeeded(where)
	}
	if err != nil {
		return nil, err
	}
	if err := refusal(where, c.Protocol()); err != nil {
		_ = c.Close() // the refusal is the error worth reporting
		return nil, err
	}
	return c, nil
}

// refusal is the ErrProtocol for a T3 Code on where speaking protocol p, nil when p is the one this client speaks.
func refusal(where string, p int) error {
	if p > protocol {
		return newerNeeded(where)
	}
	if p < protocol {
		return harness.ProtocolRefusal(fmt.Sprintf("T3 Code on %s went back to its old orchestrator; Nexul only moves forward. Update T3 Code there.", where))
	}
	return nil
}

func newerNeeded(where string) error {
	return harness.ProtocolRefusal(fmt.Sprintf("T3 Code on %s needs a newer Nexul.", where))
}

// computerName names a session's computer for a refusal, its server's host when it has no name.
func computerName(s harness.Session) string {
	if s.Name != "" {
		return s.Name
	}
	return hostOf(s.ServerURL)
}

func hostOf(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil || u.Host == "" {
		return serverURL
	}
	return u.Host
}

var _ harness.Client = (*Harness)(nil)

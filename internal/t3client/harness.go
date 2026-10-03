// Package t3client is the harness.Client for T3 Code servers on orchestration protocol 1, over the t3rpc transport.
package t3client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

// Options configures every connection the harness opens.
type Options = t3rpc.Options

// conn carries the protocol-1 thread methods (thread.go) on a shared T3 connection.
type conn struct {
	*t3rpc.Conn
	log *slog.Logger
}

// threadSub narrows *Subscription so Harness is testable with a fake.
type threadSub interface {
	Updates() <-chan Update
	Close()
	Dropped() *turnWatch
}

// rpcConn is the slice of *conn Harness needs; clientAdapter narrows SubscribeThread's return type.
type rpcConn interface {
	CreateThread(ctx context.Context, t3ProjectID, title, providerInstanceID, model string, options []harness.OptionSetting, runtimeMode string) (string, error)
	StartTurn(ctx context.Context, threadID, text, runtimeMode string, attachments []harness.Attachment) error
	Interrupt(ctx context.Context, threadID string) error
	RespondApproval(ctx context.Context, threadID, requestID, decision string) error
	RespondUserInput(ctx context.Context, threadID, requestID string, answer harness.QuestionAnswer) error
	SubscribeThread(ctx context.Context, threadID string) (threadSub, error)
	ResumeThread(ctx context.Context, threadID string, w *turnWatch) (threadSub, error)
	Providers() ([]harness.Provider, error)
	Close() error
}

type clientAdapter struct{ *conn }

func (a clientAdapter) SubscribeThread(ctx context.Context, threadID string) (threadSub, error) {
	return a.conn.SubscribeThread(ctx, threadID)
}

func (a clientAdapter) ResumeThread(ctx context.Context, threadID string, w *turnWatch) (threadSub, error) {
	return a.conn.ResumeThread(ctx, threadID, w)
}

func connect(ctx context.Context, s harness.Session, opts Options) (rpcConn, error) {
	c, err := t3rpc.Connect(ctx, s, opts)
	if err != nil {
		return nil, err
	}
	return clientAdapter{&conn{Conn: c, log: logger(opts)}}, nil
}

// Harness is the harness.Client for T3 Code servers; every T3 concept stays inside this package.
type Harness struct {
	Options Options

	// connect is a test seam; NewHarness wires the real dial.
	connect func(ctx context.Context, s harness.Session, opts Options) (rpcConn, error)
}

// NewHarness wires a Harness dialing real T3 servers.
func NewHarness(opts Options) *Harness {
	return &Harness{Options: opts, connect: connect}
}

// Kind implements harness.Client.
func (h *Harness) Kind() harness.Kind { return harness.KindT3Code }

// Pair implements harness.Client: the `t3 pair` one-time token is traded for a bearer session, then the version is read.
func (h *Harness) Pair(ctx context.Context, serverURL, secret string) (harness.PairResult, error) {
	token, expiresIn, err := t3rpc.Exchange(ctx, h.Options.HTTPClient, serverURL, secret)
	if err != nil {
		return harness.PairResult{}, fmt.Errorf("exchange pairing token: %w", err)
	}
	version, err := h.Version(ctx, serverURL)
	if err != nil {
		return harness.PairResult{}, fmt.Errorf("read T3 version: %w", err)
	}
	return harness.PairResult{BearerToken: token, ExpiresIn: expiresIn, Version: version}, nil
}

// Version implements harness.Client via the unauthenticated well-known probe.
func (h *Harness) Version(ctx context.Context, serverURL string) (string, error) {
	d, err := t3rpc.Describe(ctx, h.Options.HTTPClient, serverURL)
	if err != nil {
		return "", err
	}
	return d.ServerVersion, nil
}

// ListProjects implements harness.Client: connect, read the registry, close.
func (h *Harness) ListProjects(ctx context.Context, s harness.Session) (projects []harness.Project, err error) {
	c, err := t3rpc.Connect(ctx, s, h.Options)
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
	c, err := t3rpc.Connect(ctx, s, h.Options)
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
	c, err := t3rpc.Connect(ctx, s, h.Options)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// StartTurn reuses target.SessionID's thread if given; a failed reuse is treated as "gone" and retried once fresh.
func (h *Harness) StartTurn(ctx context.Context, target harness.Target, title string, prompts harness.TurnPrompts) (harness.StartResult, error) {
	client, err := h.connect(ctx, target.Session, h.Options)
	if err != nil {
		return harness.StartResult{}, fmt.Errorf("connect t3: %w", err)
	}

	threadID := target.SessionID
	usedStored := threadID != ""
	prompt, attachments := prompts.Incremental, []harness.Attachment(nil)
	if threadID == "" {
		threadID, err = h.createThread(ctx, client, target, title)
		if err != nil {
			_ = client.Close()
			return harness.StartResult{}, err
		}
		prompt, attachments = prompts.Full, prompts.Attachments
	}

	sub, err := h.subscribeAndStart(ctx, client, threadID, prompt, attachments)
	if err != nil && usedStored {
		// A stored thread id may be stale server-side; any failure on reuse gets exactly one retry with a fresh thread.
		threadID, err = h.createThread(ctx, client, target, title)
		if err != nil {
			_ = client.Close()
			return harness.StartResult{}, err
		}
		// The fresh replacement thread has no context: send the full prompt.
		sub, err = h.subscribeAndStart(ctx, client, threadID, prompts.Full, prompts.Attachments)
	}
	if err != nil {
		_ = client.Close()
		return harness.StartResult{}, err
	}

	updates := make(chan harness.Update, 16)
	go h.pump(ctx, target.Session, client, threadID, sub, updates)
	return harness.StartResult{SessionID: threadID, Updates: updates}, nil
}

func (h *Harness) createThread(ctx context.Context, client rpcConn, target harness.Target, title string) (string, error) {
	if target.Model == "" && target.Provider != "" {
		providers, err := client.Providers()
		if err != nil {
			return "", fmt.Errorf("list t3 providers: %w", err)
		}
		target.Model, err = t3rpc.DefaultModel(providers, target.Provider)
		if err != nil {
			return "", err
		}
	}
	threadID, err := client.CreateThread(ctx, target.ProjectID, title, target.Provider, target.Model, target.ModelOptions, RuntimeModeFullAccess)
	if err != nil {
		return "", fmt.Errorf("create t3 thread: %w", err)
	}
	return threadID, nil
}

// subscribeAndStart opens the subscription before starting the turn, so no events are missed.
func (h *Harness) subscribeAndStart(ctx context.Context, client rpcConn, threadID, prompt string, attachments []harness.Attachment) (threadSub, error) {
	sub, err := client.SubscribeThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("subscribe t3 thread: %w", err)
	}
	if err := client.StartTurn(ctx, threadID, prompt, RuntimeModeFullAccess, attachments); err != nil {
		sub.Close()
		return nil, fmt.Errorf("start t3 turn: %w", err)
	}
	return sub, nil
}

// reconnectWindow outlasts a tunnel restart or network switch, yet stays inside T3's replay range and callers' silence limits.
const (
	reconnectWindow     = 5 * time.Minute
	reconnectMinBackoff = time.Second
	reconnectMaxBackoff = 30 * time.Second
)

// reconnectingNote is the muted line a turn shows while its connection is being redialed.
const reconnectingNote = "Reconnecting to T3 Code…"

// pump forwards one turn's updates; a dropped connection is redialed and the watch resumed, never surfaced as an end.
func (h *Harness) pump(ctx context.Context, s harness.Session, client rpcConn, threadID string, sub threadSub, out chan<- harness.Update) {
	defer close(out)
	for {
		h.forward(ctx, client, threadID, sub, out)
		sub.Close()
		// The turn's connection is finished with either way; a close error on a dead socket says nothing new.
		_ = client.Close()
		dropped := sub.Dropped()
		if dropped == nil {
			return
		}
		client, sub = h.reconnect(ctx, s, threadID, dropped, out)
		if sub == nil {
			return
		}
	}
}

// reconnect redials with backoff until the watch resumes, the window closes, or T3 refuses the session.
func (h *Harness) reconnect(ctx context.Context, s harness.Session, threadID string, w *turnWatch, out chan<- harness.Update) (rpcConn, threadSub) {
	out <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityNote, Summary: reconnectingNote, At: time.Now().UTC()}}
	deadline := time.Now().Add(reconnectWindow)
	backoff := reconnectMinBackoff
	for {
		client, sub, err := h.resume(ctx, deadline, s, threadID, w)
		if err == nil {
			logger(h.Options).Info("t3client: turn resumed after a dropped connection", "thread", threadID, "after_sequence", w.lastSeq)
			return client, sub
		}
		logger(h.Options).Warn("t3client: reconnect failed", "thread", threadID, "error", err)
		if errors.Is(err, apperrs.ErrUnauthorized) {
			out <- giveUp(fmt.Sprintf("Lost the connection to T3 Code and it refused to reconnect: %v", err))
			return nil, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			out <- giveUp(fmt.Sprintf("Lost the connection to T3 Code and couldn't reconnect for %s: %v", formatWindow(reconnectWindow), err))
			return nil, nil
		}
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(min(backoff, remaining)):
		}
		backoff = min(backoff*2, reconnectMaxBackoff)
	}
}

// resume bounds the dial by the window, since a tunnel can hang a request; the resumed watch lives on ctx.
func (h *Harness) resume(ctx context.Context, deadline time.Time, s harness.Session, threadID string, w *turnWatch) (rpcConn, threadSub, error) {
	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	client, err := h.connect(dialCtx, s, h.Options)
	if err != nil {
		return nil, nil, err
	}
	sub, err := client.ResumeThread(ctx, threadID, w)
	if err != nil {
		_ = client.Close() // the resume error is the one worth reporting
		return nil, nil, err
	}
	return client, sub, nil
}

func logger(opts Options) *slog.Logger {
	if opts.Logger != nil {
		return opts.Logger
	}
	return slog.Default()
}

func giveUp(reason string) harness.Update {
	return harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: reason}}
}

// formatWindow renders a whole-minute window as "5m" rather than Duration's "5m0s".
func formatWindow(d time.Duration) string {
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func (h *Harness) forward(ctx context.Context, client rpcConn, threadID string, sub threadSub, out chan<- harness.Update) {
	for u := range sub.Updates() {
		switch {
		case u.Snapshot != nil:
			out <- harness.Update{Snapshot: &harness.Snapshot{MessageID: u.Snapshot.MessageID, Text: u.Snapshot.Text, Streaming: u.Snapshot.Streaming}}
		case u.Activity != nil:
			out <- harness.Update{Activity: u.Activity}
		case u.Approval != nil:
			// The full-access default auto-declines stray approvals; the pipeline layer turns this into a system message.
			_ = client.RespondApproval(ctx, threadID, u.Approval.RequestID, DecisionDecline)
			out <- harness.Update{Approval: &harness.Approval{Kind: u.Approval.Kind, Summary: u.Approval.Summary}}
		case u.Question != nil:
			out <- harness.Update{Question: u.Question}
		case u.Terminal != nil:
			out <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnState(u.Terminal.State), LastError: u.Terminal.LastError}}
		}
	}
}

// Interrupt aborts whatever turn is active on target's thread.
func (h *Harness) Interrupt(ctx context.Context, target harness.Target) error {
	if target.SessionID == "" {
		return fmt.Errorf("%w: no active session to interrupt", apperrs.ErrInvalid)
	}
	client, err := h.connect(ctx, target.Session, h.Options)
	if err != nil {
		return fmt.Errorf("connect t3: %w", err)
	}
	defer func() { _ = client.Close() }()
	if err := client.Interrupt(ctx, target.SessionID); err != nil {
		return fmt.Errorf("interrupt t3 thread: %w", err)
	}
	return nil
}

// Answer resolves the pending question on target's thread; like Interrupt it opens its own connection, since the
// turn's own connection belongs to the pump.
func (h *Harness) Answer(ctx context.Context, target harness.Target, requestID string, answer harness.QuestionAnswer) error {
	if target.SessionID == "" {
		return fmt.Errorf("%w: no active session to answer", apperrs.ErrInvalid)
	}
	client, err := h.connect(ctx, target.Session, h.Options)
	if err != nil {
		return fmt.Errorf("connect t3: %w", err)
	}
	defer func() { _ = client.Close() }()
	if err := client.RespondUserInput(ctx, target.SessionID, requestID, answer); err != nil {
		return fmt.Errorf("answer t3 question: %w", err)
	}
	return nil
}

var _ harness.Client = (*Harness)(nil)

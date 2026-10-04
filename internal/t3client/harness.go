// Package t3client is the harness.Client for T3 Code on orchestration protocol 1; a T3 past it answers harness.MovedError.
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
	Ready(ctx context.Context) error
}

// rpcConn is the slice of *conn Harness needs; clientAdapter narrows SubscribeThread's return type.
type rpcConn interface {
	CreateThread(ctx context.Context, t3ProjectID, title, providerInstanceID, model string, options []harness.OptionSetting, runtimeMode string) (string, error)
	StartTurn(ctx context.Context, threadID, text, runtimeMode string, attachments []harness.Attachment) error
	Interrupt(ctx context.Context, threadID string) error
	Settle(ctx context.Context, threadID string) error
	RespondApproval(ctx context.Context, threadID, requestID, decision string) error
	RespondUserInput(ctx context.Context, threadID, requestID string, answer harness.QuestionAnswer) error
	DismissUserInput(ctx context.Context, threadID, requestID string) error
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
	c, err := dial(ctx, s, opts)
	if err != nil {
		return nil, err
	}
	return clientAdapter{&conn{Conn: c, log: logger(opts)}}, nil
}

// dial connects on protocol 1; T3 refuses that dial only once it speaks a later protocol, whatever its body names.
func dial(ctx context.Context, s harness.Session, opts Options) (*t3rpc.Conn, error) {
	c, err := t3rpc.Connect(ctx, s, opts)
	var mismatch *t3rpc.ProtocolMismatchError
	if errors.As(err, &mismatch) {
		return nil, movedOn(t3rpc.ComputerName(s), max(mismatch.Version, protocolV2))
	}
	return c, err
}

// protocolV2 is the orchestration protocol harness.KindT3CodeV2's client speaks.
const protocolV2 = 2

// movedOn is nil for a T3 Code on where speaking protocol 1, a MovedError for protocol 2, and a refusal past that.
func movedOn(where string, protocol int) error {
	if protocol > protocolV2 {
		return t3rpc.NewerNeeded(where)
	}
	if protocol == protocolV2 {
		return &harness.MovedError{To: harness.KindT3CodeV2}
	}
	return nil
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

// Pair implements harness.Client; it reads the descriptor first, so a T3 that moved on keeps its one-time token unspent.
func (h *Harness) Pair(ctx context.Context, serverURL, secret string) (harness.PairResult, error) {
	version, err := h.Version(ctx, serverURL)
	if err != nil {
		return harness.PairResult{}, err
	}
	token, expiresIn, err := t3rpc.Exchange(ctx, h.Options.HTTPClient, serverURL, secret)
	if err != nil {
		return harness.PairResult{}, fmt.Errorf("exchange pairing token: %w", err)
	}
	return harness.PairResult{BearerToken: token, ExpiresIn: expiresIn, Version: version, Kind: harness.KindT3Code}, nil
}

// Version implements harness.Client via the unauthenticated well-known probe, which also says whether T3 moved on.
func (h *Harness) Version(ctx context.Context, serverURL string) (string, error) {
	d, err := t3rpc.Describe(ctx, h.Options.HTTPClient, serverURL)
	if err != nil {
		return "", fmt.Errorf("read T3 version: %w", err)
	}
	if err := movedOn(t3rpc.Host(serverURL), d.Protocol); err != nil {
		return "", err
	}
	return d.ServerVersion, nil
}

// ListProjects implements harness.Client: connect, read the registry, close.
func (h *Harness) ListProjects(ctx context.Context, s harness.Session) (projects []harness.Project, err error) {
	c, err := dial(ctx, s, h.Options)
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
	c, err := dial(ctx, s, h.Options)
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
	c, err := dial(ctx, s, h.Options)
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
	if usedStored && prompts.Answer != nil && h.answeredInT3(ctx, client, threadID, prompts.Answer.RequestID) {
		_ = client.Close() // nothing was sent on it; the close error says nothing about the result
		return answeredTurn(threadID), nil
	}

	sub, err := h.subscribeAndStart(ctx, client, threadID, prompt, attachments)
	if err != nil && usedStored {
		// A stored thread id may be stale server-side; any failure on reuse gets exactly one retry with a fresh thread.
		logger(h.Options).Info("t3client: reused thread failed, creating a new one", "thread", threadID, "error", err)
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
	if target.Worktree && threadID != target.SessionID {
		updates <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityNote, Summary: noWorktreeNote, At: time.Now().UTC()}}
	}
	go h.pump(ctx, target.Session, client, threadID, sub, updates)
	return harness.StartResult{SessionID: threadID, Updates: updates, PromptSent: true}, nil
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

// noWorktreeNote is the line a new thread shows when the person asked for a worktree, which protocol 1 cannot start.
const noWorktreeNote = "Started in the T3 project's folder: update T3 Code to start new threads in a worktree"

// answeredInT3Note is the line a turn shows when the person had already answered the question in T3 Code.
const answeredInT3Note = "Already answered in T3 Code"

// answeredInT3 clears the question an ended turn left open, since the answer travels as the next turn's prompt and
// T3 would show it pending forever; true when T3 already holds an answer, so a second turn would repeat it.
func (h *Harness) answeredInT3(ctx context.Context, client rpcConn, threadID, requestID string) bool {
	err := client.DismissUserInput(ctx, threadID, requestID)
	if errors.Is(err, apperrs.ErrConflict) {
		return true
	}
	if err != nil {
		logger(h.Options).Warn("t3client: clearing the answered question failed", "thread", threadID, "request", requestID, "error", err)
	}
	return false
}

// answeredTurn ends the turn before it starts: the run T3 began from its own answer carries on there.
func answeredTurn(threadID string) harness.StartResult {
	updates := make(chan harness.Update, 2)
	updates <- harness.Update{Activity: &harness.Activity{Kind: harness.ActivityNote, Summary: answeredInT3Note, At: time.Now().UTC()}}
	updates <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	close(updates)
	return harness.StartResult{SessionID: threadID, Updates: updates, PromptSent: true}
}

// subscribeAndStart opens the subscription and reads the thread's snapshot before starting the turn, so no events are
// missed and a thread T3 no longer has fails here instead of mid-turn.
func (h *Harness) subscribeAndStart(ctx context.Context, client rpcConn, threadID, prompt string, attachments []harness.Attachment) (threadSub, error) {
	sub, err := client.SubscribeThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("subscribe t3 thread: %w", err)
	}
	if err := sub.Ready(ctx); err != nil {
		sub.Close()
		return nil, fmt.Errorf("watch t3 thread %s: %w", threadID, err)
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

// updatedMidTurn ends a turn whose T3 Code moved to another protocol while it ran; the protocol-1 watch cannot resume there.
const updatedMidTurn = "T3 Code was updated during this turn; ask again"

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
		// Read before resuming: once the watch resumes, its subscription goroutine owns lastSeq.
		after := w.lastSeq
		client, sub, err := h.resume(ctx, deadline, s, threadID, w)
		if err == nil {
			logger(h.Options).Info("t3client: turn resumed after a dropped connection", "thread", threadID, "after_sequence", after)
			return client, sub
		}
		logger(h.Options).Warn("t3client: reconnect failed", "thread", threadID, "error", err)
		if errors.Is(err, harness.ErrProtocol) || errors.As(err, new(*harness.MovedError)) {
			out <- giveUp(updatedMidTurn)
			return nil, nil
		}
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

// Watch implements harness.Client: it only subscribes, so nothing reaches T3 and the turn there carries on untouched.
func (h *Harness) Watch(ctx context.Context, target harness.Target) (harness.StartResult, error) {
	if target.SessionID == "" {
		return harness.StartResult{}, fmt.Errorf("%w: no session to watch", apperrs.ErrInvalid)
	}
	client, err := h.connect(ctx, target.Session, h.Options)
	if err != nil {
		return harness.StartResult{}, fmt.Errorf("connect t3: %w", err)
	}
	sub, err := client.ResumeThread(ctx, target.SessionID, &turnWatch{follow: true})
	if err != nil {
		_ = client.Close() // the subscribe error is the one worth reporting
		return harness.StartResult{}, fmt.Errorf("subscribe t3 thread: %w", err)
	}
	if err := sub.Ready(ctx); err != nil {
		sub.Close()
		_ = client.Close() // the watch error is the one worth reporting
		return harness.StartResult{}, fmt.Errorf("watch t3 thread %s: %w", target.SessionID, err)
	}
	updates := make(chan harness.Update, 16)
	go h.pump(ctx, target.Session, client, target.SessionID, sub, updates)
	return harness.StartResult{SessionID: target.SessionID, Updates: updates}, nil
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

// Settle settles target's thread over its own connection, like Interrupt.
func (h *Harness) Settle(ctx context.Context, target harness.Target) error {
	if target.SessionID == "" {
		return fmt.Errorf("%w: no session to settle", apperrs.ErrInvalid)
	}
	client, err := h.connect(ctx, target.Session, h.Options)
	if err != nil {
		return fmt.Errorf("connect t3: %w", err)
	}
	defer func() { _ = client.Close() }()
	if err := client.Settle(ctx, target.SessionID); err != nil {
		return fmt.Errorf("settle t3 thread: %w", err)
	}
	return nil
}

var _ harness.Client = (*Harness)(nil)

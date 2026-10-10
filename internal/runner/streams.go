package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Reaching a computer through its runner (ADR 0146): one WebSocket stream per connection, opened by the runner.
const (
	// maxStreams caps one runner's streams, pending and open, on both ends.
	maxStreams = 32
	// streamTTL is how long a stream id waits for its runner; after it the id is gone and refused.
	streamTTL = 10 * time.Second
	// defaultT3Port is where T3 Code in its default home listens when it left no runtime file.
	defaultT3Port = 3773
)

// ---- server side ----

// streamWaiter is DialComputer waiting on the runner it asked to open one stream.
type streamWaiter struct {
	runnerID string
	result   chan dialResult
	// done closes when DialComputer stops waiting, so a late stream is closed instead of handed to nobody.
	done chan struct{}
}

type dialResult struct {
	conn net.Conn
	err  error
}

// deliver hands r to the waiting DialComputer, or closes its conn when nobody waits any more.
func (w *streamWaiter) deliver(r dialResult) bool {
	select {
	case w.result <- r:
		return true
	case <-w.done:
		if r.conn != nil {
			_ = r.conn.Close()
		}
		return false
	}
}

// streamConn ends its ServeStream when the harness client closes it.
type streamConn struct {
	net.Conn
	cancel context.CancelFunc
}

func (c *streamConn) Close() error {
	defer c.cancel()
	return c.Conn.Close()
}

// DialComputer opens a connection to T3 Code on computerID through the computer's connected personal runner.
func (h *Handler) DialComputer(ctx context.Context, computerID string) (net.Conn, error) {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return nil, err
	}
	id := ids.New()
	w := &streamWaiter{runnerID: c.id, result: make(chan dialResult), done: make(chan struct{})}
	if err := h.addWaiter(id, w); err != nil {
		return nil, err
	}
	defer h.dropWaiter(id, w)
	if err := h.sendFrame(ctx, c, Frame{Type: FrameHarnessDial, ID: id}); err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("ask the runner of computer %s to dial: %w", computerID, err))
	}
	timer := time.NewTimer(streamTTL)
	defer timer.Stop()
	select {
	case r := <-w.result:
		return r.conn, r.err
	case <-timer.C:
		return nil, apperrs.Retryable(fmt.Errorf("the runner of computer %s opened no stream within %s", computerID, streamTTL))
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// computerConn is the live control connection of computerID's personal runner.
func (h *Handler) computerConn(ctx context.Context, computerID string) (*runnerConn, error) {
	rn, err := h.repo.GetByComputer(ctx, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, apperrs.Retryable(fmt.Errorf("computer %s has no runner", computerID))
	}
	if err != nil {
		return nil, fmt.Errorf("find the runner of computer %s: %w", computerID, err)
	}
	h.mu.Lock()
	c := h.conns[rn.ID]
	h.mu.Unlock()
	if c == nil || !c.personal {
		return nil, apperrs.Retryable(fmt.Errorf("the runner of computer %s is not connected", computerID))
	}
	return c, nil
}

func (h *Handler) addWaiter(id string, w *streamWaiter) error {
	h.streamsMu.Lock()
	defer h.streamsMu.Unlock()
	n := len(h.openStreams[w.runnerID])
	for _, o := range h.streamWaiters {
		if o.runnerID == w.runnerID {
			n++
		}
	}
	if n >= maxStreams {
		return apperrs.Retryable(fmt.Errorf("the runner already has %d streams open", maxStreams))
	}
	h.streamWaiters[id] = w
	return nil
}

func (h *Handler) dropWaiter(id string, w *streamWaiter) {
	h.streamsMu.Lock()
	if h.streamWaiters[id] == w {
		delete(h.streamWaiters, id)
	}
	h.streamsMu.Unlock()
	close(w.done)
}

// claimStream takes stream id for runnerID, once, and tracks it as open; an id issued to another runner stays put.
func (h *Handler) claimStream(id, runnerID string, cancel context.CancelFunc) (*streamWaiter, bool) {
	h.streamsMu.Lock()
	defer h.streamsMu.Unlock()
	w, ok := h.streamWaiters[id]
	if !ok || w.runnerID != runnerID {
		return nil, false
	}
	delete(h.streamWaiters, id)
	if h.openStreams[runnerID] == nil {
		h.openStreams[runnerID] = map[string]context.CancelFunc{}
	}
	h.openStreams[runnerID][id] = cancel
	return w, true
}

func (h *Handler) untrackStream(runnerID, id string) {
	h.streamsMu.Lock()
	defer h.streamsMu.Unlock()
	delete(h.openStreams[runnerID], id)
	if len(h.openStreams[runnerID]) == 0 {
		delete(h.openStreams, runnerID)
	}
}

// closeStreams ends runnerID's open streams, or every runner's when runnerID is empty.
func (h *Handler) closeStreams(runnerID string) {
	h.streamsMu.Lock()
	defer h.streamsMu.Unlock()
	for rid, streams := range h.openStreams {
		if runnerID != "" && rid != runnerID {
			continue
		}
		for _, cancel := range streams {
			cancel()
		}
	}
}

// ServeStream is GET /api/runners/streams/{id}: accepted once, from the issuing runner, while DialComputer waits.
func (h *Handler) ServeStream(w http.ResponseWriter, r *http.Request) {
	rn, _, ok := h.admit(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	// The request context ends only when this handler returns, so the stream lives as long as it blocks below.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	waiter, ok := h.claimStream(id, rn.ID, cancel)
	if !ok {
		h.log.Warn("stream refused", "runner_id", rn.ID, "stream", id)
		http.Error(w, "unknown stream", http.StatusNotFound)
		return
	}
	defer h.untrackStream(rn.ID, id)
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		waiter.deliver(dialResult{err: apperrs.Retryable(fmt.Errorf("accept stream: %w", err))})
		return
	}
	defer func() { _ = ws.CloseNow() }()
	if !waiter.deliver(dialResult{conn: &streamConn{Conn: websocket.NetConn(ctx, ws, websocket.MessageBinary), cancel: cancel}}) {
		return
	}
	<-ctx.Done()
}

// deliverRefusal routes a harness_dial_refused to the DialComputer waiting on it, if this runner was asked.
func (h *Handler) deliverRefusal(c *runnerConn, f Frame) {
	h.streamsMu.Lock()
	w, ok := h.streamWaiters[f.ID]
	if ok && w.runnerID == c.id {
		delete(h.streamWaiters, f.ID)
	}
	h.streamsMu.Unlock()
	if !ok || w.runnerID != c.id {
		return
	}
	w.deliver(dialResult{err: apperrs.Retryable(fmt.Errorf("the computer's runner could not reach T3 Code: %s", f.Error))})
}

// ---- runner side ----

// startStream answers harness_dial with one stream to T3 Code; streamCtx outlives a control reconnect.
func (c *Client) startStream(streamCtx, ctx context.Context, control *websocket.Conn, id string) {
	refuse := func(err error) {
		c.log.Warn("stream refused", "runner", c.cfg.Name, "stream", id, "error", err)
		c.sendFrame(ctx, control, Frame{Type: FrameHarnessDialRefused, ID: id, Error: err.Error()})
	}
	if !c.cfg.Personal {
		refuse(errors.New("this runner is not a personal runner"))
		return
	}
	if c.openStreams.Add(1) > maxStreams {
		c.openStreams.Add(-1)
		refuse(fmt.Errorf("this runner already has %d streams open", maxStreams))
		return
	}
	c.streams.Go(func() {
		defer c.openStreams.Add(-1)
		defer c.recoverStream(id)
		local, err := c.dialT3(streamCtx)
		if err != nil {
			refuse(err)
			return
		}
		c.stream(streamCtx, id, local)
	})
}

// dialT3 connects to T3 Code on loopback, at the port its runtime file names; it never dials anything else.
func (c *Client) dialT3(ctx context.Context) (net.Conn, error) {
	addr, err := t3Address(c.cfg.T3Home, defaultT3Home())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	defer cancel()
	conn, err := c.dial(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("T3 Code is not answering on %s: %w", addr, err)
	}
	return conn, nil
}

// stream opens stream id to the server and copies bytes between it and local until either side closes.
func (c *Client) stream(ctx context.Context, id string, local net.Conn) {
	defer func() { _ = local.Close() }()
	dialCtx, cancel := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	ws, _, err := websocket.Dial(dialCtx, c.cfg.StreamURL+"/"+url.PathEscape(id), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + c.cfg.Credential}},
	})
	cancel()
	if err != nil {
		c.log.Warn("stream dial failed", "runner", c.cfg.Name, "stream", id, "error", err)
		return
	}
	c.log.Debug("stream open", "runner", c.cfg.Name, "stream", id, "t3", local.RemoteAddr().String())
	c.splice(id, websocket.NetConn(ctx, ws, websocket.MessageBinary), local)
	c.log.Debug("stream closed", "runner", c.cfg.Name, "stream", id)
}

// splice copies both ways until either side stops, then closes both; a panic in either copy ends only this pair.
func (c *Client) splice(id string, a, b net.Conn) {
	var once sync.Once
	closeBoth := func() {
		once.Do(func() {
			_ = a.Close()
			_ = b.Close()
		})
	}
	pipe := func(dst, src net.Conn) {
		defer closeBoth()
		defer c.recoverStream(id)
		_, _ = io.Copy(dst, src)
	}
	var wg sync.WaitGroup
	wg.Go(func() { pipe(a, b) })
	pipe(b, a)
	wg.Wait()
}

func (c *Client) recoverStream(id string) {
	if r := recover(); r != nil {
		c.log.Error("stream panicked", "runner", c.cfg.Name, "stream", id, "panic", r)
	}
}

// refusal is the terminal answer a personal runner gives a frame it never runs, so a waiting server fails fast.
func refusal(f Frame) (Frame, bool) {
	const reason = "a personal runner does not run this"
	switch f.Type {
	case FrameAssignBuild, FrameAssignDeploy:
		return Frame{Type: FrameDeployResult, ID: f.ID, Status: DeployStatusFailed, Error: reason}, true
	case FrameAssignUpgrade:
		return Frame{Type: FrameUpgradeResult, ID: f.ID, Status: UpgradeStatusFailed, Error: reason}, true
	case FrameJoinNetworks:
		return Frame{Type: FrameJoinNetworksResult, GatewayContainer: f.GatewayContainer, Status: BuildStatusFailed, Error: reason}, true
	case FrameDiscover:
		return Frame{Type: FrameDiscoverResult, ID: f.ID, Error: reason}, true
	case FrameLogsRequest:
		return Frame{Type: FrameLogsEnd, ID: f.ID, Error: reason}, true
	}
	return Frame{}, false
}

// t3Address reads T3 Code's port from home's runtime file; only the default home may fall back to the default port.
func t3Address(home, defaultHome string) (string, error) {
	data, err := os.ReadFile(filepath.Join(home, "userdata", "server-runtime.json"))
	if errors.Is(err, fs.ErrNotExist) && home != "" && filepath.Clean(home) == filepath.Clean(defaultHome) {
		return net.JoinHostPort("127.0.0.1", strconv.Itoa(defaultT3Port)), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("T3 Code isn't running in %s", home)
	}
	if err != nil {
		return "", fmt.Errorf("read T3 Code's runtime file: %w", err)
	}
	var rt struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := json.Unmarshal(data, &rt); err != nil {
		return "", fmt.Errorf("decode T3 Code's runtime file: %w", err)
	}
	if rt.Port < 1 || rt.Port > 65535 {
		return "", fmt.Errorf("T3 Code's runtime file names port %d", rt.Port)
	}
	host, err := loopbackHost(rt.Host)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.Itoa(rt.Port)), nil
}

// loopbackHost maps the address T3 Code bound to the loopback address that reaches it, and refuses any other.
func loopbackHost(bound string) (string, error) {
	bound = strings.Trim(bound, "[]")
	switch bound {
	case "", "0.0.0.0", "localhost":
		return "127.0.0.1", nil
	case "::":
		return "::1", nil
	}
	if ip := net.ParseIP(bound); ip != nil && ip.IsLoopback() {
		return ip.String(), nil
	}
	return "", fmt.Errorf("T3 Code listens on %s, which %w", bound, errNotLoopback)
}

// errNotLoopback is T3 Code bound to an address the runner refuses to reach.
var errNotLoopback = errors.New("is not a loopback address")

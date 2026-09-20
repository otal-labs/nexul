// Package t3client speaks T3's internal Effect RPC protocol over one WebSocket, pinned to v0.0.34.
package t3client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const (
	wsTicketPath = "/api/auth/websocket-ticket"
	wsPath       = "/ws"

	// maxFrameBytes bounds inbound WS frames well above T3's own payload caps.
	maxFrameBytes = 1 << 25

	maxHTTPResponseBytes = 1 << 20
)

// Options configures Connect. The zero value is usable: http.DefaultClient, slog.Default, 30s per-RPC timeout.
type Options struct {
	HTTPClient *http.Client
	Logger     *slog.Logger
	// RPCTimeout bounds each unary RPC round-trip and each WS write.
	RPCTimeout time.Duration
}

// Client is one authenticated Effect RPC session; safe for concurrent use, Close ends it.
type Client struct {
	conn    *websocket.Conn
	log     *slog.Logger
	timeout time.Duration
	// config is the server.getConfig handshake result, kept so providers can be read without a second RPC (providers.go).
	config json.RawMessage

	// writeMu serializes WS writes: a websocket.Conn allows only one writer at a time (concurrent writers corrupt frames).
	writeMu sync.Mutex

	mu      sync.Mutex
	pending map[string]chan serverEnvelope
	nextID  uint64
	err     error // set once, before done closes

	cancel context.CancelFunc
	ctx    context.Context // connection lifetime, independent of Connect's ctx
	done   chan struct{}
}

// requestEnvelope is the client→server Effect RPC frame: one JSON payload per WS frame, correlated by id.
type requestEnvelope struct {
	Tag     string     `json:"_tag"`
	ID      string     `json:"id"`
	RPCTag  string     `json:"tag"`
	Payload any        `json:"payload,omitempty"`
	Headers [][]string `json:"headers"`
}

// interruptEnvelope cancels an in-flight request/stream server-side.
type interruptEnvelope struct {
	Tag       string `json:"_tag"`
	RequestID string `json:"requestId"`
}

// ackEnvelope opens the next chunk's latch; a client that never acks gets exactly one chunk, then silence.
type ackEnvelope struct {
	Tag       string `json:"_tag"`
	RequestID string `json:"requestId"`
}

// ack releases the next chunk; a failure just logs, a dead connection surfaces via c.done anyway.
func (c *Client) ack(id string) {
	if err := c.send(c.ctx, ackEnvelope{Tag: "Ack", RequestID: id}); err != nil {
		c.log.Warn("t3client: ack failed", "request", id, "error", err)
		return
	}
	c.log.Debug("t3client: ack sent", "request", id)
}

type pongEnvelope struct {
	Tag string `json:"_tag"`
}

// requestID absorbs Effect's string|number request ids into one string form for map correlation.
type requestID string

func (r *requestID) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*r = requestID(s)
		return nil
	}
	*r = requestID(b)
	return nil
}

// serverEnvelope is any server→client frame; which fields are set depends on Tag.
type serverEnvelope struct {
	Tag       string            `json:"_tag"`
	RequestID requestID         `json:"requestId"`
	Values    []json.RawMessage `json:"values"`
	Exit      json.RawMessage   `json:"exit"`
	Defect    json.RawMessage   `json:"defect"`
}

type exitBody struct {
	Tag   string          `json:"_tag"`
	Value json.RawMessage `json:"value"`
	Cause json.RawMessage `json:"cause"`
}

// Connect mints a wsTicket, opens the WS, and completes the server.getConfig handshake. Callers own Close.
func Connect(ctx context.Context, s harness.Session, opts Options) (*Client, error) {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	timeout := opts.RPCTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	serverURL := strings.TrimRight(s.ServerURL, "/")

	ticket, err := mintWSTicket(ctx, httpClient, serverURL, s.BearerToken)
	if err != nil {
		return nil, err
	}

	dialCtx, cancelDial := context.WithTimeout(ctx, timeout)
	defer cancelDial()
	wsURL := serverURL + wsPath + "?wsTicket=" + url.QueryEscape(ticket)
	conn, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{HTTPClient: httpClient})
	if err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("dial T3 websocket: %w", err))
	}
	conn.SetReadLimit(maxFrameBytes)

	connCtx, cancel := context.WithCancel(context.Background())
	c := &Client{
		conn:    conn,
		log:     log,
		timeout: timeout,
		pending: map[string]chan serverEnvelope{},
		cancel:  cancel,
		ctx:     connCtx,
		done:    make(chan struct{}),
	}
	go c.readLoop()

	config, err := c.call(ctx, "server.getConfig", nil)
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("server.getConfig handshake: %w", err)
	}
	c.config = config
	c.log.Debug("t3client: connected", "server", serverURL, "config", snippet(config))
	return c, nil
}

// Close ends the session; in-flight calls and subscriptions fail/close.
func (c *Client) Close() error {
	c.cancel()
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// Done is closed once the session is dead (read error or Close); the presence keeper redials on it.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

type wsTicketResponse struct {
	Ticket   string `json:"ticket"`
	WSTicket string `json:"wsTicket"`
	Token    string `json:"token"`
}

// mintWSTicket trades the bearer session for a short-lived WS ticket; the response schema is undocumented.
func mintWSTicket(ctx context.Context, client *http.Client, serverURL, bearerToken string) (ticket string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+wsTicketPath, nil)
	if err != nil {
		return "", fmt.Errorf("build websocket-ticket request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", apperrs.Retryable(fmt.Errorf("reach T3 server: %w", err))
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseBytes))
	if err != nil {
		return "", fmt.Errorf("read websocket-ticket response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("%w: T3 rejected the bearer token (%d): %s", apperrs.ErrUnauthorized, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: T3 websocket-ticket responded %d: %s", apperrs.ErrInvalid, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out wsTicketResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode websocket-ticket response: %w", err)
	}
	for _, ticket := range []string{out.Ticket, out.WSTicket, out.Token} {
		if ticket != "" {
			return ticket, nil
		}
	}
	return "", fmt.Errorf("%w: T3 websocket-ticket response has no recognizable ticket field: %s", apperrs.ErrInvalid, snippet(body))
}

// readLoop is the single reader demultiplexing frames to pending requests.
func (c *Client) readLoop() {
	for {
		_, data, err := c.conn.Read(c.ctx)
		if err != nil {
			c.fail(err)
			return
		}
		c.log.Debug("t3client: frame received", "raw", snippet(data))
		c.handleFrame(data)
	}
}

// handleFrame accepts either one envelope or a JSON array batch of envelopes.
func (c *Client) handleFrame(data []byte) {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(data, &batch); err != nil {
			c.log.Warn("t3client: malformed batch frame skipped", "error", err, "frame", snippet(data))
			return
		}
		for _, one := range batch {
			c.handleEnvelope(one)
		}
		return
	}
	c.handleEnvelope(data)
}

func (c *Client) handleEnvelope(data []byte) {
	var env serverEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		c.log.Warn("t3client: malformed frame skipped", "error", err, "frame", snippet(data))
		return
	}
	switch env.Tag {
	case "Chunk", "Exit":
		c.deliver(env)
	case "Defect":
		// A Defect is connection-fatal in Effect RPC: fail everything in flight.
		c.log.Error("t3client: server defect", "defect", snippet(env.Defect))
		c.fail(fmt.Errorf("t3 server defect: %s", snippet(env.Defect)))
	case "Ping":
		// Ping is documented as client→server only, but T3's Effect patch adds ping/pong hooks; answer just in case.
		_ = c.send(c.ctx, pongEnvelope{Tag: "Pong"})
	case "Pong":
	default:
		c.log.Debug("t3client: unknown frame tag skipped", "tag", env.Tag, "frame", snippet(data))
	}
}

func (c *Client) deliver(env serverEnvelope) {
	c.mu.Lock()
	ch := c.pending[string(env.RequestID)]
	c.mu.Unlock()
	if ch == nil {
		c.log.Debug("t3client: frame for unknown request skipped", "request_id", string(env.RequestID), "tag", env.Tag)
		return
	}
	select {
	case ch <- env:
	default:
		// ponytail: drop-on-full, snapshots are cumulative so this self-heals; block-with-disconnect if drops bite.
		c.log.Warn("t3client: dropping frame, consumer too slow", "request_id", string(env.RequestID), "tag", env.Tag)
	}
}

// fail records the first connection error and wakes every waiter.
func (c *Client) fail(err error) {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
		close(c.done)
	}
	c.mu.Unlock()
	c.cancel()
}

func (c *Client) register() (string, chan serverEnvelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	id := strconv.FormatUint(c.nextID, 10)
	ch := make(chan serverEnvelope, 128)
	c.pending[id] = ch
	return id, ch
}

func (c *Client) unregister(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// send writes one frame under the write mutex with the client's timeout.
func (c *Client) send(ctx context.Context, v any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return wsjson.Write(ctx, c.conn, v)
}

// call performs one unary RPC; a result may arrive in the Exit or a preceding Chunk, both accepted.
func (c *Client) call(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	if payload == nil {
		// A missing payload dies on upstream ≥0.0.35; no-input calls must still carry {}.
		payload = struct{}{}
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	id, ch := c.register()
	defer c.unregister(id)
	env := requestEnvelope{Tag: "Request", ID: id, RPCTag: method, Payload: payload, Headers: [][]string{}}
	if err := c.send(ctx, env); err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("send %s: %w", method, err))
	}
	var chunkResult json.RawMessage
	for {
		select {
		case in := <-ch:
			switch in.Tag {
			case "Chunk":
				if len(in.Values) > 0 {
					chunkResult = in.Values[len(in.Values)-1]
				}
			case "Exit":
				return decodeExit(method, in.Exit, chunkResult)
			}
		case <-ctx.Done():
			_ = c.send(c.ctx, interruptEnvelope{Tag: "Interrupt", RequestID: id})
			return nil, fmt.Errorf("%s: %w", method, ctx.Err())
		case <-c.done:
			return nil, apperrs.Retryable(fmt.Errorf("%s: T3 connection lost: %w", method, c.err))
		}
	}
}

func decodeExit(method string, raw, chunkResult json.RawMessage) (json.RawMessage, error) {
	var exit exitBody
	if err := json.Unmarshal(raw, &exit); err != nil {
		return nil, fmt.Errorf("%s: malformed exit frame: %w", method, err)
	}
	if exit.Tag != "Success" {
		return nil, fmt.Errorf("%w: %s failed: %s", apperrs.ErrInvalid, method, snippet(exit.Cause))
	}
	if len(exit.Value) > 0 && !bytes.Equal(bytes.TrimSpace(exit.Value), []byte("null")) {
		return exit.Value, nil
	}
	return chunkResult, nil
}

// snippet bounds raw wire data for log/error messages.
func snippet(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}

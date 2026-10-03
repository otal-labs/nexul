// Package t3rpc speaks T3 Code's Effect RPC protocol over one WebSocket: the transport both orchestration protocols share.
package t3rpc

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
)

// Options configures Connect. The zero value is usable: http.DefaultClient, slog.Default, 30s per-RPC timeout.
type Options struct {
	HTTPClient *http.Client
	Logger     *slog.Logger
	// RPCTimeout bounds each unary RPC round-trip and each WS write.
	RPCTimeout time.Duration
	// Query rides on the /ws dial beside wsTicket.
	Query url.Values
}

// Conn is one authenticated Effect RPC session; safe for concurrent use, Close ends it.
type Conn struct {
	conn    *websocket.Conn
	log     *slog.Logger
	timeout time.Duration
	// config is the server.getConfig handshake result, kept so providers can be read without a second RPC (providers.go).
	config   json.RawMessage
	protocol int

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
func (c *Conn) ack(id string) {
	if err := c.send(c.ctx, ackEnvelope{Tag: "Ack", RequestID: id}); err != nil {
		c.log.Warn("t3rpc: ack failed", "request", id, "error", err)
		return
	}
	c.log.Debug("t3rpc: ack sent", "request", id)
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
func Connect(ctx context.Context, s harness.Session, opts Options) (*Conn, error) {
	client := httpClient(opts.HTTPClient)
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	timeout := opts.RPCTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	serverURL := strings.TrimRight(s.ServerURL, "/")

	ticket, err := mintWSTicket(ctx, client, serverURL, s.BearerToken)
	if err != nil {
		return nil, err
	}

	dialCtx, cancelDial := context.WithTimeout(ctx, timeout)
	defer cancelDial()
	query := url.Values{}
	for k, v := range opts.Query {
		query[k] = v
	}
	query.Set("wsTicket", ticket)
	conn, resp, err := websocket.Dial(dialCtx, serverURL+wsPath+"?"+query.Encode(), &websocket.DialOptions{HTTPClient: client})
	if resp != nil && resp.StatusCode == http.StatusUpgradeRequired {
		return nil, protocolMismatch(resp.Body)
	}
	if err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("dial T3 websocket: %w", err))
	}
	conn.SetReadLimit(maxFrameBytes)

	connCtx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		conn:    conn,
		log:     log,
		timeout: timeout,
		pending: map[string]chan serverEnvelope{},
		cancel:  cancel,
		ctx:     connCtx,
		done:    make(chan struct{}),
	}
	go c.readLoop()

	config, err := c.Call(ctx, "server.getConfig", nil)
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("server.getConfig handshake: %w", err)
	}
	var handshake struct {
		Environment Descriptor `json:"environment"`
	}
	if err := json.Unmarshal(config, &handshake); err != nil {
		c.log.Debug("t3rpc: handshake environment not decoded, assuming protocol 1", "error", err)
	}
	c.config, c.protocol = config, protocolOrOne(handshake.Environment.Protocol)
	c.log.Debug("t3rpc: connected", "server", serverURL, "protocol", c.protocol, "config", Snippet(config))
	return c, nil
}

// ProtocolMismatchError is T3 refusing the dial (HTTP 426) because it speaks orchestration protocol Version.
type ProtocolMismatchError struct {
	Version int
}

func (e *ProtocolMismatchError) Error() string {
	return fmt.Sprintf("T3 Code refused the connection: it speaks orchestration protocol %d", e.Version)
}

// protocolMismatch reads the version T3 names in its 426 body.
func protocolMismatch(body io.Reader) *ProtocolMismatchError {
	var refusal struct {
		Version int `json:"orchestrationProtocolVersion"`
	}
	// An unreadable body is still a refusal; Version stays 0, meaning unknown.
	_ = json.NewDecoder(body).Decode(&refusal)
	return &ProtocolMismatchError{Version: refusal.Version}
}

// Close ends the session; in-flight calls and subscriptions fail/close.
func (c *Conn) Close() error {
	c.cancel()
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// Done is closed once the session is dead (read error or Close); the presence keeper redials on it.
func (c *Conn) Done() <-chan struct{} {
	return c.done
}

// Err is why the session died, nil while it is alive.
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Config is the raw server.getConfig handshake result.
func (c *Conn) Config() json.RawMessage {
	return c.config
}

// Protocol is the orchestration protocol the handshake's environment names, 1 when it names none.
func (c *Conn) Protocol() int {
	return c.protocol
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
	return "", fmt.Errorf("%w: T3 websocket-ticket response has no recognizable ticket field: %s", apperrs.ErrInvalid, Snippet(body))
}

// readLoop is the single reader demultiplexing frames to pending requests.
func (c *Conn) readLoop() {
	for {
		_, data, err := c.conn.Read(c.ctx)
		if err != nil {
			c.fail(err)
			return
		}
		c.log.Debug("t3rpc: frame received", "raw", Snippet(data))
		c.handleFrame(data)
	}
}

// handleFrame accepts either one envelope or a JSON array batch of envelopes.
func (c *Conn) handleFrame(data []byte) {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(data, &batch); err != nil {
			c.log.Warn("t3rpc: malformed batch frame skipped", "error", err, "frame", Snippet(data))
			return
		}
		for _, one := range batch {
			c.handleEnvelope(one)
		}
		return
	}
	c.handleEnvelope(data)
}

func (c *Conn) handleEnvelope(data []byte) {
	var env serverEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		c.log.Warn("t3rpc: malformed frame skipped", "error", err, "frame", Snippet(data))
		return
	}
	switch env.Tag {
	case "Chunk", "Exit":
		c.deliver(env)
	case "Defect":
		// A Defect is connection-fatal in Effect RPC: fail everything in flight.
		c.log.Error("t3rpc: server defect", "defect", Snippet(env.Defect))
		c.fail(fmt.Errorf("t3 server defect: %s", Snippet(env.Defect)))
	case "Ping":
		// Ping is documented as client→server only, but T3's Effect patch adds ping/pong hooks; answer just in case.
		_ = c.send(c.ctx, pongEnvelope{Tag: "Pong"})
	case "Pong":
	default:
		c.log.Debug("t3rpc: unknown frame tag skipped", "tag", env.Tag, "frame", Snippet(data))
	}
}

func (c *Conn) deliver(env serverEnvelope) {
	c.mu.Lock()
	ch := c.pending[string(env.RequestID)]
	c.mu.Unlock()
	if ch == nil {
		c.log.Debug("t3rpc: frame for unknown request skipped", "request_id", string(env.RequestID), "tag", env.Tag)
		return
	}
	select {
	case ch <- env:
	default:
		// ponytail: drop-on-full, snapshots are cumulative so this self-heals; block-with-disconnect if drops bite.
		c.log.Warn("t3rpc: dropping frame, consumer too slow", "request_id", string(env.RequestID), "tag", env.Tag)
	}
}

// fail records the first connection error and wakes every waiter.
func (c *Conn) fail(err error) {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
		close(c.done)
	}
	c.mu.Unlock()
	c.cancel()
}

func (c *Conn) register() (string, chan serverEnvelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	id := strconv.FormatUint(c.nextID, 10)
	ch := make(chan serverEnvelope, 128)
	c.pending[id] = ch
	return id, ch
}

func (c *Conn) unregister(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// send writes one frame under the write mutex with the client's timeout.
func (c *Conn) send(ctx context.Context, v any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return wsjson.Write(ctx, c.conn, v)
}

// ErrConnectionLost wraps the error of a call or stream whose connection died under it.
var ErrConnectionLost = errors.New("T3 connection lost")

// request opens one RPC: it registers the id the server's frames come back under, then sends the Request frame.
func (c *Conn) request(ctx context.Context, method string, payload any) (string, chan serverEnvelope, error) {
	if payload == nil {
		// A missing payload dies on upstream ≥0.0.35; no-input calls must still carry {}.
		payload = struct{}{}
	}
	id, ch := c.register()
	env := requestEnvelope{Tag: "Request", ID: id, RPCTag: method, Payload: payload, Headers: [][]string{}}
	if err := c.send(ctx, env); err != nil {
		c.unregister(id)
		return "", nil, apperrs.Retryable(fmt.Errorf("send %s: %w", method, err))
	}
	return id, ch, nil
}

func (c *Conn) lost(method string) error {
	return apperrs.Retryable(fmt.Errorf("%s: %w: %w", method, ErrConnectionLost, c.Err()))
}

// Call performs one unary RPC; a result may arrive in the Exit or a preceding Chunk, both accepted.
func (c *Conn) Call(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	id, ch, err := c.request(ctx, method, payload)
	if err != nil {
		return nil, err
	}
	defer c.unregister(id)
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
			return nil, c.lost(method)
		}
	}
}

// Stream is one streaming RPC; Next hands over each Chunk's values, Close interrupts it server-side.
type Stream struct {
	c      *Conn
	id     string
	method string
	ch     chan serverEnvelope
	// unacked is set while the caller holds a chunk; the server sends the next one only after its Ack.
	unacked bool
}

// Stream opens method as a stream; the caller owns Close.
func (c *Conn) Stream(ctx context.Context, method string, payload any) (*Stream, error) {
	id, ch, err := c.request(ctx, method, payload)
	if err != nil {
		return nil, err
	}
	return &Stream{c: c, id: id, method: method, ch: ch}, nil
}

// Next acks the chunk it returned last, then waits for the next one. A clean end is io.EOF, a failed one an
// *ExitError, and a dead connection wraps ErrConnectionLost.
func (s *Stream) Next(ctx context.Context) ([]json.RawMessage, error) {
	if s.unacked {
		s.unacked = false
		s.c.ack(s.id)
	}
	select {
	case in := <-s.ch:
		if in.Tag == "Exit" {
			if _, err := decodeExit(s.method, in.Exit, nil); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		s.unacked = true
		return in.Values, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.c.done:
		return nil, s.c.lost(s.method)
	}
}

// Close stops the stream; the Interrupt is best-effort, since a dead connection has already ended it.
func (s *Stream) Close() {
	s.c.unregister(s.id)
	_ = s.c.send(s.c.ctx, interruptEnvelope{Tag: "Interrupt", RequestID: s.id})
}

// ExitError is an RPC that ended in an Effect Failure; it matches apperrs.ErrInvalid.
type ExitError struct {
	Method string
	// Causes are the failure's cause entries, so a caller can tell a typed failure from a defect.
	Causes []ExitCause
	cause  json.RawMessage
}

// ExitCause is one cause entry: Tag is Fail, Die or Interrupt, and a Fail carries its typed error.
type ExitCause struct {
	Tag   string `json:"_tag"`
	Error struct {
		Tag     string `json:"_tag"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("%s: %s failed: %s", apperrs.ErrInvalid, e.Method, Snippet(e.cause))
}

func (e *ExitError) Unwrap() error { return apperrs.ErrInvalid }

// Failed reports whether a Fail cause carries the typed error errorTag, such as OrchestrationV2GetThreadProjectionError.
func (e *ExitError) Failed(errorTag string) bool {
	for _, c := range e.Causes {
		if c.Tag == "Fail" && c.Error.Tag == errorTag {
			return true
		}
	}
	return false
}

func newExitError(method string, cause json.RawMessage) *ExitError {
	e := &ExitError{Method: method, cause: cause}
	// An undecodable cause still fails the call; only the typed checks lose it, and Error keeps the raw text.
	_ = json.Unmarshal(cause, &e.Causes)
	return e
}

func decodeExit(method string, raw, chunkResult json.RawMessage) (json.RawMessage, error) {
	var exit exitBody
	if err := json.Unmarshal(raw, &exit); err != nil {
		return nil, fmt.Errorf("%s: malformed exit frame: %w", method, err)
	}
	if exit.Tag != "Success" {
		return nil, newExitError(method, exit.Cause)
	}
	if len(exit.Value) > 0 && !bytes.Equal(bytes.TrimSpace(exit.Value), []byte("null")) {
		return exit.Value, nil
	}
	return chunkResult, nil
}

// Snippet bounds raw wire data for log and error messages.
func Snippet(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}

// httpClient is client, or http.DefaultClient when the caller left it nil.
func httpClient(client *http.Client) *http.Client {
	if client == nil {
		return http.DefaultClient
	}
	return client
}

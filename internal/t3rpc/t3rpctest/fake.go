// Package t3rpctest is a fake T3 Code server every T3 client's tests share: the wsTicket mint, the /ws Effect RPC
// socket, and the environment descriptor.
package t3rpctest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/otal-labs/nexul/internal/harness"
)

// Server stands in for T3 Code, which is never reachable in CI; it speaks frames recorded from the real server.
type Server struct {
	*httptest.Server
	t      testing.TB
	bearer string
	ticket string

	// TicketField names the websocket-ticket response field, since T3 has used more than one.
	TicketField string
	// Protocol is the orchestration protocol the descriptor and getConfig name, 0 to leave it out as T3 before
	// negotiation does. From 2 the /ws dial needs a matching orchestrationProtocol query, or it is refused with 426.
	Protocol int

	ConfigCalls atomic.Int32
	Subscribed  chan string         // requestId of each subscribeThread stream
	SubscribeIn chan map[string]any // payload of each subscribeThread request
	Acks        chan string         // requestId of each Ack frame received
	Dispatched  chan map[string]any // payload of each dispatchCommand (auto-acked)
	// DispatchCause fails every dispatchCommand when set; set it before connect.
	DispatchCause any

	connMu sync.Mutex
	conn   *websocket.Conn
}

type clientEnv struct {
	Tag     string          `json:"_tag"`
	ID      json.RawMessage `json:"id"`
	RPCTag  string          `json:"tag"`
	Payload json.RawMessage `json:"payload"`
}

// New starts a fake T3 server that t's cleanup stops.
func New(t testing.TB) *Server {
	f := &Server{
		t:           t,
		bearer:      "bearer-token",
		ticket:      "ws-ticket-1",
		TicketField: "ticket",
		Subscribed:  make(chan string, 4),
		SubscribeIn: make(chan map[string]any, 4),
		Acks:        make(chan string, 16),
		Dispatched:  make(chan map[string]any, 16),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/websocket-ticket", f.handleTicket)
	mux.HandleFunc("/ws", f.handleWS)
	mux.HandleFunc("/.well-known/t3/environment", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(f.descriptor())
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// Session is a session the fake accepts.
func (f *Server) Session() harness.Session {
	return harness.Session{ServerURL: f.URL, BearerToken: f.bearer}
}

func (f *Server) descriptor() map[string]any {
	d := map[string]any{"serverVersion": "0.0.34"}
	if f.Protocol != 0 {
		d["orchestrationProtocolVersion"] = f.Protocol
	}
	return d
}

func (f *Server) handleTicket(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+f.bearer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{f.TicketField: f.ticket})
}

// refuseProtocol answers a dial from another protocol the way T3 does, before it looks at the ticket.
func (f *Server) refuseProtocol(w http.ResponseWriter, r *http.Request) bool {
	if f.Protocol < 2 || r.URL.Query().Get("orchestrationProtocol") == strconv.Itoa(f.Protocol) {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUpgradeRequired)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":                         "orchestration_protocol_incompatible",
		"message":                      fmt.Sprintf("Update this client to one that supports orchestration protocol %d.", f.Protocol),
		"orchestrationProtocolVersion": f.Protocol,
	})
	return true
}

func (f *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if f.refuseProtocol(w, r) {
		return
	}
	if r.URL.Query().Get("wsTicket") != f.ticket {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	f.connMu.Lock()
	f.conn = conn
	f.connMu.Unlock()
	for {
		var env clientEnv
		if err := wsjson.Read(r.Context(), conn, &env); err != nil {
			return
		}
		if env.Tag == "Ack" {
			select {
			case f.Acks <- idString(env.ID):
			default:
			}
			continue
		}
		if env.Tag != "Request" {
			continue // Interrupt / Pong / anything else: fine to ignore in the fake
		}
		f.handleRequest(env)
	}
}

func (f *Server) handleRequest(env clientEnv) {
	switch env.RPCTag {
	case "server.getConfig":
		f.ConfigCalls.Add(1)
		f.Write(ExitSuccess(idString(env.ID), map[string]any{"ok": true, "environment": f.descriptor()}))
	case "orchestration.dispatchCommand":
		f.handleDispatch(env)
	case "orchestration.subscribeThread":
		var in map[string]any
		if err := json.Unmarshal(env.Payload, &in); err != nil {
			f.t.Errorf("fake: undecodable subscribe payload: %v", err)
		}
		select {
		case f.SubscribeIn <- in:
		default:
		}
		f.Subscribed <- idString(env.ID)
	case "orchestration.subscribeShell":
		f.Write(Chunk(idString(env.ID), map[string]any{"kind": "snapshot", "snapshot": map[string]any{
			"snapshotSequence": 1,
			"projects": []map[string]any{
				{"id": "proj-live", "title": "My App", "workspaceRoot": "/home/me/app", "deletedAt": nil},
				{"id": "proj-gone", "title": "Old", "deletedAt": "2026-01-01T00:00:00Z"},
				{"id": "proj-two", "title": "Second", "workspaceRoot": "/home/me/second"},
			},
		}}))
	default:
		f.t.Errorf("fake: unexpected RPC %q", env.RPCTag)
	}
}

func (f *Server) handleDispatch(env clientEnv) {
	var cmd map[string]any
	if err := json.Unmarshal(env.Payload, &cmd); err != nil {
		f.t.Errorf("fake: undecodable dispatch payload: %v", err)
		return
	}
	if f.DispatchCause != nil {
		f.Write(map[string]any{"_tag": "Exit", "requestId": idString(env.ID),
			"exit": map[string]any{"_tag": "Failure", "cause": f.DispatchCause}})
		return
	}
	f.Write(ExitSuccess(idString(env.ID), map[string]any{"sequence": 1}))
	f.Dispatched <- cmd
}

// Write pushes one frame to the connected client.
func (f *Server) Write(v any) {
	f.connMu.Lock()
	defer f.connMu.Unlock()
	if err := wsjson.Write(context.Background(), f.conn, v); err != nil {
		f.t.Logf("fake: write failed: %v", err)
	}
}

// Drop kills the current socket without a close frame, the way a tunnel or network blip ends it.
func (f *Server) Drop() {
	f.connMu.Lock()
	defer f.connMu.Unlock()
	_ = f.conn.CloseNow()
}

// WriteRaw pushes arbitrary bytes as one text frame, for malformed-frame tests.
func (f *Server) WriteRaw(data string) {
	f.connMu.Lock()
	defer f.connMu.Unlock()
	if err := f.conn.Write(context.Background(), websocket.MessageText, []byte(data)); err != nil {
		f.t.Logf("fake: raw write failed: %v", err)
	}
}

func idString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// ExitSuccess is the Exit frame that ends requestID with value.
func ExitSuccess(requestID string, value any) map[string]any {
	return map[string]any{
		"_tag": "Exit", "requestId": requestID,
		"exit": map[string]any{"_tag": "Success", "value": value},
	}
}

// Chunk is one Chunk frame of requestID's stream carrying items.
func Chunk(requestID string, items ...any) map[string]any {
	return map[string]any{"_tag": "Chunk", "requestId": requestID, "values": items}
}

// WaitFor receives one value from ch, failing t after five seconds.
func WaitFor[T any](t testing.TB, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s", what)
		panic("unreachable")
	}
}

package t3client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/otal-labs/nexul/internal/harness"
)

// The real T3 server is never reachable in CI (ticket 12's brief); fakeT3 is
// an httptest server speaking recorded frames per ticket 03's shapes: the
// wsTicket mint, the /ws Effect RPC socket, and the version probe.
type fakeT3 struct {
	t           *testing.T
	srv         *httptest.Server
	bearer      string
	ticket      string
	ticketField string

	configCalls atomic.Int32
	subscribed  chan string         // requestId of each subscribeThread stream
	acks        chan string         // requestId of each Ack frame received
	dispatched  chan map[string]any // payload of each dispatchCommand (auto-acked)

	connMu sync.Mutex
	conn   *websocket.Conn
}

type clientEnv struct {
	Tag     string          `json:"_tag"`
	ID      json.RawMessage `json:"id"`
	RPCTag  string          `json:"tag"`
	Payload json.RawMessage `json:"payload"`
}

func newFakeT3(t *testing.T) *fakeT3 {
	f := &fakeT3{
		t:           t,
		bearer:      "bearer-token",
		ticket:      "ws-ticket-1",
		ticketField: "ticket",
		subscribed:  make(chan string, 4),
		acks:        make(chan string, 16),
		dispatched:  make(chan map[string]any, 16),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(wsTicketPath, f.handleTicket)
	mux.HandleFunc(wsPath, f.handleWS)
	mux.HandleFunc("/.well-known/t3/environment", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"serverVersion": "0.0.34"})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeT3) session() harness.Session {
	return harness.Session{ServerURL: f.srv.URL, BearerToken: f.bearer}
}

func (f *fakeT3) connect(t *testing.T, ctx context.Context) *Client {
	c, err := Connect(ctx, f.session(), Options{HTTPClient: f.srv.Client(), RPCTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func (f *fakeT3) handleTicket(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+f.bearer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{f.ticketField: f.ticket})
}

func (f *fakeT3) handleWS(w http.ResponseWriter, r *http.Request) {
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
			case f.acks <- idString(env.ID):
			default:
			}
			continue
		}
		if env.Tag != "Request" {
			continue // Interrupt / Pong / anything else: fine to ignore in the fake
		}
		switch env.RPCTag {
		case "server.getConfig":
			f.configCalls.Add(1)
			f.write(exitSuccess(idString(env.ID), map[string]any{"ok": true}))
		case "orchestration.dispatchCommand":
			var cmd map[string]any
			if err := json.Unmarshal(env.Payload, &cmd); err != nil {
				f.t.Errorf("fake: undecodable dispatch payload: %v", err)
				continue
			}
			f.write(exitSuccess(idString(env.ID), map[string]any{"sequence": 1}))
			f.dispatched <- cmd
		case "orchestration.subscribeThread":
			f.subscribed <- idString(env.ID)
		case "orchestration.subscribeShell":
			f.write(chunk(idString(env.ID), map[string]any{"kind": "snapshot", "snapshot": map[string]any{
				"snapshotSequence": 1,
				"projects": []map[string]any{
					{"id": "proj-live", "title": "My App", "deletedAt": nil},
					{"id": "proj-gone", "title": "Old", "deletedAt": "2026-01-01T00:00:00Z"},
					{"id": "proj-two", "title": "Second"},
				},
			}}))
		default:
			f.t.Errorf("fake: unexpected RPC %q", env.RPCTag)
		}
	}
}

func (f *fakeT3) write(v any) {
	f.connMu.Lock()
	defer f.connMu.Unlock()
	if err := wsjson.Write(context.Background(), f.conn, v); err != nil {
		f.t.Logf("fake: write failed: %v", err)
	}
}

// writeRaw pushes arbitrary bytes as one text frame (malformed-frame tests).
func (f *fakeT3) writeRaw(data string) {
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

// Frame builders per ticket 03's envelope/stream-item shapes.

func exitSuccess(requestID string, value any) map[string]any {
	return map[string]any{
		"_tag": "Exit", "requestId": requestID,
		"exit": map[string]any{"_tag": "Success", "value": value},
	}
}

func chunk(requestID string, items ...any) map[string]any {
	return map[string]any{"_tag": "Chunk", "requestId": requestID, "values": items}
}

func eventItem(eventType string, payload any) map[string]any {
	return map[string]any{"kind": "event", "event": map[string]any{"type": eventType, "payload": payload}}
}

func messageSent(threadID, messageID, role, text string, streaming bool) map[string]any {
	return eventItem("thread.message-sent", map[string]any{
		"threadId": threadID, "messageId": messageID, "role": role,
		"text": text, "streaming": streaming, "turnId": "turn-1",
	})
}

func toolActivity(threadID, kind, summary string) map[string]any {
	return eventItem("thread.activity-appended", map[string]any{
		"threadId": threadID,
		"activity": map[string]any{"id": "act-1", "tone": "tool", "kind": kind, "summary": summary, "payload": map[string]any{}},
	})
}

func sessionSet(threadID, status string, activeTurnID, lastError any) map[string]any {
	return eventItem("thread.session-set", map[string]any{
		"threadId": threadID,
		"session": map[string]any{
			"threadId": threadID, "status": status, "providerName": "claude",
			"activeTurnId": activeTurnID, "lastError": lastError,
		},
	})
}

func waitFor[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s", what)
		panic("unreachable")
	}
}

// collect drains a subscription to completion, splitting updates by kind.
func collect(t *testing.T, sub *Subscription) ([]MessageSnapshot, []harness.Activity, []ApprovalRequest, *TurnResult) {
	t.Helper()
	var snaps []MessageSnapshot
	var activities []harness.Activity
	var approvals []ApprovalRequest
	var terminal *TurnResult
	for u := range sub.Updates() {
		if u.Snapshot != nil {
			snaps = append(snaps, *u.Snapshot)
		}
		if u.Activity != nil {
			activities = append(activities, *u.Activity)
		}
		if u.Approval != nil {
			approvals = append(approvals, *u.Approval)
		}
		if u.Terminal != nil {
			terminal = u.Terminal
		}
	}
	return snaps, activities, approvals, terminal
}

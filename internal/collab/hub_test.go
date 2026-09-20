package collab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// serveHub wraps a hub in an httptest server that injects the acting actor and mounts the docID wildcard exactly like production (RequireWS + withIdentity + GET /ws/collab/{docID}).
func serveHub(t *testing.T, hub *Hub, actorID string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle("GET /ws/collab/{docID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: actorID}))
		hub.ServeHTTP(w, r)
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func dial(t *testing.T, srv *httptest.Server, path string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, srv.URL+path, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	return conn
}

// recv decodes one server frame within a bound.
func recv(t *testing.T, conn *websocket.Conn) ServerMsg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var m ServerMsg
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// recvRaw reads one server frame's undecoded bytes so tests can assert the literal wire shape, since Go's json.Unmarshal matches untagged field names case-insensitively and would hide a wrong-case bug a browser's strict-cased JS reader would not tolerate.
func recvRaw(t *testing.T, conn *websocket.Conn) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	return data
}

func sendJSON(t *testing.T, conn *websocket.Conn, m ClientMsg) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, data))
}

// dialExpectError asserts a join attempt is refused with the given status: coder/websocket returns the response alongside the error on non-101.
func dialExpectError(t *testing.T, srv *httptest.Server, path string, wantStatus int) {
	t.Helper()
	_, resp, err := websocket.Dial(context.Background(), srv.URL+path, nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, wantStatus, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHubRejectsJoinWithoutPermission(t *testing.T) {
	hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: false, err: nil}, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	dialExpectError(t, srv, "/ws/collab/doc-1?mode=edit", http.StatusForbidden)
}

func TestHubRejectsUnknownMode(t *testing.T) {
	hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: true}, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	// The ServeMux pattern requires a non-empty {docID}, so the "missing doc id" guard is unreachable through routing; the unknown-mode guard is reachable and enforced before the join gate.
	dialExpectError(t, srv, "/ws/collab/doc-1?mode=admin", http.StatusBadRequest)
}

// TestHubRelayAndCommit drives two real sessions: alice's update reaches bob (with its stored seq), alice's commit writes the canonical body, and a late joiner replays the snapshot instead of the trimmed increments.
func TestHubRelayAndCommit(t *testing.T) {
	store := newFakeStore()
	writer := &fakeWriter{}
	hub := NewHub(testLogger(), store, &fakeAccess{allowed: true}, writer)
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	bob := dial(t, srv, "/ws/collab/doc-1?mode=view")
	recv(t, alice) // init (empty state)
	recv(t, bob)   // init

	sendJSON(t, alice, ClientMsg{Type: msgHello, ClientID: 101})
	sendJSON(t, alice, ClientMsg{Type: msgUpdate, Update: "YWxpY2UtdXBkYXRl"})

	relay := recv(t, bob)
	assert.Equal(t, msgUpdate, relay.Type)
	assert.Equal(t, "alice", relay.From)
	assert.Equal(t, "YWxpY2UtdXBkYXRl", relay.Payload)
	assert.Equal(t, int64(1), relay.Seq)

	sendJSON(t, alice, ClientMsg{Type: msgCommit, Update: "YWxpY2Utc25hcA==", BaseSeq: 1, Title: "T", Body: `{"type":"doc"}`})
	committed := recv(t, bob)
	assert.Equal(t, msgCommit, committed.Type)

	require.Eventually(t, func() bool { return writer.count() == 1 }, time.Second, 10*time.Millisecond)
	require.Equal(t, `{"type":"doc"}`, writer.calls[0].body)

	// A title-only edit commits with an unchanged body — it must still write
	// (the title lives outside the Y.Doc, so the body-based dedupe alone
	// silently dropped renames), while a fully identical commit stays deduped.
	sendJSON(t, alice, ClientMsg{Type: msgCommit, Update: "YWxpY2Utc25hcA==", BaseSeq: 2, Title: "Renamed", Body: `{"type":"doc"}`})
	renamed := recv(t, bob)
	assert.Equal(t, "Renamed", renamed.Title, "the commit relay must carry the rename so peers update their title inputs")
	require.Eventually(t, func() bool { return writer.count() == 2 }, time.Second, 10*time.Millisecond)
	require.Equal(t, "Renamed", writer.calls[1].title)

	sendJSON(t, alice, ClientMsg{Type: msgCommit, Update: "YWxpY2Utc25hcA==", BaseSeq: 3, Title: "Renamed", Body: `{"type":"doc"}`})
	recv(t, bob)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, writer.count(), "an identical title+body commit must stay deduped")

	carol := dial(t, srv, "/ws/collab/doc-1?mode=view")
	init := recv(t, carol)
	require.NotNil(t, init.Snapshot)
	assert.Equal(t, "YWxpY2Utc25hcA==", init.Snapshot.Payload)
	assert.Empty(t, init.Updates)
}

// TestHubInitSnapshotWireShapeMatchesFrontendContract pins the ws-25 conflict-banner bug scenario (commit persists a snapshot, a fresh session replays it) and inspects the raw wire bytes because the browser (protocol.ts) reads init.snapshot.payload as a case-sensitive JS property with no server-side renaming, so a round-trip through ServerMsg (as recv does) would not catch a wrong-case regression since Go's json.Unmarshal matches untagged fields case-insensitively.
func TestHubInitSnapshotWireShapeMatchesFrontendContract(t *testing.T) {
	store := newFakeStore()
	hub := NewHub(testLogger(), store, &fakeAccess{allowed: true}, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	recv(t, alice) // init (empty state, first-ever open)

	sendJSON(t, alice, ClientMsg{Type: msgHello, ClientID: 101})
	sendJSON(t, alice, ClientMsg{
		Type: msgCommit, Update: "YWxpY2Utc25hcHNob3Q=", BaseSeq: 0, Title: "T", Body: `{"type":"doc"}`,
	})
	require.NoError(t, alice.Close(websocket.StatusNormalClosure, ""))

	// Navigate back: a brand new session for the same doc replays the
	// snapshot just committed.
	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		_, ok := hub.sessions["doc-1"]
		return !ok
	}, time.Second, 10*time.Millisecond)
	bob := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	raw := recvRaw(t, bob)

	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &wire))
	var snapshot map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(wire["snapshot"], &snapshot))

	var payload string
	require.NoError(t, json.Unmarshal(snapshot["payload"], &payload),
		"init.snapshot.payload must exist in lowercase on the wire (frontend contract, protocol.ts): got raw snapshot %s", snapshot)
	assert.Equal(t, "YWxpY2Utc25hcHNob3Q=", payload)
}

func TestHubPresenceRosterAndLeave(t *testing.T) {
	hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: true}, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	recv(t, alice) // init

	// bob joins first so alice's presence RELAY is observable: the relayed frame is the ordering point that proves the hub stored the presence before any late joiner's roster snapshot.
	bob := dial(t, srv, "/ws/collab/doc-1?mode=view")
	recv(t, bob) // init

	sendJSON(t, alice, ClientMsg{Type: msgHello, ClientID: 101})
	sendJSON(t, alice, ClientMsg{Type: msgPresence, ClientID: 101, Payload: "YXdhcmVuZXNz"})

	relayed := recv(t, bob)
	assert.Equal(t, msgPresence, relayed.Type)
	assert.Equal(t, 101, relayed.ClientID)

	carol := dial(t, srv, "/ws/collab/doc-1?mode=view")
	init := recv(t, carol)
	require.Len(t, init.Presence, 1)
	assert.Equal(t, 101, init.Presence[0].ClientID)

	// Closing alice's socket must broadcast leave with her client id.
	require.NoError(t, alice.Close(websocket.StatusNormalClosure, ""))
	leave := recv(t, carol)
	assert.Equal(t, msgLeave, leave.Type)
	assert.Equal(t, 101, leave.ClientID)
}

// switchableAccess lets a test flip the join gate while connections are live (mutex-guarded, so the race detector stays quiet).
type switchableAccess struct {
	mu      sync.Mutex
	allowed bool
}

func (s *switchableAccess) Can(context.Context, string, string, permissions.Action) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.allowed, nil
}

func (s *switchableAccess) set(allowed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowed = allowed
}

func TestHubDeniesSecondUserWithoutEditBit(t *testing.T) {
	access := &switchableAccess{allowed: true}
	hub := NewHub(testLogger(), newFakeStore(), access, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	recv(t, alice) // init

	// The access checker is shared; flip it to deny so the second join must be refused at the gate, not by the docs use-cases.
	access.set(false)

	dialExpectError(t, srv, "/ws/collab/doc-1?mode=edit", http.StatusForbidden)
}

func TestHubDispatchGuards(t *testing.T) {
	hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: true}, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	bob := dial(t, srv, "/ws/collab/doc-1?mode=view")
	recv(t, alice) // init
	recv(t, bob)   // init

	// Guarded frames (presence before hello, unknown types) are dropped: the first frame bob sees after init must be the presence relay of the post-hello message, not any of the guarded ones.
	sendJSON(t, alice, ClientMsg{Type: msgPresence, ClientID: 99, Payload: "YXdhcmVuZXNz"})
	sendJSON(t, alice, ClientMsg{Type: "ping"})
	sendJSON(t, alice, ClientMsg{Type: msgHello, ClientID: 99})
	sendJSON(t, alice, ClientMsg{Type: msgPresence, ClientID: 99, Payload: "YXdhcmVuZXNz"})
	got := recv(t, bob)
	assert.Equal(t, msgPresence, got.Type)
	assert.Equal(t, 99, got.ClientID)
}

func TestNewHubDefaultsLogger(t *testing.T) {
	hub := NewHub(nil, newFakeStore(), &fakeAccess{allowed: true}, &fakeWriter{})
	require.NotNil(t, hub)
	require.NotNil(t, hub.log)
}

func TestHubRoomReclaimedAfterLastLeave(t *testing.T) {
	hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: true}, &fakeWriter{})
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	recv(t, alice) // init
	require.NoError(t, alice.Close(websocket.StatusNormalClosure, ""))

	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		_, ok := hub.sessions["doc-1"]
		return !ok
	}, time.Second, 10*time.Millisecond)
}

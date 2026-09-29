package live

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// serveHub wraps a hub in an httptest server. The handler blocks in the hub's
// read loop until the client disconnects, so Close is how tests end it.
func serveHub(t *testing.T, hub *Hub) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// dialClient connects a browser-style client and waits until the hub has
// registered it, so a subsequent Publish is guaranteed to arrive.
func dialClient(t *testing.T, hub *Hub, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):], nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		hub.mu.Lock()
		n := len(hub.clients)
		hub.mu.Unlock()
		if n == 1 {
			return conn
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("hub never registered the client")
	return nil
}

func readFrame(t *testing.T, conn *websocket.Conn) Frame {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var f Frame
	require.NoError(t, json.Unmarshal(data, &f))
	return f
}

func TestHub_PublishFansOutToConnectedClients(t *testing.T) {
	hub := New(testLogger())
	srv := serveHub(t, hub)
	conn := dialClient(t, hub, srv)

	require.NoError(t, hub.Publish(context.Background(), "runner.connected", map[string]any{"runner_id": "r-1"}))
	f := readFrame(t, conn)
	assert.Equal(t, "runner.connected", f.Topic)
	assert.Equal(t, "event", f.Type)
	require.NotNil(t, f.Payload)
}

func TestHub_PublishWithNoClientsIsNoOp(t *testing.T) {
	hub := New(testLogger())
	require.NoError(t, hub.Publish(context.Background(), "runner.connected", map[string]string{"runner_id": "r-1"}))
}

func TestHub_PublishUnencodablePayloadErrors(t *testing.T) {
	hub := New(testLogger())
	err := hub.Publish(context.Background(), "runner.connected", make(chan int))
	require.Error(t, err)
}

// TestHub_AudienceKeepsAFrameFromSocketsItRefuses has two people connected and an audience that lets one of them
// read the frame: only that socket receives it, and the next frame the other may read still arrives.
func TestHub_AudienceKeepsAFrameFromSocketsItRefuses(t *testing.T) {
	hub := New(testLogger())
	hub.SetAudience(func(ctx context.Context, topic string, _ any) bool {
		actor, _ := identity.ActorFromCtx(ctx)
		return topic == "public" || actor.ID == "alice"
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := identity.WithActor(r.Context(), identity.Actor{ID: r.URL.Query().Get("user")})
		hub.ServeHTTP(w, r.WithContext(ctx))
	}))
	t.Cleanup(srv.Close)
	dial := func(user string) *websocket.Conn {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)
		conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):]+"?user="+user, nil)
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.CloseNow() })
		return conn
	}
	alice, bob := dial("alice"), dial("bob")
	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		return len(hub.clients) == 2
	}, 3*time.Second, 5*time.Millisecond)

	require.NoError(t, hub.Publish(context.Background(), "private", map[string]string{"title": "secret"}))
	require.NoError(t, hub.Publish(context.Background(), "public", map[string]string{"id": "1"}))

	assert.Equal(t, "private", readFrame(t, alice).Topic)
	assert.Equal(t, "public", readFrame(t, alice).Topic)
	assert.Equal(t, "public", readFrame(t, bob).Topic, "bob's first frame is the one he may read")
}

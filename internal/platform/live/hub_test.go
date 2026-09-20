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

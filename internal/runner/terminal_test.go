package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPtyTerminal_RunsTheShellAndReportsItsExit drives the default shell on a real pseudo-terminal through a WebSocket,
// as the server's stream does: typed keys run, a resize reaches the shell, and its exit comes back as a text frame.
func TestPtyTerminal_RunsTheShellAndReportsItsExit(t *testing.T) {
	term, err := startPty(Frame{Cols: 80, Rows: 24})
	require.NoError(t, err)
	defer term.close()

	relayed := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.CloseNow() }()
		term.relay(r.Context(), ws)
		close(relayed)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()

	require.NoError(t, conn.Write(ctx, websocket.MessageText, []byte(`{"cols":100,"rows":30}`)))
	line, exit := "echo nexul-$((40+2)) $(tput cols)\r", "exit 7\r"
	want := "nexul-42 100"
	if runtime.GOOS == "windows" {
		line, exit, want = "Write-Output \"nexul-$(40+2) $($Host.UI.RawUI.WindowSize.Width)\"\r", "exit 7\r", "nexul-42 100"
	}
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte(line)))
	var out strings.Builder
	for !strings.Contains(out.String(), want) {
		_, p, err := conn.Read(ctx)
		require.NoError(t, err, "read so far: %q", out.String())
		out.Write(p)
	}
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte(exit)))
	for {
		typ, p, err := conn.Read(ctx)
		require.NoError(t, err, "read so far: %q", out.String())
		if typ == websocket.MessageText {
			assert.JSONEq(t, `{"exit":7}`, string(p))
			break
		}
	}
	<-relayed
}

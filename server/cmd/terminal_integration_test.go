package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dialTerminal(ctx context.Context, srv *httptest.Server, jobID, token string) (*websocket.Conn, *http.Response, error) {
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/pairing/terminals/"+jobID+"?token="+url.QueryEscape(token), nil)
}

// readTerminalUntil reads the terminal's output until it contains want, and returns everything read.
func readTerminalUntil(t *testing.T, ctx context.Context, conn *websocket.Conn, want string) string {
	t.Helper()
	var out strings.Builder
	for !strings.Contains(out.String(), want) {
		typ, p, err := conn.Read(ctx)
		require.NoError(t, err, "read so far: %q", out.String())
		if typ == websocket.MessageBinary {
			out.Write(p)
		}
	}
	return out.String()
}

// TestIntegration_TerminalRunsAShellOnAComputer: alice opens a terminal on her computer, types into a real shell on a
// pseudo-terminal through its runner, and its exit is recorded with what it printed; nobody else can attach to it.
func TestIntegration_TerminalRunsAShellOnAComputer(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("the terminal's shell is bash")
	}
	c, _ := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, enrollToken := addComputer(t, c, c.alice, "")
	startShellRunner(t, c, srv, id, enrollToken, true)

	rec := callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/"+id+"/terminals", c.owner, `{"cols":80,"rows":24}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, "the workspace Owner cannot open one on alice's computer")
	rec = callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/"+id+"/terminals", c.alice, `{"cols":80,"rows":24}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var job struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &job))

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	_, resp, err := dialTerminal(ctx, srv, job.ID, c.owner)
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "nobody but alice attaches")

	conn, _, err := dialTerminal(ctx, srv, job.ID, c.alice)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()
	require.NoError(t, conn.Write(ctx, websocket.MessageText, []byte(`{"cols":100,"rows":30}`)))
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte("echo nexul-$((40+2)) $(tput cols)\r")))
	readTerminalUntil(t, ctx, conn, "nexul-42 100")
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte("stty -echo; wc -c\r")))
	paste := strings.Repeat(strings.Repeat("a", 999)+"\n", 70) + "\x04"
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte(paste)), "a paste over the default 32 KB message limit")
	readTerminalUntil(t, ctx, conn, "70000")
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte("stty echo\r")))
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte("exit 7\r")))
	for {
		typ, p, err := conn.Read(ctx)
		require.NoError(t, err)
		if typ == websocket.MessageText {
			assert.JSONEq(t, `{"exit":7}`, string(p))
			break
		}
	}

	require.Eventually(t, func() bool {
		rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/commands/"+job.ID, c.alice, "")
		return strings.Contains(rec.Body.String(), `"status":"failed"`)
	}, 10*time.Second, 20*time.Millisecond)
	rec = callAPI(t, c.h, http.MethodGet, "/api/pairing/commands/"+job.ID, c.alice, "")
	assert.Contains(t, rec.Body.String(), `"adapter":"terminal"`)
	assert.Contains(t, rec.Body.String(), `"exit_code":7`)
	assert.Contains(t, rec.Body.String(), "nexul-42 100")
}

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/runner"
)

// startShellRunner enrolls alice's computer and runs a runner new enough for commands, with the computer's own opt-in.
func startShellRunner(t *testing.T, c privacyCast, srv *httptest.Server, computerID, enrollToken string, shellEnabled bool) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"token":"`+enrollToken+`","machine":"alice-laptop","os":"linux","arch":"amd64"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var enrolled struct {
		Name       string `json:"name"`
		Credential string `json:"credential"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &enrolled))
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	personal := runner.NewClient(runner.ClientConfig{
		URL: base + "/ws/runner", StreamURL: base + "/api/runners/streams", Credential: enrolled.Credential, Name: enrolled.Name,
		Version: "v0.3.101-beta", Logger: logger, Executor: runner.NewShellExecutor(nil, runner.ExecutorConfig{}, logger),
		Personal: true, ShellEnabled: shellEnabled, T3Home: t.TempDir(),
	})
	ctx, stop := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- personal.Run(ctx) }()
	t.Cleanup(func() {
		stop()
		<-stopped
	})
	require.Eventually(t, func() bool {
		rn, err := c.svc.computers.ComputerRunner(t.Context(), computerID)
		return err == nil && rn.Connected
	}, 10*time.Second, 20*time.Millisecond)
}

func shellCast(t *testing.T, shellEnabled bool) (privacyCast, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the commands are POSIX shell")
	}
	c, _ := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, enrollToken := addComputer(t, c, c.alice, "")
	startShellRunner(t, c, srv, id, enrollToken, shellEnabled)
	return c, id
}

func TestIntegration_CommandRunsOnAComputerThroughItsRunner(t *testing.T) {
	t.Parallel()
	c, id := shellCast(t, true)

	got := callToolResult(t, c.h, c.alice, "command_run", `{"computer_id":"`+id+`","command":"echo hello; echo oops >&2; exit 3"}`)
	assert.Contains(t, got, `\"status\":\"failed\"`)
	assert.Contains(t, got, `\"exit_code\":3`)
	assert.Contains(t, got, "hello")
	assert.Contains(t, got, "oops")

	start := time.Now()
	got = callToolResult(t, c.h, c.alice, "command_run", `{"computer_id":"`+id+`","command":"sleep 30","timeout_seconds":1}`)
	assert.Contains(t, got, `\"status\":\"timed_out\"`)
	assert.Less(t, time.Since(start), 10*time.Second)

	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+id+"/commands", c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"total":2`)
	assert.Contains(t, rec.Body.String(), `"adapter":"mcp"`)
}

func TestShellJobs_OnlyTheOwnerStartsOrReadsThemOverHTTPAndMCP(t *testing.T) {
	t.Parallel()
	c, id := shellCast(t, true)
	rec := callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/"+id+"/commands", c.alice, `{"command":"true"}`)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	var job struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &job))

	for name, token := range map[string]string{"a workspace Owner": c.owner, "a role holding every bit": c.everyBits} {
		t.Run(name, func(t *testing.T) {
			for _, r := range []struct{ method, path, body string }{
				{http.MethodPost, "/api/pairing/computers/" + id + "/commands", `{"command":"id"}`},
				{http.MethodGet, "/api/pairing/computers/" + id + "/commands", ""},
				{http.MethodGet, "/api/pairing/commands/" + job.ID, ""},
				{http.MethodPost, "/api/pairing/commands/" + job.ID + "/cancel", ""},
			} {
				rec := callAPI(t, c.h, r.method, r.path, token, r.body)
				assert.Equal(t, http.StatusNotFound, rec.Code, r.method+" "+r.path)
			}
			got := callToolResult(t, c.h, token, "command_run", `{"computer_id":"`+id+`","command":"id"}`)
			assert.Contains(t, got, "not found")
			assert.Contains(t, callToolResult(t, c.h, token, "computer_list", `{"id":"`+id+`","commands":{}}`), "not one of yours")
		})
	}
}

func TestShellJobs_RefusedWithoutTheComputersOwnOptIn(t *testing.T) {
	t.Parallel()
	c, id := shellCast(t, false)

	got := callToolResult(t, c.h, c.alice, "command_run", `{"computer_id":"`+id+`","command":"echo should-not-run"}`)

	assert.Contains(t, got, `\"status\":\"refused\"`)
	assert.Contains(t, got, "nexul shell on")
	assert.NotContains(t, got, "should-not-run\\n")
	rec := callAPI(t, c.h, http.MethodGet, "/api/pairing/computers/"+id+"/commands", c.alice, "")
	assert.Contains(t, rec.Body.String(), `"status":"refused"`, "the refusal is on the record")
}

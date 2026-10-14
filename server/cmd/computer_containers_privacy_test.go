package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/runner"
)

// connectComputerRunner enrolls computerID's personal runner and connects it as a fake that lists one container and
// answers each logs_request with one line, counting every container request that reaches the computer.
func connectComputerRunner(t *testing.T, c privacyCast, srv *httptest.Server, computerID string) *atomic.Int32 {
	t.Helper()
	ctx := t.Context()
	now := time.Now().UTC()
	_, codeHash, err := hostcred.MintCode()
	require.NoError(t, err)
	require.NoError(t, c.store.Runners.CreateEnrollment(ctx, &runner.EnrollmentCode{ID: "id-" + codeHash, CodeHash: codeHash, Name: "computer-ab12cd34", OwnerUserID: "u-alice", ComputerID: computerID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	credential, credentialHash, err := hostcred.MintCredential("nxr_")
	require.NoError(t, err)
	require.NoError(t, c.store.Runners.Enroll(ctx, codeHash, &runner.Runner{ID: "r-laptop", Name: "computer-ab12cd34", OwnerUserID: "u-alice", ComputerID: computerID, LastSeen: now, CreatedAt: now}, credentialHash, now, nil))

	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/runner", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": {"Bearer " + credential}}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	requests := &atomic.Int32{}
	go func() {
		for {
			var f runner.Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			switch f.Type {
			case runner.FrameDiscover:
				requests.Add(1)
				_ = wsjson.Write(ctx, conn, runner.Frame{Type: runner.FrameDiscoverResult, ID: f.ID, Containers: []runner.DiscoveredContainer{{Name: "shop-web-1", Image: "shop:1", Status: "running"}}})
			case runner.FrameLogsRequest:
				requests.Add(1)
				_ = wsjson.Write(ctx, conn, runner.Frame{Type: runner.FrameLogsChunk, ID: f.ID, Lines: []runner.ContainerLogLine{{TS: "2026-10-11T10:00:00Z", Stream: runner.LogStreamStdout, Line: "listening on :8080"}}})
				if !f.Follow {
					_ = wsjson.Write(ctx, conn, runner.Frame{Type: runner.FrameLogsEnd, ID: f.ID})
				}
			}
		}
	}()
	require.Eventually(t, func() bool {
		rn, err := c.store.Runners.GetByID(ctx, "r-laptop")
		return err == nil && rn.Connected
	}, 5*time.Second, 10*time.Millisecond)
	return requests
}

func reportDocker(t *testing.T, c privacyCast, computerID string, available bool) {
	t.Helper()
	report, err := json.Marshal(map[string]any{
		"runner_id": "r-laptop", "computer_id": computerID, "user_id": "u-alice", "members_only": true,
		"facts": map[string]any{"hostname": "alice-laptop", "t3": map[string]any{"state": "not_running"}, "docker": map[string]any{"available": available}},
	})
	require.NoError(t, err)
	require.NoError(t, c.svc.pairingSvc.HandleFactsReported(t.Context(), eventbus.Event{Topic: "runner.facts_reported", Payload: report}))
}

// dialContainerLogs opens the live tail socket as token's holder.
func dialContainerLogs(ctx context.Context, srv *httptest.Server, computerID, token string) (*websocket.Conn, *http.Response, error) {
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/pairing/computers/"+computerID+"/containers/shop-web-1/logs?tail=20&token="+url.QueryEscape(token), nil)
}

// TestComputerContainers_OnlyTheOwnerSeesThem: alice lists the containers on her computer and reads their logs over
// HTTP, MCP and the logs socket; the workspace Owner and a role holding every bit get not found on every path, and none
// of their calls reaches her computer. Once her runner reports no Docker, nothing reaches it for her either.
func TestComputerContainers_OnlyTheOwnerSeesThem(t *testing.T) {
	c, _ := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, _ := addComputer(t, c, c.alice, "Alice laptop")
	requests := connectComputerRunner(t, c, srv, id)
	reportDocker(t, c, id, true)
	containers := "/api/pairing/computers/" + id + "/containers"
	logs := containers + "/shop-web-1/logs?tail=5"
	logsTool := `{"id":"` + id + `","container_logs":{"container":"shop-web-1","tail":5}}`

	t.Run("the owner reads them on every path", func(t *testing.T) {
		rec := callAPI(t, c.h, http.MethodGet, containers, c.alice, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.JSONEq(t, `{"containers":[{"name":"shop-web-1","image":"shop:1","status":"running"}]}`, rec.Body.String())
		rec = callAPI(t, c.h, http.MethodGet, logs, c.alice, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "listening on :8080")
		assert.Contains(t, callTool(t, c.h, c.alice, "computer_list", logsTool), "listening on :8080")

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		viewer, _, err := dialContainerLogs(ctx, srv, id, c.alice)
		require.NoError(t, err)
		var msg struct {
			Lines []struct {
				Line string `json:"line"`
			} `json:"lines"`
		}
		require.NoError(t, wsjson.Read(ctx, viewer, &msg))
		require.Len(t, msg.Lines, 1)
		assert.Equal(t, "listening on :8080", msg.Lines[0].Line)
		_ = viewer.Close(websocket.StatusNormalClosure, "")
	})

	t.Run("nobody else, whatever they hold", func(t *testing.T) {
		before := requests.Load()
		for who, token := range map[string]string{"workspace Owner": c.owner, "every bit": c.everyBits} {
			for _, path := range []string{containers, logs} {
				rec := callAPI(t, c.h, http.MethodGet, path, token, "")
				assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s as %s", path, who)
				assert.NotContains(t, rec.Body.String(), "shop-web-1", "GET %s as %s", path, who)
			}
			body := callToolResult(t, c.h, token, "computer_list", logsTool)
			assert.Contains(t, body, "not found", "computer_list as %s", who)
			assert.NotContains(t, body, "listening", "computer_list as %s", who)

			_, resp, err := dialContainerLogs(t.Context(), srv, id, token)
			require.Error(t, err, "the logs socket as %s", who)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "the logs socket as %s", who)
		}
		assert.Equal(t, before, requests.Load(), "no call of theirs reached alice's computer")
	})

	t.Run("a computer without Docker takes no container request", func(t *testing.T) {
		reportDocker(t, c, id, false)
		before := requests.Load()
		for _, path := range []string{containers, logs} {
			rec := callAPI(t, c.h, http.MethodGet, path, c.alice, "")
			assert.Equal(t, http.StatusConflict, rec.Code, path)
		}
		_, resp, err := dialContainerLogs(t.Context(), srv, id, c.alice)
		require.Error(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		assert.NotContains(t, callTool(t, c.h, c.alice, "computer_list", `{"id":"`+id+`"}`), "shop-web-1", "and lists none")
		assert.Equal(t, before, requests.Load())
	})
}

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

// logsBus satisfies the runner handler's bus; runner lifecycle events are not what these tests are about.
type logsBus struct{ noopPublisher }

func (logsBus) Subscribe(context.Context, string, eventbus.Handler) error { return nil }

// fakeRunner is a runner connection driven by the test: it answers each logs_request with one line carrying
// the stack's secret, ends a snapshot there, and reports every logs_cancel it receives.
type fakeRunner struct {
	requests  chan runner.Frame
	cancelled chan string
}

func connectFakeRunner(t *testing.T, store *storage.Store, handler *runner.Handler) *fakeRunner {
	t.Helper()
	ctx := t.Context()
	now := time.Now().UTC()
	require.NoError(t, store.Machines.Create(ctx, &runner.Machine{ID: "machine-1", Name: "m1", FirstSeen: now, LastSeen: now}))
	_, codeHash, err := hostcred.MintCode()
	require.NoError(t, err)
	require.NoError(t, store.Runners.CreateEnrollment(ctx, &runner.EnrollmentCode{CodeHash: codeHash, Name: "logs", Machine: "m1", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	credential, credentialHash, err := hostcred.MintCredential("nxr_")
	require.NoError(t, err)
	require.NoError(t, store.Runners.Enroll(ctx, codeHash, &runner.Runner{ID: "runner-logs", Name: "logs", MachineID: "machine-1", LastSeen: now, CreatedAt: now}, credentialHash, now))

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": {"Bearer " + credential}}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	require.Eventually(t, func() bool { return len(handler.Runners()) == 1 }, 3*time.Second, 10*time.Millisecond)

	r := &fakeRunner{requests: make(chan runner.Frame, 8), cancelled: make(chan string, 8)}
	go func() {
		for {
			var f runner.Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			switch f.Type {
			case runner.FrameLogsRequest:
				r.requests <- f
				_ = wsjson.Write(ctx, conn, runner.Frame{Type: runner.FrameLogsChunk, ID: f.ID, Lines: []runner.ContainerLogLine{
					{TS: "2026-09-30T10:00:00Z", Stream: runner.LogStreamStderr, Line: "rejected key sk_live_abcdef"},
				}})
				if !f.Follow {
					_ = wsjson.Write(ctx, conn, runner.Frame{Type: runner.FrameLogsEnd, ID: f.ID})
				}
			case runner.FrameLogsCancel:
				r.cancelled <- f.ID
			}
		}
	}()
	return r
}

func asUser(userID string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: userID})))
	})
}

func TestIntegration_ContainerLogs(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-shop", ProjectID: "project-general", Name: "shop", Slug: "shop", Machine: "m1",
		Strategy: deploy.StrategyRun, Env: map[string]string{"API_KEY": "sk_live_abcdef"}, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Services.Create(ctx, &deploy.Container{ID: "ctr-web", StackID: "stack-shop", Name: "web", ContainerName: "shop-web-1", Status: deploy.ServiceStatusRunning}))

	runners := runner.NewHandler(runner.HandlerConfig{Bus: logsBus{}, Repo: f.store.Runners, Machines: f.store.Machines, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	f.svc.deploySvc.SetLogSource(deployLogSourceAdapter{handler: runners})
	fake := connectFakeRunner(t, f.store, runners)
	masked := deploy.ContainerLogLine{TS: "2026-09-30T10:00:00Z", Stream: "stderr", Line: "rejected key ••••"}

	t.Run("the snapshot route returns masked lines to a holder of stacks:logs, and nobody else", func(t *testing.T) {
		routes := deploy.NewHandler(f.svc.deploySvc).Routes()
		for user, want := range map[string]int{uOwner: http.StatusOK, uReader: http.StatusForbidden, uOutsider: http.StatusNotFound} {
			rec := httptest.NewRecorder()
			routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/stack-shop/services/web/logs?tail=5", nil).
				WithContext(identity.WithActor(ctx, identity.Actor{ID: user})))
			require.Equal(t, want, rec.Code, user)
			if want != http.StatusOK {
				continue
			}
			var body struct {
				Lines []deploy.ContainerLogLine `json:"lines"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, []deploy.ContainerLogLine{masked}, body.Lines)
			req := <-fake.requests
			assert.Equal(t, runner.Frame{Type: runner.FrameLogsRequest, ID: req.ID, Container: "shop-web-1", Tail: 5}, req)
		}
	})

	t.Run("a scoped token reads logs only with stacks:logs, whatever its creator holds", func(t *testing.T) {
		gateway := f.svc.integrationsSvc.RequireIntegration(func(h http.Handler) http.Handler { return h }, deploy.NewHandler(f.svc.deploySvc).Routes())
		for scope, want := range map[integrations.Scope]int{"stacks:read": http.StatusForbidden, "stacks:logs": http.StatusOK} {
			token, _, _, err := f.svc.integrationsSvc.Install(as(uOwner), uOwner, "logs-"+string(scope), integrations.TrustCommunity, "https://example.com/hook", []integrations.Scope{scope})
			require.NoError(t, err)
			for _, path := range []string{"/api/stacks/stack-shop/services/web/logs", "/api/services/stack-shop/services/web/logs"} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("Authorization", "Bearer "+token)
				rec := httptest.NewRecorder()
				gateway.ServeHTTP(rec, req)
				assert.Equal(t, want, rec.Code, "%s on %s", scope, path)
				if rec.Code == http.StatusOK {
					<-fake.requests
				}
			}
		}
	})

	t.Run("closing the live socket cancels the runner's stream", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.Handle("GET /ws/stacks/{id}/services/{name}/logs", asUser(uOwner, http.HandlerFunc(deploy.NewHandler(f.svc.deploySvc).LogsSocket)))
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		viewer, _, err := websocket.Dial(wctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/stacks/stack-shop/services/web/logs?tail=20", nil)
		require.NoError(t, err)
		var msg struct {
			Lines []deploy.ContainerLogLine `json:"lines"`
		}
		require.NoError(t, wsjson.Read(wctx, viewer, &msg))
		assert.Equal(t, []deploy.ContainerLogLine{masked}, msg.Lines)
		req := <-fake.requests
		assert.True(t, req.Follow)
		assert.Equal(t, 20, req.Tail)

		require.NoError(t, viewer.Close(websocket.StatusNormalClosure, ""))
		select {
		case id := <-fake.cancelled:
			assert.Equal(t, req.ID, id)
		case <-wctx.Done():
			t.Fatal("the runner never got logs_cancel after the viewer left")
		}
	})

	t.Run("the live socket refuses a reader without stacks:logs before upgrading", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.Handle("GET /ws/stacks/{id}/services/{name}/logs", asUser(uReader, http.HandlerFunc(deploy.NewHandler(f.svc.deploySvc).LogsSocket)))
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		_, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/stacks/stack-shop/services/web/logs", nil)
		require.Error(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

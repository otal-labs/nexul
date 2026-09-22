package runner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/release"
)

func newTestHandler(bus Bus, repo Repo, opts ...func(*HandlerConfig)) *Handler {
	cfg := HandlerConfig{
		Bus:               bus,
		Repo:              repo,
		Logger:            testLogger(),
		HeartbeatInterval: 50 * time.Millisecond,
		MissedHeartbeats:  3,
		WriteTimeout:      time.Second,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return NewHandler(cfg)
}

func newTestConn() *runnerConn {
	c := &runnerConn{id: "r-1"}
	c.touchHeartbeat()
	return c
}

func decodeEvent[T any](t *testing.T, ev eventbus.Event) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(ev.Payload, &out))
	return out
}

func TestHandler_dispatchFrame(t *testing.T) {
	tests := []struct {
		name      string
		frame     Frame
		wantTopic string
	}{
		{name: "heartbeat", frame: Frame{Type: FrameHeartbeat, RunnerID: "r-1", TS: 1700000000}, wantTopic: TopicRunnerHeartbeat},
		{name: "first build_progress is started", frame: Frame{Type: FrameBuildProgress, ID: "b1", Step: 1, Total: 2}, wantTopic: TopicDeployBuildStarted},
		{name: "later build_progress", frame: Frame{Type: FrameBuildProgress, ID: "b1", Step: 2, Total: 2}, wantTopic: TopicDeployBuildProgress},
		{name: "build_result", frame: Frame{Type: FrameBuildResult, ID: "b1", Status: BuildStatusFailed, Error: "boom"}, wantTopic: TopicDeployBuildCompleted},
		{name: "deploy_progress", frame: Frame{Type: FrameDeployProgress, ID: "d1", Phase: DeployPhasePulling}, wantTopic: TopicDeployDeployProgress},
		{name: "deploy_log", frame: Frame{Type: FrameDeployLog, ID: "d1", Phase: LogPhaseBuild, Log: "Step 1/4", TS: 1695379028112}, wantTopic: TopicDeployLog},
		{name: "deploy_result", frame: Frame{Type: FrameDeployResult, ID: "d1", Status: DeployStatusFailed, Error: "oom"}, wantTopic: TopicDeployStatusChanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newFakeBus()
			repo := newFakeRunnerRepo()
			h := newTestHandler(bus, repo)
			require.NoError(t, repo.Create(context.Background(), &Runner{ID: "r-1"}))

			require.NoError(t, h.dispatchFrame(context.Background(), newTestConn(), tt.frame))
			assert.Len(t, bus.topicEvents(tt.wantTopic), 1)
		})
	}
}

func TestHandler_dispatchFrame_Payloads(t *testing.T) {
	t.Run("build_progress step 1 carries started payload", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())
		f := Frame{Type: FrameBuildProgress, ID: "b1", Step: 1, Total: 3, Log: "go"}

		require.NoError(t, h.dispatchFrame(context.Background(), newTestConn(), f))
		evs := bus.topicEvents(TopicDeployBuildStarted)
		require.Len(t, evs, 1)
		got := decodeEvent[BuildStartedEvent](t, evs[0])
		assert.Equal(t, "b1", got.ID)
		assert.Equal(t, 3, got.Total)
		assert.Equal(t, "go", got.Log)
	})

	t.Run("build_result carries status and artifacts", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())
		f := Frame{Type: FrameBuildResult, ID: "b1", Status: BuildStatusSuccess, Artifacts: []string{"bin/api"}}

		require.NoError(t, h.dispatchFrame(context.Background(), newTestConn(), f))
		evs := bus.topicEvents(TopicDeployBuildCompleted)
		require.Len(t, evs, 1)
		got := decodeEvent[BuildCompletedEvent](t, evs[0])
		assert.Equal(t, BuildStatusSuccess, got.Status)
		assert.Equal(t, []string{"bin/api"}, got.Artifacts)
	})

	t.Run("deploy_log carries phase, lines and millisecond timestamp", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())
		f := Frame{Type: FrameDeployLog, ID: "d1", Phase: LogPhaseCheckout, Log: "clone org/app@main\nCloning into '/data/repo'...\n", TS: 1695379028112}

		require.NoError(t, h.dispatchFrame(context.Background(), newTestConn(), f))
		evs := bus.topicEvents(TopicDeployLog)
		require.Len(t, evs, 1)
		got := decodeEvent[DeployLogEvent](t, evs[0])
		assert.Equal(t, DeployLogEvent{ID: "d1", Phase: "checkout", Log: "clone org/app@main\nCloning into '/data/repo'...\n", TS: 1695379028112}, got)
		assert.JSONEq(t, `{"id":"d1","phase":"checkout","log":"clone org/app@main\nCloning into '/data/repo'...\n","ts":1695379028112}`, string(evs[0].Payload))
	})

	t.Run("deploy_result carries status and address", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())
		f := Frame{Type: FrameDeployResult, ID: "d1", Status: DeployStatusHealthy, Address: "172.18.0.4"}

		require.NoError(t, h.dispatchFrame(context.Background(), newTestConn(), f))
		evs := bus.topicEvents(TopicDeployStatusChanged)
		require.Len(t, evs, 1)
		got := decodeEvent[DeployStatusChangedEvent](t, evs[0])
		assert.Equal(t, DeployStatusHealthy, got.Status)
		assert.Equal(t, "172.18.0.4", got.Address)
	})
}

func TestHandler_dispatchFrame_HeartbeatUpdatesRepo(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	require.NoError(t, repo.Create(context.Background(), &Runner{ID: "r-1"}))
	h := newTestHandler(bus, repo)

	f := Frame{Type: FrameHeartbeat, RunnerID: "r-1", TS: 1700000000}
	require.NoError(t, h.dispatchFrame(context.Background(), newTestConn(), f))
	repo.mu.Lock()
	hb := repo.heartbeats
	repo.mu.Unlock()
	assert.Equal(t, 1, hb)
	evs := bus.topicEvents(TopicRunnerHeartbeat)
	require.Len(t, evs, 1)
	got := decodeEvent[RunnerHeartbeatEvent](t, evs[0])
	assert.Equal(t, "r-1", got.RunnerID)
	assert.Equal(t, int64(1700000000), got.TS)
}

func TestHandler_dispatchFrame_BusFailure(t *testing.T) {
	bus := newFakeBus()
	bus.publishErr = assert.AnError
	h := newTestHandler(bus, newFakeRunnerRepo())

	err := h.dispatchFrame(context.Background(), newTestConn(), Frame{Type: FrameBuildResult, ID: "b1", Status: BuildStatusSuccess})
	require.Error(t, err)
}

func TestHandler_handleRequest(t *testing.T) {
	t.Run("queues when no runner is connected", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())

		req := DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Repo: "org/app", Ref: "main"}
		require.NoError(t, h.handleRequest(context.Background(), eventbus.Event{Payload: mustJSON(t, req)}))
		require.Len(t, h.pending, 1)
		assert.Equal(t, "b1", h.pending[0].ID)
	})

	t.Run("malformed payload is fatal", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())

		err := h.handleRequest(context.Background(), eventbus.Event{Payload: []byte("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})

	t.Run("unknown kind is fatal", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())

		req := DeployRequestedEvent{ID: "b1", Kind: "wat"}
		err := h.handleRequest(context.Background(), eventbus.Event{Payload: mustJSON(t, req)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})

	t.Run("target with no matching runner queues instead of dispatching to a mismatched one", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())
		qa := &runnerConn{id: "r-qa", name: "qa"}
		h.mu.Lock()
		h.conns["r-qa"] = qa
		h.mu.Unlock()

		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", Target: "prod"}
		require.NoError(t, h.handleRequest(context.Background(), eventbus.Event{Payload: mustJSON(t, req)}))
		require.Len(t, h.Queue(), 1)
		assert.Equal(t, "prod", h.Queue()[0].Target)
		assert.Nil(t, qa.jobSnapshot())
	})
}

// A targeted job fits only a runner whose machine matches; an empty target fits any idle runner.
func TestHandler_idleConnected_TargetRouting(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string // runner id expected back, "" for nil
	}{
		{"target matches the only fitting idle runner", "prod", "r-prod"},
		{"target with no fitting runner returns nil", "prod", ""},
		{"empty target fits any idle runner", "", "r-qa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
			qa := &runnerConn{id: "r-qa", name: "qa", machine: "qa"}
			h.mu.Lock()
			h.conns["r-qa"] = qa
			h.mu.Unlock()
			if tt.want == "r-prod" {
				prod := &runnerConn{id: "r-prod", name: "prod", machine: "prod"}
				h.mu.Lock()
				h.conns["r-prod"] = prod
				h.mu.Unlock()
			}

			got := h.idleConnected(DeployRequestedEvent{Target: tt.target})
			if tt.want == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.id)
		})
	}
}

// A runner skips jobs targeted at someone else's machine and takes the first it fits, leaving the rest in order.
func TestHandler_popFitting_SkipsToFirstMatchingJob(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	prodJob := DeployRequestedEvent{ID: "d-prod", Kind: RequestDeploy, Target: "prod"}
	anyJob := DeployRequestedEvent{ID: "d-any", Kind: RequestDeploy}
	h.pending = []DeployRequestedEvent{prodJob, anyJob}
	qa := &runnerConn{id: "r-qa", name: "qa", machine: "qa"}

	got, ok := h.popFitting(qa)
	require.True(t, ok)
	assert.Equal(t, "d-any", got.ID)
	require.Len(t, h.pending, 1)
	assert.Equal(t, "d-prod", h.pending[0].ID)
}

// Two runners can share one machine (a pool); a job targeted at that machine goes to whichever is idle, and a
// runner on a different machine never takes it even if it's idle too (issue 05).
func TestHandler_idleConnected_PoolsByMachine(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	busy := &runnerConn{id: "r-1", machine: "prod"}
	busy.setJob(&DeployRequestedEvent{ID: "running"})
	idleSameMachine := &runnerConn{id: "r-2", machine: "prod"}
	idleOtherMachine := &runnerConn{id: "r-3", machine: "staging"}
	h.mu.Lock()
	h.conns["r-1"] = busy
	h.conns["r-2"] = idleSameMachine
	h.conns["r-3"] = idleOtherMachine
	h.mu.Unlock()

	got := h.idleConnected(DeployRequestedEvent{Target: "prod"})
	require.NotNil(t, got)
	assert.Equal(t, "r-2", got.id)
}

type fakeGitTokens struct {
	token string
	err   error
}

func (f fakeGitTokens) RepoToken(context.Context, string) (string, error) { return f.token, f.err }

// A build frame carries the connector token the server resolved; deploy frames never do.
func TestHandler_buildFrame_AttachesCloneTokenToBuilds(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.GitTokens = fakeGitTokens{token: "ghu_job"} })
	build, err := h.buildFrame(context.Background(), DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Repo: "org/app", Ref: "main"})
	require.NoError(t, err)
	assert.Equal(t, "ghu_job", build.GitToken)
	deploy, err := h.buildFrame(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "s", Image: "img"})
	require.NoError(t, err)
	assert.Empty(t, deploy.GitToken)

	h = newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.GitTokens = fakeGitTokens{err: errors.New("github not connected")} })
	build, err = h.buildFrame(context.Background(), DeployRequestedEvent{ID: "b2", Kind: RequestBuild, Repo: "org/app", Ref: "main"})
	require.NoError(t, err, "a missing token degrades to the runner's own, never blocks dispatch")
	assert.Empty(t, build.GitToken)
}

// TestHandler_resolveEnv covers ticket 14's dispatch-time env resolution: the
// bus event's own Env field (redacted keys only) is never the source of
// truth for what reaches a runner — resolveEnv is.
func TestHandler_resolveEnv(t *testing.T) {
	t.Run("no EnvLookup wired resolves to nil", func(t *testing.T) {
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
		env, err := h.resolveEnv(context.Background(), DeployRequestedEvent{Service: "api"})
		require.NoError(t, err)
		assert.Nil(t, env)
	})

	t.Run("blank service resolves to nil without calling the lookup", func(t *testing.T) {
		envs := &fakeEnvLookup{err: errors.New("must not be called")}
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.Envs = envs })
		env, err := h.resolveEnv(context.Background(), DeployRequestedEvent{})
		require.NoError(t, err)
		assert.Nil(t, env)
	})

	t.Run("resolves the real values by service name", func(t *testing.T) {
		envs := &fakeEnvLookup{values: map[string]map[string]string{"api": {"DB_PASSWORD": "sup3rSecret"}}}
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.Envs = envs })
		env, err := h.resolveEnv(context.Background(), DeployRequestedEvent{Service: "api"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"DB_PASSWORD": "sup3rSecret"}, env)
	})

	t.Run("lookup failure is wrapped", func(t *testing.T) {
		envs := &fakeEnvLookup{err: errors.New("boom")}
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.Envs = envs })
		_, err := h.resolveEnv(context.Background(), DeployRequestedEvent{Service: "api"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api")
	})
}

func TestHandler_ServeHTTP_Auth(t *testing.T) {
	bus := newFakeBus()
	h := newTestHandler(bus, newFakeRunnerRepo())
	srv := httptest.NewServer(h)
	defer srv.Close()

	tests := []struct {
		name string
		url  string
		want int
	}{
		{name: "missing token", url: srv.URL + "?runner_id=r1", want: http.StatusUnauthorized},
		{name: "wrong token", url: srv.URL + "?token=nope&runner_id=r1", want: http.StatusUnauthorized},
		{name: "missing runner id", url: srv.URL + "?token=s3cr3t", want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(tt.url)
			require.NoError(t, err)
			defer func() { require.NoError(t, resp.Body.Close()) }()
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}

func TestHandler_ServeHTTP_SecretLookupFailure(t *testing.T) {
	repo := newFakeRunnerRepo()
	repo.secretErr = assert.AnError
	h := newTestHandler(newFakeBus(), repo)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "?token=whatever&runner_id=r1")
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestHandler_HeartbeatTimeout(t *testing.T) {
	repo := newFakeRunnerRepo()
	bus := newFakeBus()
	h := newTestHandler(bus, repo)
	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=quiet", nil)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

	_, _, err = conn.Read(ctx)
	require.Error(t, err, "server must close a runner that misses 3 heartbeats")

	eventually(t, 2*time.Second, func() bool { return repo.connectedCount() == 0 })
	eventually(t, 2*time.Second, func() bool { return len(bus.topicEvents(TopicRunnerDisconnected)) == 1 })
}

// TestHandler_Disconnect_WritesDisconnectedAfterRequestContextEnds covers readLoop's teardown: whether the
// request context is cancelled under the handler or the runner process is killed, the row still ends
// connected=false and runner.disconnected is still published.
func TestHandler_Disconnect_WritesDisconnectedAfterRequestContextEnds(t *testing.T) {
	tests := []struct {
		name string
		drop func(conn *websocket.Conn, cancelRequest context.CancelFunc)
	}{
		{"request context cancelled", func(_ *websocket.Conn, cancelRequest context.CancelFunc) { cancelRequest() }},
		{"runner process killed", func(conn *websocket.Conn, _ context.CancelFunc) { _ = conn.CloseNow() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRunnerRepo()
			bus := newFakeBus()
			h := newTestHandler(bus, repo)
			cancelRequest := make(chan context.CancelFunc, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithCancel(r.Context())
				cancelRequest <- cancel
				h.ServeHTTP(w, r.WithContext(ctx))
			}))
			defer srv.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1", nil)
			require.NoError(t, err)
			defer func() { _ = conn.CloseNow() }() // already closed by the drop or the server; the second close errors

			eventually(t, 2*time.Second, func() bool { return repo.connectedCount() == 1 })

			tt.drop(conn, <-cancelRequest)

			eventually(t, 2*time.Second, func() bool { return len(bus.topicEvents(TopicRunnerDisconnected)) == 1 })
			got, err := repo.GetByID(context.Background(), "r-1")
			require.NoError(t, err)
			assert.False(t, got.Connected, "the disconnect write must land after the request context is gone")
		})
	}
}

func TestHandler_ServeHTTP_RecordsRunnerVersion(t *testing.T) {
	repo := newFakeRunnerRepo()
	bus := newFakeBus()
	h := newTestHandler(bus, repo)
	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&version=v0.1.6", nil)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }() // already closed below; a second CloseNow errors on the closed conn

	eventually(t, 2*time.Second, func() bool {
		r, err := repo.GetByID(context.Background(), "r-1")
		return err == nil && r.Version == "v0.1.6"
	})

	require.NoError(t, conn.CloseNow())

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	conn2, _, err := websocket.Dial(ctx2, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&version=v0.1.7", nil)
	require.NoError(t, err)
	defer func() { _ = conn2.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

	eventually(t, 2*time.Second, func() bool {
		r, err := repo.GetByID(context.Background(), "r-1")
		return err == nil && r.Version == "v0.1.7"
	})
}

func TestHandler_dispatchFrame_JoinNetworksResultDoesNotError(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	err := h.dispatchFrame(context.Background(), newTestConn(), Frame{
		Type: FrameJoinNetworksResult, GatewayContainer: "gw", Status: BuildStatusFailed, Error: "boom",
	})
	require.NoError(t, err, "a join result is logged, never dead-lettered")
}

// TestHandler_JoinNetworks_SendsFrameToConnectedRunner covers the ad hoc join_networks path: an exposure
// created for a running stack asks the runner connected under the target's machine name to join.
func TestHandler_JoinNetworks_SendsFrameToConnectedRunner(t *testing.T) {
	repo := newFakeRunnerRepo()
	h := newTestHandler(newFakeBus(), repo)
	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&name=host1", nil)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

	eventually(t, 2*time.Second, func() bool {
		_, err := repo.GetByID(context.Background(), "r-1")
		return err == nil
	})

	require.NoError(t, h.JoinNetworks(context.Background(), "host1", "gateway-container", []string{"net1"}))

	var raw json.RawMessage
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &raw))
	frame, err := ParseFrame(raw)
	require.NoError(t, err)
	assert.Equal(t, FrameJoinNetworks, frame.Type)
	assert.Equal(t, "gateway-container", frame.GatewayContainer)
	assert.Equal(t, []string{"net1"}, frame.JoinNetworks)
}

// TestHandler_JoinNetworks_NoConnectedRunnerIsANoOp covers the best-effort contract: a machine with nothing
// currently connected is not an error, since the next deploy carries the same join step.
func TestHandler_JoinNetworks_NoConnectedRunnerIsANoOp(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	err := h.JoinNetworks(context.Background(), "no-such-machine", "gateway-container", []string{"net1"})
	require.NoError(t, err)
}

// TestHandler_handleUpgradeRequest_DispatchesToInstanceRunner covers the Service -> Handler dispatch edge: an
// instance.upgrade_requested event sends assign_upgrade to the connected runner id "instance" and holds its slot.
func TestHandler_handleUpgradeRequest_DispatchesToInstanceRunner(t *testing.T) {
	repo := newFakeRunnerRepo()
	h := newTestHandler(newFakeBus(), repo)
	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id="+instanceRunnerID, nil)
	require.NoError(t, err)
	// the server side may close first, so a failed second close is expected here
	defer func() { _ = conn.CloseNow() }()

	eventually(t, 2*time.Second, func() bool {
		_, err := repo.GetByID(context.Background(), instanceRunnerID)
		return err == nil
	})

	require.NoError(t, h.handleUpgradeRequest(context.Background(), eventbus.Event{
		Payload: mustJSON(t, InstanceUpgradeRequestedEvent{ID: "u-1", Version: "v0.2.1"}),
	}))

	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	require.NoError(t, err)
	var raw json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	frame, err := ParseFrame(raw)
	require.NoError(t, err)
	assert.Equal(t, FrameAssignUpgrade, frame.Type)
	assert.Equal(t, "u-1", frame.ID)
	assert.Equal(t, "v0.2.1", frame.Version)

	h.mu.Lock()
	c := h.conns[instanceRunnerID]
	h.mu.Unlock()
	require.NotNil(t, c)
	assert.Equal(t, "u-1", c.upgradeSnapshot(), "dispatch holds the runner's slot like a deploy job would")
}

func TestHandler_handleUpgradeRequest_NoConnectedRunnerIsANoOp(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	err := h.handleUpgradeRequest(context.Background(), eventbus.Event{
		Payload: mustJSON(t, InstanceUpgradeRequestedEvent{ID: "u-1", Version: "v0.2.1"}),
	})
	require.NoError(t, err)
}

func TestHandler_handleUpgradeRequest_MalformedPayloadIsFatal(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	err := h.handleUpgradeRequest(context.Background(), eventbus.Event{Payload: []byte("{")})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrFatal))
}

// TestHandler_idleConnected_UpgradeBusyRunnerIsSkipped covers the busy-slot rule: a runner mid-upgrade never
// takes a deploy job, the same as one with a running job (instance-upgrade spec).
func TestHandler_idleConnected_UpgradeBusyRunnerIsSkipped(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	c := &runnerConn{id: instanceRunnerID}
	c.setUpgrade("u-1")
	h.mu.Lock()
	h.conns[instanceRunnerID] = c
	h.mu.Unlock()

	got := h.idleConnected(DeployRequestedEvent{})
	assert.Nil(t, got)
}

func TestHandler_dispatchFrame_UpgradeProgressLogsOnly(t *testing.T) {
	bus := newFakeBus()
	h := newTestHandler(bus, newFakeRunnerRepo())
	err := h.dispatchFrame(context.Background(), newTestConn(), Frame{Type: FrameUpgradeProgress, ID: "u-1", Log: "pulling image"})
	require.NoError(t, err)
	assert.Empty(t, bus.published, "progress is logged, not published")
}

func TestHandler_dispatchFrame_UpgradeResult(t *testing.T) {
	t.Run("started persists and leaves the slot held", func(t *testing.T) {
		upgrades := newFakeUpgradeRepo()
		now := time.Now().UTC()
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{ID: "u-1", Status: UpgradeStatusPending, CreatedAt: now, UpdatedAt: now}))
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo(), func(c *HandlerConfig) { c.Upgrades = upgrades })
		c := newTestConn()
		c.setUpgrade("u-1")

		require.NoError(t, h.dispatchFrame(context.Background(), c, Frame{
			Type: FrameUpgradeResult, ID: "u-1", Status: UpgradeStatusStarted, Log: "helper container abc123",
		}))

		got, err := upgrades.GetByID(context.Background(), "u-1")
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusStarted, got.Status)
		assert.Equal(t, "u-1", c.upgradeSnapshot(), "the connection is about to be torn down by the runner's own restart")
		require.Len(t, bus.topicEvents(TopicInstanceUpgradeChanged), 1)
		evt := decodeEvent[Upgrade](t, bus.topicEvents(TopicInstanceUpgradeChanged)[0])
		assert.Equal(t, UpgradeStatusStarted, evt.Status)
	})

	t.Run("failed persists and frees the slot", func(t *testing.T) {
		upgrades := newFakeUpgradeRepo()
		now := time.Now().UTC()
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{ID: "u-1", Status: UpgradeStatusPending, CreatedAt: now, UpdatedAt: now}))
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo(), func(c *HandlerConfig) { c.Upgrades = upgrades })
		c := newTestConn()
		c.setUpgrade("u-1")

		require.NoError(t, h.dispatchFrame(context.Background(), c, Frame{
			Type: FrameUpgradeResult, ID: "u-1", Status: UpgradeStatusFailed, Error: "helper container exited 1",
		}))

		got, err := upgrades.GetByID(context.Background(), "u-1")
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusFailed, got.Status)
		assert.Equal(t, "helper container exited 1", got.Error)
		assert.Empty(t, c.upgradeSnapshot())
		assert.Len(t, bus.topicEvents(TopicInstanceUpgradeChanged), 1)
	})
}

// TestHandler_Disconnect_UpgradeStatus covers the readLoop cleanup path in each record state: pending fails
// with "runner disconnected", started (the expected pre-restart drop) is left alone.
func TestHandler_Disconnect_UpgradeStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantStatus string
		wantError  string
	}{
		{"pending fails on disconnect", UpgradeStatusPending, UpgradeStatusFailed, "runner disconnected"},
		{"started is left unchanged", UpgradeStatusStarted, UpgradeStatusStarted, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRunnerRepo()
			upgrades := newFakeUpgradeRepo()
			now := time.Now().UTC()
			require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
				ID: "u-1", FromVersion: "v1", ToVersion: "v2", Status: tt.status, CreatedAt: now, UpdatedAt: now,
			}))
			h := newTestHandler(newFakeBus(), repo, func(c *HandlerConfig) { c.Upgrades = upgrades })
			srv := httptest.NewServer(h)
			defer srv.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id="+instanceRunnerID, nil)
			require.NoError(t, err)

			eventually(t, 2*time.Second, func() bool {
				_, err := repo.GetByID(context.Background(), instanceRunnerID)
				return err == nil
			})
			require.NoError(t, h.handleUpgradeRequest(context.Background(), eventbus.Event{
				Payload: mustJSON(t, InstanceUpgradeRequestedEvent{ID: "u-1", Version: "v2"}),
			}))
			readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, _, err = conn.Read(readCtx)
			readCancel()
			require.NoError(t, err, "wait for assign_upgrade to land before closing, so the slot is actually held")

			require.NoError(t, conn.CloseNow())

			// SetConnected(false) runs right after the upgrade cleanup in readLoop's defer, so this also fences
			// the upgrade write.
			eventually(t, 2*time.Second, func() bool { return repo.connectedCount() == 0 })

			got, err := upgrades.GetByID(context.Background(), "u-1")
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantError, got.Error)
		})
	}
}

// TestHandler_sendUpdateIfNeeded covers the best-effort update push on connect: a release-build server tells a
// runner reporting an older real version to fetch and swap itself, and stays silent otherwise.
func TestHandler_sendUpdateIfNeeded(t *testing.T) {
	t.Run("sends an update frame when the runner reports a different version", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		gh := fakeGitHub(t, "v0.2.0", "binary-bytes")
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
		h.SetUpdateSource(&fakeSettingsReader{url: "https://app.example.com"}, release.New(release.Config{APIBase: gh.URL}))
		srv := httptest.NewServer(h)
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&version=v0.1.5&os=linux&arch=amd64", nil)
		require.NoError(t, err)
		defer func() { _ = conn.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

		frame := readFrame(t, conn, 2*time.Second)
		assert.Equal(t, FrameUpdate, frame.Type)
		assert.Equal(t, "v0.2.0", frame.Version)
		assert.Equal(t, "https://app.example.com/api/runners/download/linux-amd64?version=v0.2.0", frame.URL)
		assert.Equal(t, "deadbeef", frame.Sha256)
	})

	t.Run("stays silent when the runner reports dev", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		gh := fakeGitHub(t, "v0.2.0", "binary-bytes")
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
		h.SetUpdateSource(&fakeSettingsReader{url: "https://app.example.com"}, release.New(release.Config{APIBase: gh.URL}))
		srv := httptest.NewServer(h)
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&version=dev&os=linux&arch=amd64", nil)
		require.NoError(t, err)
		defer func() { _ = conn.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

		assertNoFrame(t, conn, 300*time.Millisecond)
	})

	t.Run("stays silent when the runner already reports the server's own version", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		gh := fakeGitHub(t, "v0.2.0", "binary-bytes")
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
		h.SetUpdateSource(&fakeSettingsReader{url: "https://app.example.com"}, release.New(release.Config{APIBase: gh.URL}))
		srv := httptest.NewServer(h)
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&version=v0.2.0&os=linux&arch=amd64", nil)
		require.NoError(t, err)
		defer func() { _ = conn.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

		assertNoFrame(t, conn, 300*time.Millisecond)
	})

	t.Run("stays silent for an unsupported os/arch target", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		gh := fakeGitHub(t, "v0.2.0", "binary-bytes")
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
		h.SetUpdateSource(&fakeSettingsReader{url: "https://app.example.com"}, release.New(release.Config{APIBase: gh.URL}))
		srv := httptest.NewServer(h)
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, wsURL(srv)+"?token=s3cr3t&runner_id=r-1&version=v0.1.5&os=plan9&arch=amd64", nil)
		require.NoError(t, err)
		defer func() { _ = conn.CloseNow() }() // CloseNow after a read error or heartbeat close returns an expected "already closed" error

		assertNoFrame(t, conn, 300*time.Millisecond)
	})
}

func readFrame(t *testing.T, conn *websocket.Conn, wait time.Duration) *Frame {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	frame, err := ParseFrame(data)
	require.NoError(t, err)
	return frame
}

func assertNoFrame(t *testing.T, conn *websocket.Conn, wait time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	_, _, err := conn.Read(ctx)
	require.Error(t, err, "expected no frame to arrive")
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

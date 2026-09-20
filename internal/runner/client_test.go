package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every run-strategy field on the assign frame must reach the executor; cloudflared once started with the image's default command because Command was dropped here.
func TestClient_startJob_CopiesRunFields(t *testing.T) {
	exec := &fakeExecutor{deployFn: func(context.Context, DeployRequestedEvent, func(Frame)) {}}
	c := newTestClient("ws://server:8081/ws/runner", exec)
	c.startJob(context.Background(), nil, Frame{
		Type: FrameAssignDeploy, ID: "d1", Service: "cloudflared-instance", Image: "cloudflare/cloudflared:latest",
		Strategy: "run", Network: "nexul_default",
		Ports: []string{"80:80"}, Mounts: []string{"/var/run/docker.sock:/var/run/docker.sock:ro"},
		Command: []string{"tunnel", "--no-autoupdate", "run"}, GitToken: "job-token",
	})
	require.Eventually(t, func() bool {
		exec.mu.Lock()
		defer exec.mu.Unlock()
		return len(exec.deploys) == 1
	}, time.Second, 5*time.Millisecond)
	got := exec.deploys[0]
	assert.Equal(t, []string{"tunnel", "--no-autoupdate", "run"}, got.Command)
	assert.Equal(t, []string{"80:80"}, got.Ports)
	assert.Equal(t, []string{"/var/run/docker.sock:/var/run/docker.sock:ro"}, got.Mounts)
	assert.Equal(t, RequestDeploy, got.Kind)
	assert.Equal(t, "job-token", got.GitToken)
}

func newTestClient(srvURL string, exec Executor) *Client {
	return NewClient(ClientConfig{
		URL:               srvURL,
		Token:             "tok",
		RunnerID:          "r-1",
		Name:              "alpha",
		Logger:            testLogger(),
		Executor:          exec,
		HeartbeatInterval: 20 * time.Millisecond,
		ConnectTimeout:    time.Second,
		WriteTimeout:      time.Second,
		BackoffBase:       5 * time.Millisecond,
		BackoffMax:        50 * time.Millisecond,
	})
}

func runClient(t *testing.T, c *Client) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	t.Cleanup(cancel)
	return cancel, done
}

func TestClient_wsURL(t *testing.T) {
	c := newTestClient("ws://server:8081/ws/runner", &fakeExecutor{})
	u, err := url.Parse(c.wsURL())
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "tok", q.Get("token"))
	assert.Equal(t, "r-1", q.Get("runner_id"))
	assert.Equal(t, "alpha", q.Get("name"))
	assert.Equal(t, "ws", u.Scheme)
	assert.Empty(t, q.Get("version"), "no version configured means no query param")
	assert.Equal(t, runtime.GOOS, q.Get("os"))
	assert.Equal(t, runtime.GOARCH, q.Get("arch"))
}

func TestClient_wsURL_Version(t *testing.T) {
	c := newTestClient("ws://server:8081/ws/runner", &fakeExecutor{})
	c.cfg.Version = "v0.1.6"
	u, err := url.Parse(c.wsURL())
	require.NoError(t, err)
	assert.Equal(t, "v0.1.6", u.Query().Get("version"))
}

func TestClient_SendsHeartbeat(t *testing.T) {
	beats := make(chan Frame, 10)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var raw json.RawMessage
			if err := wsjson.Read(ctx, conn, &raw); err != nil {
				return
			}
			f, err := ParseFrame(raw)
			if err != nil {
				return
			}
			beats <- *f
		}
	})

	client := newTestClient(wsURL(srv), &fakeExecutor{})
	cancel, done := runClient(t, client)

	first := <-beats
	assert.Equal(t, FrameHeartbeat, first.Type)
	assert.Equal(t, "r-1", first.RunnerID)
	assert.Positive(t, first.TS)

	<-beats
	cancel()
	assert.NoError(t, <-done, "graceful stop on ctx cancel")
}

func TestClient_ExecutesAssignedBuild(t *testing.T) {
	results := make(chan Frame, 8)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return
		}
		assign := Frame{Type: FrameAssignBuild, ID: "build-1", Repo: "org/app", Ref: "main", Steps: []string{"go build"}}
		if err := wsjson.Write(ctx, conn, assign); err != nil {
			return
		}
		for {
			var r json.RawMessage
			if err := wsjson.Read(ctx, conn, &r); err != nil {
				return
			}
			fr, err := ParseFrame(r)
			if err != nil {
				return
			}
			results <- *fr
			if fr.Type == FrameBuildResult {
				return
			}
		}
	})

	exec := &fakeExecutor{}
	client := newTestClient(wsURL(srv), exec)
	cancel, done := runClient(t, client)

	progress := make([]Frame, 0, 4)
	var result Frame
	for result.Type != FrameBuildResult {
		select {
		case f := <-results:
			if f.Type == FrameBuildProgress {
				progress = append(progress, f)
			}
			result = f
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for build_result")
		}
	}
	assert.Equal(t, BuildStatusSuccess, result.Status)
	assert.Equal(t, "build-1", result.ID)
	require.NotEmpty(t, progress)
	assert.Equal(t, "build-1", progress[0].ID)

	assert.Contains(t, exec.buildIDs(), "build-1")
	cancel()
	assert.NoError(t, <-done)
}

// TestClient_ExecutesJoinNetworks covers the standalone join_networks path: a
// server-pushed join_networks frame, outside the deploy flow, reaches the executor and gets a result back.
func TestClient_ExecutesJoinNetworks(t *testing.T) {
	results := make(chan Frame, 4)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return
		}
		ask := Frame{Type: FrameJoinNetworks, GatewayContainer: "gw", JoinNetworks: []string{"net1"}}
		if err := wsjson.Write(ctx, conn, ask); err != nil {
			return
		}
		for {
			var r json.RawMessage
			if err := wsjson.Read(ctx, conn, &r); err != nil {
				return
			}
			fr, err := ParseFrame(r)
			if err != nil {
				return
			}
			results <- *fr
			if fr.Type == FrameJoinNetworksResult {
				return
			}
		}
	})

	exec := &fakeExecutor{}
	client := newTestClient(wsURL(srv), exec)
	cancel, done := runClient(t, client)

	var result Frame
	select {
	case result = <-results:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for join_networks_result")
	}
	assert.Equal(t, FrameJoinNetworksResult, result.Type)
	assert.Equal(t, "gw", result.GatewayContainer)
	assert.Equal(t, BuildStatusSuccess, result.Status)

	require.Eventually(t, func() bool {
		exec.mu.Lock()
		defer exec.mu.Unlock()
		return len(exec.joinCalls) == 1
	}, time.Second, 5*time.Millisecond)
	exec.mu.Lock()
	assert.Equal(t, "gw", exec.joinCalls[0].GatewayContainer)
	assert.Equal(t, []string{"net1"}, exec.joinCalls[0].Networks)
	exec.mu.Unlock()

	cancel()
	assert.NoError(t, <-done)
}

// TestClient_ExecutesAssignedUpgrade covers the upgrade dispatch path: assign_upgrade goes through the same
// startJob single-job slot as assign_deploy, reaches Executor.Upgrade with the raw frame, and jobFinished clears
// the slot once upgrade_result is sent (instance-upgrade spec, ticket 01).
func TestClient_ExecutesAssignedUpgrade(t *testing.T) {
	results := make(chan Frame, 4)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return
		}
		assign := Frame{Type: FrameAssignUpgrade, ID: "up-1", Version: "v0.2.0-beta-004"}
		if err := wsjson.Write(ctx, conn, assign); err != nil {
			return
		}
		for {
			var r json.RawMessage
			if err := wsjson.Read(ctx, conn, &r); err != nil {
				return
			}
			fr, err := ParseFrame(r)
			if err != nil {
				return
			}
			results <- *fr
			if fr.Type == FrameUpgradeResult {
				return
			}
		}
	})

	exec := &fakeExecutor{}
	exec.upgradeFn = func(_ context.Context, req Frame, send func(Frame) error) error {
		if err := send(Frame{Type: FrameUpgradeProgress, ID: req.ID, Log: "resolved compose project nexul at /opt/nexul"}); err != nil {
			return err
		}
		return send(Frame{Type: FrameUpgradeResult, ID: req.ID, Status: UpgradeStatusStarted, Log: "helper-id"})
	}
	client := newTestClient(wsURL(srv), exec)
	cancel, done := runClient(t, client)

	progress := make([]Frame, 0, 2)
	var result Frame
	for result.Type != FrameUpgradeResult {
		select {
		case f := <-results:
			if f.Type == FrameUpgradeProgress {
				progress = append(progress, f)
			}
			result = f
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for upgrade_result")
		}
	}
	assert.Equal(t, UpgradeStatusStarted, result.Status)
	assert.Equal(t, "up-1", result.ID)
	assert.Equal(t, "helper-id", result.Log)
	require.NotEmpty(t, progress)

	exec.mu.Lock()
	require.Len(t, exec.upgrades, 1)
	assert.Equal(t, "up-1", exec.upgrades[0].ID)
	assert.Equal(t, "v0.2.0-beta-004", exec.upgrades[0].Version)
	exec.mu.Unlock()

	require.Eventually(t, func() bool {
		client.mu.Lock()
		defer client.mu.Unlock()
		return client.jobID == "" && client.jobCancel == nil
	}, time.Second, 5*time.Millisecond, "jobFinished must clear the job slot once upgrade_result is sent")

	cancel()
	assert.NoError(t, <-done)
}

func TestClient_CancelAbortsRunningJob(t *testing.T) {
	started := make(chan struct{})
	aborted := make(chan struct{})
	exec := &fakeExecutor{}
	exec.buildFn = func(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
		close(started)
		<-ctx.Done()
		close(aborted)
		send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: "cancelled"})
	}

	results := make(chan Frame, 8)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return
		}
		assign := Frame{Type: FrameAssignBuild, ID: "build-1", Repo: "org/app", Ref: "main", Steps: []string{"long"}}
		if err := wsjson.Write(ctx, conn, assign); err != nil {
			return
		}
		cancel := Frame{Type: FrameCancel, ID: "build-1"}
		if err := wsjson.Write(ctx, conn, cancel); err != nil {
			return
		}
		for {
			var r json.RawMessage
			if err := wsjson.Read(ctx, conn, &r); err != nil {
				return
			}
			fr, err := ParseFrame(r)
			if err != nil {
				return
			}
			results <- *fr
		}
	})

	client := newTestClient(wsURL(srv), exec)
	cancel, done := runClient(t, client)

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job never started")
	}
	select {
	case <-aborted:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel frame did not abort the job")
	}

	select {
	case f := <-results:
		assert.Equal(t, FrameBuildResult, f.Type)
		assert.Equal(t, BuildStatusFailed, f.Status)
		assert.Equal(t, "cancelled", f.Error)
	case <-time.After(2 * time.Second):
		t.Fatal("no result frame after cancel")
	}
	cancel()
	assert.NoError(t, <-done)
}

// TestClient_HandlesUpdateFrame_AppliesWhenIdle exercises the read loop's own FrameUpdate case (as opposed to
// update_test.go's direct handleUpdate calls, which bypass the websocket wiring entirely).
func TestClient_HandlesUpdateFrame_AppliesWhenIdle(t *testing.T) {
	t.Setenv("NEXUL_RUNNER_ID", "host-1")
	body := "new-binary-bytes"
	sum := sha256.Sum256([]byte(body))
	sha := hex.EncodeToString(sum[:])
	var downloaded atomic.Bool
	updateSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloaded.Store(true)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(updateSrv.Close)

	exe := writeExe(t, "old-binary")
	calls := stubReexec(t)
	origExeFn := runnerExecutableFn
	runnerExecutableFn = func() (string, error) { return exe, nil }
	t.Cleanup(func() { runnerExecutableFn = origExeFn })

	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		update := Frame{Type: FrameUpdate, Version: "v0.2.0", URL: updateSrv.URL, Sha256: sha}
		if err := wsjson.Write(ctx, conn, update); err != nil {
			return
		}
		// Keep draining heartbeats so the connection stays open long enough for the update download to finish;
		// a single Read here would close the connection (and cancel the in-flight download) on the first beat.
		for {
			var raw json.RawMessage
			if err := wsjson.Read(ctx, conn, &raw); err != nil {
				return
			}
		}
	})

	client := newTestClient(wsURL(srv), &fakeExecutor{})
	client.cfg.Version = "v0.1.6"
	cancel, done := runClient(t, client)

	require.Eventually(t, func() bool { return downloaded.Load() }, 2*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(calls.Calls()) == 1 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, exe, calls.Calls()[0])
	// Wait for the client to fully stop before the deferred cleanups restore the package-level stubs: an
	// in-flight handleUpdate goroutine reading them past this point would race the next test's own stubbing.
	cancel()
	assert.NoError(t, <-done)
}

func TestClient_ReconnectsAfterDrop(t *testing.T) {
	accepts := make(chan struct{}, 4)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		select {
		case accepts <- struct{}{}:
		default:
		}
		// Wait for the runner's heartbeat, then drop the connection so the
		// client is forced to reconnect.
		var raw json.RawMessage
		_ = wsjson.Read(ctx, conn, &raw)
	})

	client := newTestClient(wsURL(srv), &fakeExecutor{})
	cancel, done := runClient(t, client)

	select {
	case <-accepts:
	case <-time.After(2 * time.Second):
		t.Fatal("first connection never accepted")
	}
	select {
	case <-accepts:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not reconnect after the server closed")
	}
	cancel()
	assert.NoError(t, <-done)
}

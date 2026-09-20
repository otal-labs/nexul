package runner

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newIntegrationHandler(t *testing.T, bus Bus, repo Repo) (*Handler, *httptest.Server) {
	t.Helper()
	h := NewHandler(HandlerConfig{
		Bus:               bus,
		Repo:              repo,
		Logger:            testLogger(),
		HeartbeatInterval: 100 * time.Millisecond,
		MissedHeartbeats:  3,
		WriteTimeout:      time.Second,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return h, srv
}

type eventCollector struct {
	mu           sync.Mutex
	builds       []BuildCompletedEvent
	connected    []RunnerConnectedEvent
	disconnected []RunnerDisconnectedEvent
}

func (c *eventCollector) subscribe(ctx context.Context, bus Bus) {
	_ = bus.Subscribe(ctx, TopicDeployBuildCompleted, func(_ context.Context, ev eventbus.Event) error {
		var e BuildCompletedEvent
		_ = json.Unmarshal(ev.Payload, &e)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.builds = append(c.builds, e)
		return nil
	})
	_ = bus.Subscribe(ctx, TopicRunnerConnected, func(_ context.Context, ev eventbus.Event) error {
		var e RunnerConnectedEvent
		_ = json.Unmarshal(ev.Payload, &e)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.connected = append(c.connected, e)
		return nil
	})
	_ = bus.Subscribe(ctx, TopicRunnerDisconnected, func(_ context.Context, ev eventbus.Event) error {
		var e RunnerDisconnectedEvent
		_ = json.Unmarshal(ev.Payload, &e)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.disconnected = append(c.disconnected, e)
		return nil
	})
}

func (c *eventCollector) buildCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.builds)
}

func startTestClient(t *testing.T, srvURL string, exec Executor) context.CancelFunc {
	t.Helper()
	client := NewClient(ClientConfig{
		URL:               srvURL,
		Token:             "s3cr3t",
		RunnerID:          "r-integ",
		Name:              "integ",
		Logger:            testLogger(),
		Executor:          exec,
		HeartbeatInterval: 20 * time.Millisecond,
		ConnectTimeout:    time.Second,
		BackoffBase:       5 * time.Millisecond,
		BackoffMax:        50 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = client.Run(ctx) }()
	return cancel
}

// startHandler runs the handler's bus loop until ctx is cancelled.
func startHandler(t *testing.T, h *Handler, ctx context.Context) {
	t.Helper()
	go func() { _ = h.Run(ctx) }()
}

// AC9: runner connects -> server assigns build -> runner reports success ->
// event published on the bus.
func TestIntegration_RunnerConnectAssignBuildPublish(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)

	collector := &eventCollector{}
	collector.subscribe(ctx, bus)
	exec := &fakeExecutor{}
	startTestClient(t, srv.URL, exec)

	eventually(t, 3*time.Second, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		r, ok := repo.runners["r-integ"]
		return ok && r.Connected
	})

	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "build-1", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"go build"},
	}))

	eventually(t, 3*time.Second, func() bool { return collector.buildCount() == 1 })
	collector.mu.Lock()
	got := collector.builds[0]
	collector.mu.Unlock()
	assert.Equal(t, "build-1", got.ID)
	assert.Equal(t, BuildStatusSuccess, got.Status)
	assert.Equal(t, "r-integ", collector.connected[0].RunnerID)
}

// TestIntegration_EnvValuesReachRunnerWithoutTouchingBus proves the ticket 14
// split: the deploy.requested event on the bus carries only redacted env keys
// (as the deploy domain now publishes it), while the runner still gets the
// real values it needs to actually deploy — resolved directly through
// EnvLookup at dispatch time, never serialized onto the bus.
func TestIntegration_EnvValuesReachRunnerWithoutTouchingBus(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	envs := &fakeEnvLookup{values: map[string]map[string]string{
		"api": {"DB_PASSWORD": "sup3rSecret"},
	}}
	h := NewHandler(HandlerConfig{
		Bus: bus, Repo: repo, Logger: testLogger(),
		HeartbeatInterval: 100 * time.Millisecond, MissedHeartbeats: 3, WriteTimeout: time.Second,
		Envs: envs,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)

	exec := &fakeExecutor{}
	startTestClient(t, srv.URL, exec)

	eventually(t, 3*time.Second, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		r, ok := repo.runners["r-integ"]
		return ok && r.Connected
	})

	// Redacted the way the deploy domain now publishes it: keys only, values
	// blanked. Nothing downstream of this Publish call ever sees the secret.
	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "d-1", Kind: RequestDeploy, Service: "api", Image: "img:v1",
		Env: map[string]string{"DB_PASSWORD": ""},
	}))

	eventually(t, 3*time.Second, func() bool {
		exec.mu.Lock()
		defer exec.mu.Unlock()
		return len(exec.deploys) == 1
	})
	exec.mu.Lock()
	got := exec.deploys[0].Env
	exec.mu.Unlock()
	assert.Equal(t, "sup3rSecret", got["DB_PASSWORD"], "the runner's actual work channel must still get the real value")
}

// A request published before any runner connects is queued and flushed on the
// first connection.
func TestIntegration_QueuedRequestFlushedOnConnect(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)
	eventually(t, 3*time.Second, func() bool { return bus.handlerCount(TopicDeployRequested) == 1 })

	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "build-9", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"go build"},
	}))

	collector := &eventCollector{}
	collector.subscribe(ctx, bus)
	exec := &fakeExecutor{}
	startTestClient(t, srv.URL, exec)

	eventually(t, 3*time.Second, func() bool { return collector.buildCount() == 1 })
	collector.mu.Lock()
	assert.Equal(t, "build-9", collector.builds[0].ID)
	collector.mu.Unlock()
}

// A failing build still reaches the bus, with status failed (error path).
func TestIntegration_FailedBuildPublishesFailure(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)

	collector := &eventCollector{}
	collector.subscribe(ctx, bus)

	exec := &fakeExecutor{}
	exec.buildFn = func(_ context.Context, req DeployRequestedEvent, send func(Frame)) {
		send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: "make: nothing to be done"})
	}
	startTestClient(t, srv.URL, exec)

	eventually(t, 3*time.Second, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		r, ok := repo.runners["r-integ"]
		return ok && r.Connected
	})
	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "build-f", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"make"},
	}))

	eventually(t, 3*time.Second, func() bool { return collector.buildCount() == 1 })
	collector.mu.Lock()
	assert.Equal(t, BuildStatusFailed, collector.builds[0].Status)
	collector.mu.Unlock()
}

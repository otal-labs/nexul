package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingDeploy signals when a deploy starts and when its context is
// cancelled, and otherwise blocks forever — the shape of a job that is
// mid-flight when something else happens to it.
func blockingDeploy(started chan<- string, stopped chan<- string) func(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	return func(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
		started <- req.ID
		<-ctx.Done()
		stopped <- req.ID
	}
}

// A runner that disappears mid-job marks the job failed with a clear
// reason, and the job is never requeued for another runner.
func TestIntegration_DisconnectMidJobMarksFailed_NotRequeued(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)
	eventually(t, 5*time.Second, func() bool { return bus.handlerCount(TopicDeployRequested) == 1 })

	started := make(chan string, 1)
	stopped := make(chan string, 1)
	exec := &fakeExecutor{deployFn: blockingDeploy(started, stopped)}
	clientCancel := startTestClient(t, srv.URL, exec)

	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "deploy-1", Kind: RequestDeploy, Service: "api", Image: "ghcr.io/x/api", Strategy: "compose",
	}))
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("job never started")
	}

	clientCancel() // runner process dies mid-job

	eventually(t, 5*time.Second, func() bool {
		return len(bus.topicEvents(TopicDeployStatusChanged)) == 1
	})
	evs := bus.topicEvents(TopicDeployStatusChanged)
	got := decodeEvent[DeployStatusChangedEvent](t, evs[0])
	assert.Equal(t, "deploy-1", got.ID)
	assert.Equal(t, DeployStatusFailed, got.Status)
	assert.Equal(t, "runner disconnected", got.Error)

	// Never auto-requeued: the job is not sitting in the queue, so a fresh
	// runner cannot pick it up.
	assert.Empty(t, h.Queue())
	select {
	case id := <-stopped:
		assert.Equal(t, "deploy-1", id)
	case <-time.After(5 * time.Second):
		t.Fatal("job context never cancelled on disconnect")
	}
}

// A cancel frame stops a running job and the outcome lands as failed
// with reason "cancelled".
func TestIntegration_CancelRunningJob(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)
	eventually(t, 5*time.Second, func() bool {
		return bus.handlerCount(TopicDeployRequested) == 1 && bus.handlerCount(TopicDeployCancelRequested) == 1
	})

	started := make(chan string, 1)
	stopped := make(chan string, 1)
	exec := &fakeExecutor{deployFn: blockingDeploy(started, stopped)}
	startTestClient(t, srv.URL, exec)

	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "deploy-2", Kind: RequestDeploy, Service: "api", Image: "ghcr.io/x/api", Strategy: "compose",
	}))
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("job never started")
	}

	require.NoError(t, bus.Publish(ctx, TopicDeployCancelRequested, DeployCancelRequestedEvent{ID: "deploy-2"}))

	select {
	case id := <-stopped:
		assert.Equal(t, "deploy-2", id, "cancel frame must abort the running job")
	case <-time.After(5 * time.Second):
		t.Fatal("job context never cancelled on cancel")
	}
	eventually(t, 5*time.Second, func() bool {
		return len(bus.topicEvents(TopicDeployStatusChanged)) == 1
	})
	got := decodeEvent[DeployStatusChangedEvent](t, bus.topicEvents(TopicDeployStatusChanged)[0])
	assert.Equal(t, "deploy-2", got.ID)
	assert.Equal(t, DeployStatusFailed, got.Status)
	assert.Equal(t, "cancelled", got.Error)
	assert.Empty(t, h.Queue())
}

// A runner takes one job at a time; when it finishes, the next queued
// job is dispatched to it (FIFO).
func TestIntegration_CompletedJobPumpsQueue(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)
	eventually(t, 5*time.Second, func() bool { return bus.handlerCount(TopicDeployRequested) == 1 })

	collector := &eventCollector{}
	collector.subscribe(ctx, bus)

	// Publish two builds before any runner connects so both queue.
	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "build-1", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"go build"},
	}))
	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "build-2", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"go build"},
	}))
	require.Len(t, h.Queue(), 2)

	exec := &fakeExecutor{}
	startTestClient(t, srv.URL, exec)

	eventually(t, 5*time.Second, func() bool { return collector.buildCount() == 2 })
	collector.mu.Lock()
	got := map[string]bool{}
	for _, b := range collector.builds {
		got[b.ID] = true
	}
	collector.mu.Unlock()
	assert.True(t, got["build-1"], "first queued job dispatched")
	assert.True(t, got["build-2"], "second job dispatched after the first completed")
	assert.Empty(t, h.Queue())
}

// A runner shows the job it is executing and stops showing it once the job
// completes. The executor holds the job open until the test has observed the
// running state, so the assertion never races a transient window.
func TestIntegration_RunnerShowsRunningJobThenIdle(t *testing.T) {
	bus := newFakeBus()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, bus, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	startHandler(t, h, ctx)
	eventually(t, 5*time.Second, func() bool { return bus.handlerCount(TopicDeployRequested) == 1 })

	started := make(chan struct{})
	release := make(chan struct{})
	exec := &fakeExecutor{}
	exec.buildFn = func(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
		send(Frame{Type: FrameBuildProgress, ID: req.ID, Step: 1, Total: 1})
		close(started)
		<-release
		send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusSuccess})
		send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusHealthy})
	}
	startTestClient(t, srv.URL, exec)
	eventually(t, 5*time.Second, func() bool {
		return repo.connectedCount() == 1
	})

	require.NoError(t, bus.Publish(ctx, TopicDeployRequested, DeployRequestedEvent{
		ID: "build-1", Kind: RequestBuild, Repo: "org/app", Ref: "main", Steps: []string{"go build"},
	}))
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("job never started")
	}

	// The job is held open, so the running state is guaranteed to be visible.
	eventually(t, 5*time.Second, func() bool {
		for _, st := range h.Runners() {
			if st.RunningJob != nil && st.RunningJob.ID == "build-1" {
				return true
			}
		}
		return false
	})

	close(release)

	// Once the terminal result is processed the runner is idle again.
	eventually(t, 5*time.Second, func() bool {
		for _, st := range h.Runners() {
			if st.RunningJob != nil {
				return false
			}
		}
		return true
	})
}

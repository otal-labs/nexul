package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestHandler_idleConnected_SkipsBusyRunner(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	busy := &runnerConn{id: "r-1"}
	busy.setJob(&DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"})
	idle := &runnerConn{id: "r-2"}
	h.mu.Lock()
	h.conns["r-1"] = busy
	h.conns["r-2"] = idle
	h.mu.Unlock()

	assert.Equal(t, idle, h.idleConnected(DeployRequestedEvent{}))
}

func TestHandler_idleConnected_NoneIdle(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	busy := &runnerConn{id: "r-1"}
	busy.setJob(&DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy})
	h.mu.Lock()
	h.conns["r-1"] = busy
	h.mu.Unlock()

	assert.Nil(t, h.idleConnected(DeployRequestedEvent{}))
}

func TestHandler_Runners_ExposesRunningJob(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	c := &runnerConn{id: "r-1"}
	c.setJob(&DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"})
	h.mu.Lock()
	h.conns["r-1"] = c
	h.mu.Unlock()

	got := h.Runners()
	require.Len(t, got, 1)
	assert.Equal(t, "r-1", got[0].RunnerID)
	require.NotNil(t, got[0].RunningJob)
	assert.Equal(t, "d-1", got[0].RunningJob.ID)
	assert.Equal(t, RequestDeploy, got[0].RunningJob.Kind)
	assert.Equal(t, "api", got[0].RunningJob.Service)
}

func TestHandler_Runners_IdleRunner(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	h.mu.Lock()
	h.conns["r-1"] = &runnerConn{id: "r-1"}
	h.mu.Unlock()

	got := h.Runners()
	require.Len(t, got, 1)
	assert.Nil(t, got[0].RunningJob)
}

func TestHandler_Queue_ReturnsFIFO(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	h.queue(DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"})
	h.queue(DeployRequestedEvent{ID: "b-2", Kind: RequestBuild, Service: "web"})

	got := h.Queue()
	require.Len(t, got, 2)
	assert.Equal(t, "d-1", got[0].ID)
	assert.Equal(t, RequestDeploy, got[0].Kind)
	assert.Equal(t, "api", got[0].Service)
	assert.Equal(t, "b-2", got[1].ID)
}

func TestHandler_Cancel_QueuedJob(t *testing.T) {
	bus := newFakeBus()
	h := newTestHandler(bus, newFakeRunnerRepo())
	h.queue(DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"})

	require.NoError(t, h.Cancel(context.Background(), "d-1"))
	assert.Empty(t, h.Queue())

	evs := bus.topicEvents(TopicDeployStatusChanged)
	require.Len(t, evs, 1)
	got := decodeEvent[DeployStatusChangedEvent](t, evs[0])
	assert.Equal(t, "d-1", got.ID)
	assert.Equal(t, DeployStatusFailed, got.Status)
	assert.Equal(t, "cancelled", got.Error)
}

func TestHandler_Cancel_UnknownID_IsNoOp(t *testing.T) {
	bus := newFakeBus()
	h := newTestHandler(bus, newFakeRunnerRepo())
	h.queue(DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy})

	require.NoError(t, h.Cancel(context.Background(), "nope"))
	assert.Len(t, h.Queue(), 1)
	assert.Empty(t, bus.topicEvents(TopicDeployStatusChanged))
}

func TestHandler_handleCancelRequest(t *testing.T) {
	t.Run("valid payload cancels the job", func(t *testing.T) {
		bus := newFakeBus()
		h := newTestHandler(bus, newFakeRunnerRepo())
		h.queue(DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"})

		require.NoError(t, h.handleCancelRequest(context.Background(),
			eventbus.Event{Payload: mustJSON(t, DeployCancelRequestedEvent{ID: "d-1"})}))
		assert.Empty(t, h.Queue())
		require.Len(t, bus.topicEvents(TopicDeployStatusChanged), 1)
	})

	t.Run("malformed payload is fatal", func(t *testing.T) {
		h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
		err := h.handleCancelRequest(context.Background(), eventbus.Event{Payload: []byte("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

// A deploy.requested that cannot reach any runner sits in the queue and never
// disappears silently — Cancel is the only way out for an orphaned job.
func TestHandler_handleRequest_NoRunnerQueues(t *testing.T) {
	bus := newFakeBus()
	h := newTestHandler(bus, newFakeRunnerRepo())
	req := DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"}

	require.NoError(t, h.handleRequest(context.Background(), eventbus.Event{Payload: mustJSON(t, req)}))
	require.Len(t, h.Queue(), 1)
	assert.Empty(t, bus.topicEvents(TopicDeployStatusChanged))
}

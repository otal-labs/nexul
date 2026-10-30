package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestHandler_idleConnected_SkipsBusyRunner(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	busy := &runnerConn{id: "r-1"}
	busy.setJob(&DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"}, time.Time{})
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
	busy.setJob(&DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy}, time.Time{})
	h.mu.Lock()
	h.conns["r-1"] = busy
	h.mu.Unlock()

	assert.Nil(t, h.idleConnected(DeployRequestedEvent{}))
}

func TestHandler_Runners_ExposesRunningJob(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	c := &runnerConn{id: "r-1"}
	c.setJob(&DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api"}, time.Time{})
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

func TestDeployRequestedEvent_Views_CarryWhoStartedTheJobAndWhen(t *testing.T) {
	requested := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	started := requested.Add(90 * time.Second)
	tests := []struct {
		name string
		req  DeployRequestedEvent
	}{
		{"started by a person", DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, Service: "api", TriggeredBy: "user-1", RequestedAt: requested}},
		{"started by nobody", DeployRequestedEvent{ID: "d-2", Kind: RequestBuild, Service: "web", RequestedAt: requested}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queued := tt.req.Queued()
			assert.Equal(t, tt.req.TriggeredBy, queued.TriggeredBy)
			assert.Equal(t, requested, queued.RequestedAt)

			running := tt.req.RunningJob(started)
			assert.Equal(t, tt.req.TriggeredBy, running.TriggeredBy)
			assert.Equal(t, started, running.StartedAt)
		})
	}
}

func TestHandler_Queue_CarriesTheRequestsTimeAndRequester(t *testing.T) {
	requested := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	h.queue(DeployRequestedEvent{ID: "d-1", Kind: RequestDeploy, TriggeredBy: "user-1", RequestedAt: requested})

	got := h.Queue()
	require.Len(t, got, 1)
	assert.Equal(t, "user-1", got[0].TriggeredBy)
	assert.Equal(t, requested, got[0].RequestedAt)
}

// A job assigned to a runner is stamped with the hub's clock when it is assigned, not when the deploy was requested.
func TestHandler_Runners_StampsStartedAtWhenTheJobIsAssigned(t *testing.T) {
	assigned := time.Date(2026, 8, 2, 12, 5, 0, 0, time.UTC)
	f := newStreamFixture(t)
	f.h.cfg.Now = func() time.Time { return assigned }
	req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", Image: "api:1", TriggeredBy: "user-1", RequestedAt: assigned.Add(-time.Minute)}
	require.NoError(t, f.h.handleRequest(t.Context(), eventbus.Event{Payload: mustJSON(t, req)}))
	require.Len(t, f.h.Queue(), 1, "no runner yet, so the job waits")

	frames, _ := f.control(t, f.repo.enrolled("r-build", "build-box", "prod"))
	assert.Equal(t, FrameAssignDeploy, nextFrame(t, frames).Type)

	got := f.h.Runners()
	require.Len(t, got, 1)
	require.NotNil(t, got[0].RunningJob)
	assert.Equal(t, "user-1", got[0].RunningJob.TriggeredBy)
	assert.Equal(t, assigned, got[0].RunningJob.StartedAt)
	assert.Empty(t, f.h.Queue())
}

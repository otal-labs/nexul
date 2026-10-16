package runner

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func (h *Handler) waitingTeardowns(machine string) []Frame {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Frame(nil), h.teardowns[machine]...)
}

func (h *Handler) anyWaitingTeardowns() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.teardowns)
}

func stackDeleted(t *testing.T, h *Handler, e StackDeletedEvent) error {
	t.Helper()
	return h.handleStackDeleted(t.Context(), eventbus.Event{Payload: mustJSON(t, e)})
}

func readTeardownFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) Frame {
	t.Helper()
	var f Frame
	require.NoError(t, wsjson.Read(ctx, conn, &f))
	return f
}

func TestHandler_StackDeleted_ARejectedEventSendsNothing(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())

	t.Run("a payload that cannot be parsed is dead-lettered", func(t *testing.T) {
		err := h.handleStackDeleted(t.Context(), eventbus.Event{Payload: []byte("{")})
		require.ErrorIs(t, err, apperrs.ErrFatal)
	})
	t.Run("an unmanaged stack's containers are the owner's and stay", func(t *testing.T) {
		require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "shop", Strategy: "compose", Managed: false}))
		assert.Zero(t, h.anyWaitingTeardowns(), "nothing waits for a runner either")
	})
	t.Run("a stack with no machine has no runner to ask", func(t *testing.T) {
		require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Slug: "shop", Strategy: "compose", Managed: true}))
		assert.Zero(t, h.anyWaitingTeardowns())
	})
	t.Run("a slug docker would read as a flag is dropped, not retried", func(t *testing.T) {
		require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "-f", Strategy: "run", Managed: true}))
		assert.Zero(t, h.anyWaitingTeardowns())
	})
}

func TestHandler_StackDeleted_TheMachinesRunnerGetsOneTeardown(t *testing.T) {
	for name, tc := range map[string]struct{ strategy string }{
		"a compose stack": {strategy: "compose"},
		"a run stack":     {strategy: "run"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := newFakeRunnerRepo()
			h := newTestHandler(newFakeBus(), repo)
			srv := httptest.NewServer(h)
			defer srv.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			target, _, err := dialRunner(ctx, srv, repo.enrolled("r-1", "r-1", "host1"), "")
			require.NoError(t, err)
			defer func() { _ = target.CloseNow() }()
			other, _, err := dialRunner(ctx, srv, repo.enrolled("r-2", "r-2", "host2"), "")
			require.NoError(t, err)
			defer func() { _ = other.CloseNow() }()
			eventually(t, 2*time.Second, func() bool { return repo.connectedCount() == 2 })

			require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "edge", Strategy: tc.strategy, Managed: true}))

			got := readTeardownFrame(t, ctx, target)
			assert.Equal(t, Frame{Type: FrameStackTeardown, StackSlug: "edge", Strategy: tc.strategy}, got)
			readCtx, readCancel := context.WithTimeout(ctx, 200*time.Millisecond)
			defer readCancel()
			var stray Frame
			assert.Error(t, wsjson.Read(readCtx, other, &stray), "a runner on another machine hears nothing")
			assert.Zero(t, h.anyWaitingTeardowns())
		})
	}
}

func TestHandler_StackDeleted_AnOfflineMachineIsTornDownWhenItsRunnerConnects(t *testing.T) {
	repo := newFakeRunnerRepo()
	h := newTestHandler(newFakeBus(), repo)
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "edge", Strategy: "run", Managed: true}), "the delete does not wait for the runner")
	require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "shop", Strategy: "compose", Managed: true}))
	require.Len(t, h.waitingTeardowns("host1"), 2)

	elsewhere, _, err := dialRunner(ctx, srv, repo.enrolled("r-2", "r-2", "host2"), "")
	require.NoError(t, err)
	defer func() { _ = elsewhere.CloseNow() }()
	eventually(t, 2*time.Second, func() bool { return repo.connectedCount() == 1 })
	require.Len(t, h.waitingTeardowns("host1"), 2, "another machine's runner does not take them")

	conn, _, err := dialRunner(ctx, srv, repo.enrolled("r-1", "r-1", "host1"), "")
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()

	assert.Equal(t, Frame{Type: FrameStackTeardown, StackSlug: "edge", Strategy: "run"}, readTeardownFrame(t, ctx, conn))
	assert.Equal(t, Frame{Type: FrameStackTeardown, StackSlug: "shop", Strategy: "compose"}, readTeardownFrame(t, ctx, conn), "in the order they were deleted")
	eventually(t, time.Second, func() bool { return len(h.waitingTeardowns("host1")) == 0 })
}

func TestHandler_StackTeardownResult_AFailureIsLoggedNotFatal(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	c := &runnerConn{id: "r-1"}
	require.NoError(t, h.dispatchFrame(t.Context(), c, Frame{Type: FrameStackTeardownResult, StackSlug: "edge", Status: BuildStatusFailed, Error: "daemon down"}))
}

func TestClient_ExecutesStackTeardown(t *testing.T) {
	results := make(chan Frame, 1)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameStackTeardown, StackSlug: "edge", Strategy: "run"})
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameStackTeardownResult {
				results <- f
				return
			}
		}
	})
	exec := &fakeExecutor{}
	cancel, done := runClient(t, newTestClient(wsURL(srv), exec))
	defer func() { cancel(); <-done }()

	select {
	case result := <-results:
		assert.Equal(t, Frame{Type: FrameStackTeardownResult, StackSlug: "edge", Status: BuildStatusSuccess}, result)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stack_teardown_result")
	}
	exec.mu.Lock()
	defer exec.mu.Unlock()
	assert.Equal(t, []string{"run edge"}, exec.teardowns)
}

func TestShellExecutor_Teardown_RunsTheStacksOwnCommand(t *testing.T) {
	boom := errors.New("Cannot connect to the Docker daemon")
	missing := errors.New("docker rm -f edge: Error response from daemon: No such container: edge")
	for name, tc := range map[string]struct {
		strategy string
		fail     error
		want     []string
		status   string
	}{
		"compose brings the project down":  {strategy: "compose", want: []string{"docker", "compose", "-p", "edge", "down", "--remove-orphans"}, status: BuildStatusSuccess},
		"run removes the container":        {strategy: "run", want: []string{"docker", "rm", "-f", "edge"}, status: BuildStatusSuccess},
		"no strategy is a run stack":       {strategy: "", want: []string{"docker", "rm", "-f", "edge"}, status: BuildStatusSuccess},
		"a container already gone is done": {strategy: "run", fail: missing, want: []string{"docker", "rm", "-f", "edge"}, status: BuildStatusSuccess},
		"a compose failure is reported":    {strategy: "compose", fail: boom, want: []string{"docker", "compose", "-p", "edge", "down", "--remove-orphans"}, status: BuildStatusFailed},
		"a run failure is reported":        {strategy: "run", fail: boom, want: []string{"docker", "rm", "-f", "edge"}, status: BuildStatusFailed},
	} {
		t.Run(name, func(t *testing.T) {
			var calls [][]string
			cmd := func(_ context.Context, _ string, _ func(string), name string, args ...string) error {
				calls = append(calls, append([]string{name}, args...))
				return tc.fail
			}
			rf := &frameRecorder{}

			newTestExecutor(cmd).Teardown(t.Context(), "edge", tc.strategy, rf.send)

			assert.Equal(t, [][]string{tc.want}, calls)
			last := rf.last()
			assert.Equal(t, FrameStackTeardownResult, last.Type)
			assert.Equal(t, "edge", last.StackSlug)
			assert.Equal(t, tc.status, last.Status)
		})
	}
}

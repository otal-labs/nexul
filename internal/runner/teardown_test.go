package runner

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeTeardownRepo is an in-memory TeardownRepo; its rows outlive any handler built on it, like the table does.
type fakeTeardownRepo struct {
	mu                          sync.Mutex
	rows                        map[string][]Frame
	saveErr, listErr, deleteErr error
	afterDelete                 func()
}

func newFakeTeardownRepo() *fakeTeardownRepo { return &fakeTeardownRepo{rows: map[string][]Frame{}} }

func (f *fakeTeardownRepo) Save(_ context.Context, machine string, fr Frame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.rows[machine] = append(f.rows[machine], fr)
	return nil
}

func (f *fakeTeardownRepo) List(_ context.Context, machine string) ([]Frame, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]Frame(nil), f.rows[machine]...), nil
}

func (f *fakeTeardownRepo) Delete(_ context.Context, machine, slug string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	var kept []Frame
	for _, fr := range f.rows[machine] {
		if fr.StackSlug != slug {
			kept = append(kept, fr)
		}
	}
	f.rows[machine] = kept
	if f.afterDelete != nil {
		f.afterDelete()
	}
	return nil
}

func (f *fakeTeardownRepo) waiting(machine string) []Frame {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Frame(nil), f.rows[machine]...)
}

func (f *fakeTeardownRepo) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, rows := range f.rows {
		n += len(rows)
	}
	return n
}

func newTeardownHandler(repo *fakeRunnerRepo, store TeardownRepo) *Handler {
	return newTestHandler(newFakeBus(), repo, func(c *HandlerConfig) { c.Teardowns = store })
}

// openConn is a connected runner on machine; the websocket.Conn is the runner's end, which reads what the handler sends.
func openConn(t *testing.T, machine string) (*runnerConn, *websocket.Conn) {
	t.Helper()
	served, release := make(chan *websocket.Conn, 1), make(chan struct{})
	srv := wsTestServer(t, func(_ context.Context, conn *websocket.Conn) {
		served <- conn
		<-release
	})
	t.Cleanup(func() { close(release) })
	client, _, err := websocket.Dial(t.Context(), wsURL(srv), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.CloseNow() })
	return &runnerConn{id: "r-open", name: "r-open", machine: machine, ws: <-served}, client
}

// closedConn is a connected runner whose socket is gone, so every send to it fails.
func closedConn(t *testing.T, machine string) *runnerConn {
	t.Helper()
	srv := wsTestServer(t, func(context.Context, *websocket.Conn) {})
	client, _, err := websocket.Dial(t.Context(), wsURL(srv), nil)
	require.NoError(t, err)
	require.NoError(t, client.CloseNow())
	return &runnerConn{id: "r-gone", name: "r-gone", machine: machine, ws: client}
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
	store := newFakeTeardownRepo()
	h := newTeardownHandler(newFakeRunnerRepo(), store)

	t.Run("a payload that cannot be parsed is dead-lettered", func(t *testing.T) {
		err := h.handleStackDeleted(t.Context(), eventbus.Event{Payload: []byte("{")})
		require.ErrorIs(t, err, apperrs.ErrFatal)
	})
	t.Run("an unmanaged stack's containers are the owner's and stay", func(t *testing.T) {
		require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "shop", Strategy: "compose", Managed: false}))
		assert.Zero(t, store.total(), "nothing waits for a runner either")
	})
	t.Run("a stack with no machine has no runner to ask", func(t *testing.T) {
		require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Slug: "shop", Strategy: "compose", Managed: true}))
		assert.Zero(t, store.total())
	})
	t.Run("a slug docker would read as a flag is dropped, not retried", func(t *testing.T) {
		require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "-f", Strategy: "run", Managed: true}))
		assert.Zero(t, store.total())
	})
}

func TestHandler_StackDeleted_TheMachinesRunnerGetsOneTeardown(t *testing.T) {
	for name, tc := range map[string]struct{ strategy string }{
		"a compose stack": {strategy: "compose"},
		"a run stack":     {strategy: "run"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := newFakeRunnerRepo()
			store := newFakeTeardownRepo()
			h := newTeardownHandler(repo, store)
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
			require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host2", Slug: "marker", Strategy: tc.strategy, Managed: true}))
			assert.Equal(t, "marker", readTeardownFrame(t, ctx, other).StackSlug, "a runner on another machine hears only its own machine's teardown")
			assert.Zero(t, store.total())
		})
	}
}

func TestHandler_StackDeleted_AnOfflineMachineIsTornDownWhenItsRunnerConnects(t *testing.T) {
	repo := newFakeRunnerRepo()
	store := newFakeTeardownRepo()
	deleting := newTeardownHandler(repo, store)

	require.NoError(t, stackDeleted(t, deleting, StackDeletedEvent{Machine: "host1", Slug: "edge", Strategy: "run", Managed: true}), "the delete does not wait for the runner")
	require.NoError(t, stackDeleted(t, deleting, StackDeletedEvent{Machine: "host1", Slug: "shop", Strategy: "compose", Managed: true}))
	require.Len(t, store.waiting("host1"), 2)

	// A new handler on the same store is the server after a restart.
	h := newTeardownHandler(repo, store)
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	elsewhere, _, err := dialRunner(ctx, srv, repo.enrolled("r-2", "r-2", "host2"), "")
	require.NoError(t, err)
	defer func() { _ = elsewhere.CloseNow() }()
	eventually(t, 2*time.Second, func() bool { return repo.connectedCount() == 1 })
	require.Len(t, store.waiting("host1"), 2, "another machine's runner does not take them")

	conn, _, err := dialRunner(ctx, srv, repo.enrolled("r-1", "r-1", "host1"), "")
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()

	assert.Equal(t, Frame{Type: FrameStackTeardown, StackSlug: "edge", Strategy: "run"}, readTeardownFrame(t, ctx, conn))
	assert.Equal(t, Frame{Type: FrameStackTeardown, StackSlug: "shop", Strategy: "compose"}, readTeardownFrame(t, ctx, conn), "in the order they were deleted")
	eventually(t, time.Second, func() bool { return len(store.waiting("host1")) == 0 })
}

func TestHandler_StackDeleted_AFailedSendIsKeptForTheNextConnect(t *testing.T) {
	repo := newFakeRunnerRepo()
	store := newFakeTeardownRepo()
	h := newTeardownHandler(repo, store)
	gone := closedConn(t, "host1")
	h.conns[gone.id] = gone

	require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "edge", Strategy: "run", Managed: true}))

	assert.Equal(t, []Frame{{Type: FrameStackTeardown, StackSlug: "edge", Strategy: "run"}}, store.waiting("host1"))
}

func TestHandler_StackDeleted_AStoreErrorIsLoggedNotFatal(t *testing.T) {
	store := newFakeTeardownRepo()
	store.saveErr = errors.New("database is locked")
	h := newTeardownHandler(newFakeRunnerRepo(), store)

	require.NoError(t, stackDeleted(t, h, StackDeletedEvent{Machine: "host1", Slug: "edge", Strategy: "run", Managed: true}))
	assert.Zero(t, store.total())

	noStore := newTestHandler(newFakeBus(), newFakeRunnerRepo())
	require.NoError(t, stackDeleted(t, noStore, StackDeletedEvent{Machine: "host1", Slug: "edge", Strategy: "run", Managed: true}))
}

func TestHandler_FlushTeardowns_StopsAtTheFirstFailedSendAndKeepsTheRest(t *testing.T) {
	store := newFakeTeardownRepo()
	h := newTeardownHandler(newFakeRunnerRepo(), store)
	for _, slug := range []string{"edge", "shop", "api"} {
		require.NoError(t, store.Save(t.Context(), "host1", Frame{Type: FrameStackTeardown, StackSlug: slug, Strategy: "run"}))
	}
	c, runnersEnd := openConn(t, "host1")
	store.afterDelete = func() { _ = c.ws.CloseNow() }

	h.flushTeardowns(t.Context(), c)

	assert.Equal(t, "edge", readTeardownFrame(t, t.Context(), runnersEnd).StackSlug)
	waiting := store.waiting("host1")
	require.Len(t, waiting, 2, "the one that was sent is dropped")
	assert.Equal(t, []string{"shop", "api"}, []string{waiting[0].StackSlug, waiting[1].StackSlug})
}

func TestHandler_FlushTeardowns_AStoreErrorSendsNothingMore(t *testing.T) {
	t.Run("a list that fails", func(t *testing.T) {
		store := newFakeTeardownRepo()
		store.listErr = errors.New("database is locked")
		h := newTeardownHandler(newFakeRunnerRepo(), store)
		h.flushTeardowns(t.Context(), closedConn(t, "host1"))
	})
	t.Run("a delete that fails stops before the next teardown", func(t *testing.T) {
		store := newFakeTeardownRepo()
		h := newTeardownHandler(newFakeRunnerRepo(), store)
		for _, slug := range []string{"edge", "shop"} {
			require.NoError(t, store.Save(t.Context(), "host1", Frame{Type: FrameStackTeardown, StackSlug: slug, Strategy: "run"}))
		}
		store.deleteErr = errors.New("database is locked")
		c, runnersEnd := openConn(t, "host1")

		h.flushTeardowns(t.Context(), c)
		require.NoError(t, h.sendFrame(t.Context(), c, Frame{Type: FrameStackTeardown, StackSlug: "marker", Strategy: "run"}))

		assert.Equal(t, "edge", readTeardownFrame(t, t.Context(), runnersEnd).StackSlug)
		assert.Equal(t, "marker", readTeardownFrame(t, t.Context(), runnersEnd).StackSlug, "the second teardown waits until the first is dropped")
		assert.Len(t, store.waiting("host1"), 2)
	})
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

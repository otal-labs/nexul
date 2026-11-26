package shelljob

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type message struct {
	typ websocket.MessageType
	p   []byte
}

// pipeEnd is one end of an in-memory connection; closing either end ends both, as a dropped socket does.
type pipeEnd struct {
	in, out chan message
	closed  chan struct{}
	once    *sync.Once
}

func newPipe() (*pipeEnd, *pipeEnd) {
	ab, ba, closed, once := make(chan message), make(chan message), make(chan struct{}), &sync.Once{}
	return &pipeEnd{in: ba, out: ab, closed: closed, once: once}, &pipeEnd{in: ab, out: ba, closed: closed, once: once}
}

func (e *pipeEnd) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case m := <-e.in:
		return m.typ, m.p, nil
	case <-e.closed:
		return 0, nil, io.EOF
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (e *pipeEnd) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
	select {
	case e.out <- message{typ, p}:
		return nil
	case <-e.closed:
		return io.EOF
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *pipeEnd) Close() { e.once.Do(func() { close(e.closed) }) }

func (e *pipeEnd) isClosed() bool {
	select {
	case <-e.closed:
		return true
	default:
		return false
	}
}

// terminalFixture holds a set of terminals whose runner ends the test drives.
type terminalFixture struct {
	store   *fakeStore
	t       *Terminals
	runners map[string]*pipeEnd
}

func newTerminalFixture() *terminalFixture {
	store := &fakeStore{jobs: map[string]Job{}}
	return &terminalFixture{
		store:   store,
		runners: map[string]*pipeEnd{},
		t: NewTerminals(Config{
			Store: store, Event: func(Job) eventbus.OutboxEvent { return eventbus.OutboxEvent{} },
			StartError: func(err error) string { return err.Error() },
		}),
	}
}

func (f *terminalFixture) open(t *testing.T, id, computer string) {
	t.Helper()
	_, err := f.t.Open(t.Context(), Job{ID: id, ComputerID: computer, RequesterID: "alice"}, TerminalSize{Cols: 80, Rows: 24},
		func(context.Context, Job, TerminalSize) (TerminalStream, error) {
			server, runner := newPipe()
			f.runners[id] = runner
			return server, nil
		})
	require.NoError(t, err)
}

// attach relays a new viewer to id in the background and returns the viewer's end and the relay's result.
func (f *terminalFixture) attach(id, user string) (*pipeEnd, chan error) {
	viewer, server := newPipe()
	done := make(chan error, 1)
	go func() {
		done <- f.t.Attach(context.Background(), id, user, func() (TerminalStream, error) { return server, nil })
	}()
	return viewer, done
}

func TestTerminalOpen_InvalidSize_RecordsNothing(t *testing.T) {
	f := newTerminalFixture()
	_, err := f.t.Open(t.Context(), Job{ID: "t1", ComputerID: "c1"}, TerminalSize{Cols: 0, Rows: 24}, nil)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, f.store.jobs)
}

func TestTerminalOpen_RunnerRefuses_RecordsTheRefusal(t *testing.T) {
	f := newTerminalFixture()
	_, err := f.t.Open(t.Context(), Job{ID: "t1", ComputerID: "c1"}, TerminalSize{Cols: 80, Rows: 24},
		func(context.Context, Job, TerminalSize) (TerminalStream, error) {
			return nil, errors.New("commands are turned off on this computer")
		})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	job := f.store.job("t1")
	assert.Equal(t, Refused, job.Status)
	assert.Equal(t, AdapterTerminal, job.Adapter)
	assert.Equal(t, "commands are turned off on this computer", job.Error)
	assert.False(t, f.t.Live("t1"))
}

func TestTerminalOpen_RunnerFull_Refuses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newTerminalFixture()
		for i := range TerminalsAtOnce {
			f.open(t, string(rune('a'+i)), "c1")
		}
		_, err := f.t.Open(t.Context(), Job{ID: "over", ComputerID: "c1"}, TerminalSize{Cols: 80, Rows: 24}, nil)
		require.ErrorIs(t, err, apperrs.ErrConflict)
		f.open(t, "elsewhere", "c2")
		f.t.CancelWhere(func(Job) bool { return true }, "")
	})
}

func TestTerminalAttach_SomeoneElseOrTwice_FindsNoTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newTerminalFixture()
		f.open(t, "t1", "c1")
		err := f.t.Attach(t.Context(), "t1", "bob", func() (TerminalStream, error) {
			t.Fatal("accepted a socket for someone else")
			return nil, nil
		})
		require.ErrorIs(t, err, apperrs.ErrNotFound)

		viewer, done := f.attach("t1", "alice")
		synctest.Wait()
		err = f.t.Attach(t.Context(), "t1", "alice", func() (TerminalStream, error) {
			t.Fatal("accepted a second viewer")
			return nil, nil
		})
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		viewer.Close()
		require.NoError(t, <-done)
	})
}

func TestTerminalRelay_ShellExits_RecordsItsOutputAndCode(t *testing.T) {
	tests := []struct {
		name     string
		exit     string
		status   string
		exitCode int
	}{
		{"exit 0", `{"exit":0}`, Succeeded, 0},
		{"exit 3", `{"exit":3}`, Failed, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newTerminalFixture()
				f.open(t, "t1", "c1")
				runner := f.runners["t1"]
				viewer, done := f.attach("t1", "alice")
				ctx := t.Context()

				require.NoError(t, viewer.Write(ctx, websocket.MessageBinary, []byte("ls\r")))
				typ, p, err := runner.Read(ctx)
				require.NoError(t, err)
				assert.Equal(t, websocket.MessageBinary, typ)
				assert.Equal(t, "ls\r", string(p))

				require.NoError(t, viewer.Write(ctx, websocket.MessageText, []byte(`{"cols":0,"rows":5}`)))
				require.NoError(t, viewer.Write(ctx, websocket.MessageText, []byte(`{"cols":120,"rows":40}`)))
				typ, p, err = runner.Read(ctx)
				require.NoError(t, err)
				assert.Equal(t, websocket.MessageText, typ)
				assert.JSONEq(t, `{"cols":120,"rows":40}`, string(p), "a bad size never reaches the runner")

				require.NoError(t, runner.Write(ctx, websocket.MessageBinary, []byte("Desktop  Code\r\n")))
				_, p, err = viewer.Read(ctx)
				require.NoError(t, err)
				assert.Equal(t, "Desktop  Code\r\n", string(p))
				require.NoError(t, runner.Write(ctx, websocket.MessageText, []byte(tt.exit)))
				_, _, err = viewer.Read(ctx)
				require.NoError(t, err)
				runner.Close()
				require.NoError(t, <-done)

				job := f.store.job("t1")
				assert.Equal(t, tt.status, job.Status)
				require.NotNil(t, job.ExitCode)
				assert.Equal(t, tt.exitCode, *job.ExitCode)
				assert.Equal(t, "Desktop  Code\r\n", job.Output)
				assert.False(t, f.t.Live("t1"))
			})
		})
	}
}

func TestTerminalRelay_ViewerCloses_CancelsAndClosesTheRunner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newTerminalFixture()
		f.open(t, "t1", "c1")
		viewer, done := f.attach("t1", "alice")
		synctest.Wait()
		viewer.Close()
		require.NoError(t, <-done)
		assert.True(t, f.runners["t1"].isClosed())
		job := f.store.job("t1")
		assert.Equal(t, Cancelled, job.Status)
		assert.Equal(t, "the terminal was closed", job.Error)
	})
}

func TestTerminalRelay_RunnerDrops_Fails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newTerminalFixture()
		f.open(t, "t1", "c1")
		viewer, done := f.attach("t1", "alice")
		synctest.Wait()
		f.runners["t1"].Close()
		require.NoError(t, <-done)
		assert.True(t, viewer.isClosed())
		assert.Equal(t, Failed, f.store.job("t1").Status)
	})
}

func TestTerminalCancelWhere_ClosesOpenTerminals_SayingWhy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newTerminalFixture()
		f.open(t, "relayed", "c1")
		f.open(t, "waiting", "c1")
		f.open(t, "other", "c2")
		viewer, done := f.attach("relayed", "alice")
		synctest.Wait()

		f.t.CancelWhere(func(j Job) bool { return j.ComputerID == "c1" }, "commands from Nexul were turned off")
		require.NoError(t, <-done)
		assert.True(t, viewer.isClosed())
		assert.True(t, f.runners["waiting"].isClosed())
		for _, id := range []string{"relayed", "waiting"} {
			job := f.store.job(id)
			assert.Equal(t, Cancelled, job.Status, id)
			assert.Equal(t, "commands from Nexul were turned off", job.Error, id)
		}
		assert.True(t, f.t.Live("other"))
		f.t.Cancel("other", "")
		assert.Equal(t, "stopped by a person", f.store.job("other").Error)
	})
}

func TestTerminalOpen_NobodyAttaches_ClosesAfterTheWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newTerminalFixture()
		f.open(t, "t1", "c1")
		synctest.Wait()
		assert.True(t, f.t.Live("t1"))
		time.Sleep(AttachWindow)
		synctest.Wait()
		assert.True(t, f.runners["t1"].isClosed())
		job := f.store.job("t1")
		assert.Equal(t, Cancelled, job.Status)
		assert.Equal(t, "nobody opened the terminal", job.Error)
	})
}

func TestTail_KeepsTheLastMaxOutputBytes(t *testing.T) {
	var out tail
	out.write([]byte(strings.Repeat("a", MaxOutput)))
	assert.False(t, out.truncated)
	out.write([]byte("bc"))
	assert.True(t, out.truncated)
	assert.Len(t, out.b, MaxOutput)
	assert.Equal(t, "abc", string(out.b[len(out.b)-3:]))
}

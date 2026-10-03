package presence

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type fakeConn struct {
	done      chan struct{}
	closeOnce sync.Once
}

func newFakeConn() *fakeConn { return &fakeConn{done: make(chan struct{})} }

func (c *fakeConn) Done() <-chan struct{} { return c.done }

func (c *fakeConn) Close() error {
	c.closeOnce.Do(func() { close(c.done) })
	return nil
}

// dropServerSide simulates T3 closing the socket.
func (c *fakeConn) dropServerSide() { c.closeOnce.Do(func() { close(c.done) }) }

// fakeT3 records dials per computer ID and hands out fakeConns.
type fakeT3 struct {
	mu       sync.Mutex
	dials    map[string]int
	conns    map[string]*fakeConn
	dialErrs map[string]error
	dialed   chan string
}

func newFakeT3() *fakeT3 {
	return &fakeT3{dials: map[string]int{}, conns: map[string]*fakeConn{}, dialErrs: map[string]error{}, dialed: make(chan string, 64)}
}

func (f *fakeT3) hold(_ context.Context, s harness.Session) (harness.Conn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dials[s.Name]++
	f.dialed <- s.Name
	if err := f.dialErrs[s.Name]; err != nil {
		return nil, err
	}
	c := newFakeConn()
	f.conns[s.Name] = c
	return c, nil
}

func (f *fakeT3) conn(id string) *fakeConn {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conns[id]
}

func (f *fakeT3) dialCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dials[id]
}

func waitDial(t *testing.T, f *fakeT3, want string) {
	t.Helper()
	select {
	case id := <-f.dialed:
		require.Equal(t, want, id)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for dial of %s", want)
	}
}

func computer(id string) pairing.Computer {
	return pairing.Computer{ID: id, Kind: harness.KindT3Code, Name: id, ServerURL: "https://" + id + ".example.com", TokenExpiresAt: time.Now().Add(time.Hour), BearerToken: "tok-" + id}
}

type sessionsFunc struct {
	mu        sync.Mutex
	computers []pairing.Computer
}

func (s *sessionsFunc) set(cs ...pairing.Computer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.computers = cs
}

func (s *sessionsFunc) list(context.Context, string) ([]pairing.Computer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]pairing.Computer(nil), s.computers...), nil
}

func newTestKeeper(sessions *sessionsFunc, t3 *fakeT3) *Keeper {
	return New(Config{Sessions: sessions.list, Harnesses: harnesstest.Registry(&harnesstest.Client{HoldFn: t3.hold}), Linger: 20 * time.Millisecond})
}

func TestKeeper_ConnectsOnFirstBrowserSocket(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"), computer("c2"))
	t3 := newFakeT3()
	k := newTestKeeper(sessions, t3)

	k.Connected("u1")
	got := map[string]bool{}
	for range 2 {
		select {
		case id := <-t3.dialed:
			got[id] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for dials")
		}
	}
	assert.True(t, got["c1"] && got["c2"], "both computers dialed, got %v", got)
}

func TestKeeper_LastDisconnectClosesAfterLinger(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"))
	t3 := newFakeT3()
	k := newTestKeeper(sessions, t3)

	k.Connected("u1")
	k.Connected("u1") // second tab
	waitDial(t, t3, "c1")

	k.Disconnected("u1")
	conn := t3.conn("c1")
	require.NotNil(t, conn)
	select {
	case <-conn.done:
		t.Fatal("connection closed while one browser socket remained")
	case <-time.After(60 * time.Millisecond):
	}

	k.Disconnected("u1")
	select {
	case <-conn.done:
	case <-time.After(2 * time.Second):
		t.Fatal("connection not closed after last disconnect + linger")
	}
}

func TestKeeper_ReconnectWithinLingerKeepsConnection(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"))
	t3 := newFakeT3()
	k := newTestKeeper(sessions, t3)

	k.Connected("u1")
	waitDial(t, t3, "c1")
	conn := t3.conn("c1")

	k.Disconnected("u1")
	k.Connected("u1") // page refresh inside the linger window
	time.Sleep(80 * time.Millisecond)
	select {
	case <-conn.done:
		t.Fatal("connection dropped across a page refresh")
	default:
	}
	assert.Equal(t, 1, t3.dialCount("c1"), "no redial across a refresh")
}

func TestKeeper_RedialsWhenServerDrops(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"))
	t3 := newFakeT3()
	k := newTestKeeper(sessions, t3)

	k.Connected("u1")
	waitDial(t, t3, "c1")
	t3.conn("c1").dropServerSide()
	waitDial(t, t3, "c1") // redial after backoff
}

func TestKeeper_RefreshPicksUpNewComputerAndDropsDeleted(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"))
	t3 := newFakeT3()
	k := newTestKeeper(sessions, t3)

	k.Connected("u1")
	waitDial(t, t3, "c1")
	c1 := t3.conn("c1")

	sessions.set(computer("c2")) // c1 deleted, c2 paired
	k.Refresh("u1")
	waitDial(t, t3, "c2")
	select {
	case <-c1.done:
	case <-time.After(2 * time.Second):
		t.Fatal("deleted computer's connection not closed")
	}
}

func TestKeeper_UnauthorizedStopsRedialUntilRefresh(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"))
	t3 := newFakeT3()
	t3.mu.Lock()
	t3.dialErrs["c1"] = apperrs.ErrUnauthorized
	t3.mu.Unlock()
	k := newTestKeeper(sessions, t3)

	k.Connected("u1")
	waitDial(t, t3, "c1")
	time.Sleep(80 * time.Millisecond)
	assert.Equal(t, 1, t3.dialCount("c1"), "no redial after an unauthorized dial")

	// Re-pair: token works now, Refresh restarts the loop.
	t3.mu.Lock()
	delete(t3.dialErrs, "c1")
	t3.mu.Unlock()
	k.Refresh("u1")
	waitDial(t, t3, "c1")
}

func TestKeeper_RefreshWhileOfflineIsNoop(t *testing.T) {
	t.Parallel()
	sessions := &sessionsFunc{}
	sessions.set(computer("c1"))
	t3 := newFakeT3()
	k := newTestKeeper(sessions, t3)

	k.Refresh("u1")
	select {
	case id := <-t3.dialed:
		t.Fatalf("dialed %s for an offline user", id)
	case <-time.After(60 * time.Millisecond):
	}
}

func TestKeeper_OnlineFollowsFirstSocketAndLingerEnd(t *testing.T) {
	t.Parallel()
	k := New(Config{Sessions: (&sessionsFunc{}).list, Linger: 300 * time.Millisecond})
	changed := make(chan string, 8)
	k.SetOnlineChanged(func(userID string) { changed <- userID })

	k.Connected("u1")
	require.Equal(t, "u1", <-changed, "first socket announces the user online")
	k.Connected("u1") // second tab
	k.Disconnected("u1")
	assert.Empty(t, changed, "a second tab or one of two closing is not a presence change")
	assert.Equal(t, map[string]bool{"u1": true}, k.Online())

	k.Disconnected("u1")
	assert.True(t, k.Online()["u1"], "online through the linger window, so a refresh never reads as offline")
	select {
	case id := <-changed:
		assert.Equal(t, "u1", id)
	case <-time.After(2 * time.Second):
		t.Fatal("going offline after the linger window was not announced")
	}
	assert.Empty(t, k.Online())
}

// dialLevels records the level of every failed-dial log line.
type dialLevels struct {
	mu  sync.Mutex
	got []slog.Level
}

func (d *dialLevels) Enabled(context.Context, slog.Level) bool { return true }
func (d *dialLevels) WithAttrs([]slog.Attr) slog.Handler       { return d }
func (d *dialLevels) WithGroup(string) slog.Handler            { return d }

func (d *dialLevels) Handle(_ context.Context, r slog.Record) error {
	if r.Message == "presence: dial failed, retrying" {
		d.mu.Lock()
		d.got = append(d.got, r.Level)
		d.mu.Unlock()
	}
	return nil
}

func (d *dialLevels) levels() []slog.Level {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.got)
}

func TestKeeper_ProtocolRefusal_KeepsRetryingButWarnsOncePerRefusal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sessions := &sessionsFunc{}
		sessions.set(computer("c1"))
		t3 := newFakeT3()
		refuse := func(on bool) {
			t3.mu.Lock()
			defer t3.mu.Unlock()
			delete(t3.dialErrs, "c1")
			if on {
				t3.dialErrs["c1"] = harness.ProtocolRefusal("T3 Code on c1 went back to its old orchestrator; Nexul only moves forward. Update T3 Code there.")
			}
		}
		logs := &dialLevels{}
		k := New(Config{Sessions: sessions.list, Harnesses: harnesstest.Registry(&harnesstest.Client{HoldFn: t3.hold}), Logger: slog.New(logs), Linger: time.Millisecond})

		refuse(true)
		k.Connected("u1")
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		refused := t3.dialCount("c1")
		assert.Greater(t, refused, 3, "keeps retrying, so updating T3 Code recovers on its own")
		levels := logs.levels()
		require.Len(t, levels, refused)
		assert.Equal(t, slog.LevelWarn, levels[0])
		assert.Equal(t, slices.Repeat([]slog.Level{slog.LevelDebug}, refused-1), levels[1:])

		refuse(false)
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		require.Equal(t, map[string]string{"c1": StateConnected}, k.Status("u1"), "the update was picked up without a re-pair")
		refuse(true)
		t3.conn("c1").dropServerSide()
		time.Sleep(time.Minute)
		synctest.Wait()
		assert.Equal(t, slog.LevelWarn, logs.levels()[refused], "going back again after a connection is a new refusal, said once")

		k.Disconnected("u1")
		time.Sleep(time.Second)
	})
}

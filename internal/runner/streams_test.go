package runner

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// streamMux serves a handler's two runner routes the way server/cmd mounts them.
func streamMux(h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/ws/runner", h)
	mux.HandleFunc("GET /api/runners/streams/{id}", h.ServeStream)
	return mux
}

// streamFixture is a runner handler on a real HTTP server, with fake runners the test drives frame by frame.
type streamFixture struct {
	h    *Handler
	repo *fakeRunnerRepo
	srv  *httptest.Server
}

func newStreamFixture(t *testing.T) *streamFixture {
	t.Helper()
	repo := newFakeRunnerRepo()
	h := newTestHandler(newFakeBus(), repo)
	srv := httptest.NewServer(streamMux(h))
	t.Cleanup(srv.Close)
	return &streamFixture{h: h, repo: repo, srv: srv}
}

// control connects a fake runner holding credential and returns every frame but heartbeats the server sends it.
func (f *streamFixture) control(t *testing.T, credential string) (<-chan Frame, *websocket.Conn) {
	t.Helper()
	n := len(f.h.Runners())
	conn, _, err := dialRunner(t.Context(), f.srv, credential, "/ws/runner")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == n+1 })
	frames := make(chan Frame, 16)
	go func() {
		for {
			var fr Frame
			if err := wsjson.Read(context.Background(), conn, &fr); err != nil {
				return
			}
			frames <- fr
		}
	}()
	return frames, conn
}

func (f *streamFixture) openStream(ctx context.Context, credential, id string) (*websocket.Conn, *http.Response, error) {
	return websocket.Dial(ctx, wsURL(f.srv)+"/api/runners/streams/"+id, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + credential}},
	})
}

type dialed struct {
	conn net.Conn
	err  error
}

func dialComputer(ctx context.Context, h *Handler, computerID string) <-chan dialed {
	out := make(chan dialed, 1)
	go func() {
		conn, err := h.DialComputer(ctx, computerID)
		out <- dialed{conn, err}
	}()
	return out
}

func nextFrame(t *testing.T, frames <-chan Frame) Frame {
	t.Helper()
	select {
	case f := <-frames:
		return f
	case <-time.After(5 * time.Second):
		t.Fatal("no frame within 5s")
		return Frame{}
	}
}

func TestDialComputer_RefusesAStreamFromAnotherRunner(t *testing.T) {
	f := newStreamFixture(t)
	alice := f.repo.personal("r-alice", "u-alice", "c-laptop")
	bob := f.repo.personal("r-bob", "u-bob", "c-desktop")
	frames, _ := f.control(t, alice)
	ctx := t.Context()

	result := dialComputer(ctx, f.h, "c-laptop")
	dial := nextFrame(t, frames)
	require.Equal(t, FrameHarnessDial, dial.Type)

	_, resp, err := f.openStream(ctx, bob, dial.ID)
	require.Error(t, err, "another runner's credential never claims the stream")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	ws, _, err := f.openStream(ctx, alice, dial.ID)
	require.NoError(t, err, "the refusal left the id for the runner it was issued to")
	got := <-result
	require.NoError(t, got.err)
	// The runner's end closes first, so the harness end's close handshake does not wait on a peer that stopped reading.
	defer func() { _ = got.conn.Close() }()
	defer func() { _ = ws.CloseNow() }()

	_, err = got.conn.Write([]byte("GET / HTTP/1.1\r\n"))
	require.NoError(t, err)
	_, data, err := ws.Read(ctx)
	require.NoError(t, err)
	assert.Equal(t, "GET / HTTP/1.1\r\n", string(data), "bytes the harness writes reach the runner's end")
}

func TestDialComputer_StreamIDIsSingleUse(t *testing.T) {
	f := newStreamFixture(t)
	alice := f.repo.personal("r-alice", "u-alice", "c-laptop")
	frames, _ := f.control(t, alice)
	ctx := t.Context()

	result := dialComputer(ctx, f.h, "c-laptop")
	dial := nextFrame(t, frames)
	ws, _, err := f.openStream(ctx, alice, dial.ID)
	require.NoError(t, err)
	got := <-result
	require.NoError(t, got.err)
	defer func() { _ = got.conn.Close() }()
	defer func() { _ = ws.CloseNow() }()

	_, resp, err := f.openStream(ctx, alice, dial.ID)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "the same runner cannot open a second socket on a used id")
}

func TestDialComputer_ExpiredStreamIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeRunnerRepo()
		h := newTestHandler(newFakeBus(), repo, func(c *HandlerConfig) { c.HeartbeatInterval = time.Hour })
		client := serveMem(t, streamMux(h))
		alice := repo.personal("r-alice", "u-alice", "c-laptop")
		ctx := t.Context()
		control := memDial(t, ctx, client, "ws://nexul.test/ws/runner", alice)
		synctest.Wait()

		result := dialComputer(ctx, h, "c-laptop")
		var dial Frame
		require.NoError(t, wsjson.Read(ctx, control, &dial))
		require.Equal(t, FrameHarnessDial, dial.Type)

		time.Sleep(streamTTL + time.Second)
		got := <-result
		require.ErrorIs(t, got.err, apperrs.ErrRetryable, "DialComputer gave up after the stream's life")
		_, resp, err := websocket.Dial(ctx, "ws://nexul.test/api/runners/streams/"+dial.ID, &websocket.DialOptions{
			HTTPClient: client, HTTPHeader: http.Header{"Authorization": {"Bearer " + alice}},
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "a runner arriving after 10 seconds is refused")
		_ = control.CloseNow()
	})
}

func TestDialComputer_CapsStreamsPerRunner(t *testing.T) {
	f := newStreamFixture(t)
	frames, _ := f.control(t, f.repo.personal("r-alice", "u-alice", "c-laptop"))
	ctx, cancel := context.WithCancel(t.Context())

	pending := make([]<-chan dialed, maxStreams)
	for i := range pending {
		pending[i] = dialComputer(ctx, f.h, "c-laptop")
		nextFrame(t, frames)
	}
	_, err := f.h.DialComputer(t.Context(), "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrRetryable)
	assert.Contains(t, err.Error(), "already has 32 streams open")

	cancel()
	for _, p := range pending {
		require.ErrorIs(t, (<-p).err, context.Canceled)
	}
}

func TestDialComputer_RevokingTheRunnerClosesItsStreams(t *testing.T) {
	f := newStreamFixture(t)
	alice := f.repo.personal("r-alice", "u-alice", "c-laptop")
	frames, _ := f.control(t, alice)
	ctx := t.Context()

	result := dialComputer(ctx, f.h, "c-laptop")
	ws, _, err := f.openStream(ctx, alice, nextFrame(t, frames).ID)
	require.NoError(t, err)
	defer func() { _ = ws.CloseNow() }()
	got := <-result
	require.NoError(t, got.err)
	defer func() { _ = got.conn.Close() }()

	f.h.Uninstall(ctx, "r-alice")

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, _, err = ws.Read(readCtx)
	require.Error(t, err)
	assert.NotErrorIs(t, err, context.DeadlineExceeded, "the server closed the runner's end of the stream")
	_, err = io.ReadAll(got.conn)
	require.Error(t, err, "and the harness end with it")
}

func TestDialComputer_ComputerWithoutAConnectedRunnerIsRetryable(t *testing.T) {
	f := newStreamFixture(t)
	f.repo.personal("r-alice", "u-alice", "c-laptop")
	f.control(t, f.repo.enrolled("r-build", "build-box", "prod"))

	for _, computer := range []string{"c-laptop", "c-unknown", ""} {
		_, err := f.h.DialComputer(t.Context(), computer)
		require.ErrorIs(t, err, apperrs.ErrRetryable, "computer %q", computer)
	}
}

func TestDispatch_NeverPicksAPersonalRunner(t *testing.T) {
	f := newStreamFixture(t)
	req := DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Repo: "org/app", Ref: "main"}
	require.NoError(t, f.h.handleRequest(t.Context(), eventbus.Event{Payload: mustJSON(t, req)}))
	_, _ = f.control(t, f.repo.personal("r-alice", "u-alice", "c-laptop"))
	require.NoError(t, f.h.handleRequest(t.Context(), eventbus.Event{Payload: mustJSON(t, DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", Image: "api:1"})}))

	assert.Len(t, f.h.Queue(), 2, "a job with no target machine stays queued while only a personal runner is connected")
	assert.Nil(t, f.h.connNamed("computer-r-alice"))
	assert.Nil(t, f.h.connOnMachine(""))

	frames, _ := f.control(t, f.repo.enrolled("r-build", "build-box", "prod"))
	assert.Equal(t, FrameAssignBuild, nextFrame(t, frames).Type, "a deploy runner takes the same job")
}

func TestPersonalRunner_RefusesDeployFrames(t *testing.T) {
	sent := []Frame{
		{Type: FrameAssignBuild, ID: "b1", Repo: "org/app", Ref: "main"},
		{Type: FrameAssignDeploy, ID: "d1", Service: "api", Image: "api:1"},
		{Type: FrameAssignUpgrade, ID: "u1", Version: "v1.2.3"},
		{Type: FrameJoinNetworks, GatewayContainer: "gw", JoinNetworks: []string{"shop_default"}},
		{Type: FrameDiscover, ID: "x1"},
		{Type: FrameLogsRequest, ID: "l1", Container: "shop-web-1", Tail: 10},
	}
	const reason = "a personal runner does not run this"
	want := []Frame{
		{Type: FrameDeployResult, ID: "b1", Status: DeployStatusFailed, Error: reason},
		{Type: FrameDeployResult, ID: "d1", Status: DeployStatusFailed, Error: reason},
		{Type: FrameUpgradeResult, ID: "u1", Status: UpgradeStatusFailed, Error: reason},
		{Type: FrameJoinNetworksResult, GatewayContainer: "gw", Status: BuildStatusFailed, Error: reason},
		{Type: FrameDiscoverResult, ID: "x1", Error: reason},
		{Type: FrameLogsEnd, ID: "l1", Error: reason},
	}
	answers := make(chan []Frame, 1)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for _, f := range sent {
			_ = wsjson.Write(ctx, conn, f)
		}
		var got []Frame
		for len(got) < len(sent) {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type != FrameHeartbeat && f.Type != FrameFacts {
				got = append(got, f)
			}
		}
		answers <- got
	})
	exec := &fakeExecutor{}
	c := newTestClient(wsURL(srv), exec)
	c.cfg.Personal = true
	cancel, done := runClient(t, c)

	assert.Equal(t, want, <-answers, "each frame is answered as failed, in order, by the read loop itself")
	cancel()
	<-done
	assert.Empty(t, exec.buildIDs())
	assert.Empty(t, exec.deploys)
	assert.Empty(t, exec.upgrades)
	assert.Empty(t, exec.joinCalls)
}

// dialRecorder counts every connection the runner attempts, and answers each with the next conn in line.
type dialRecorder struct {
	mu    sync.Mutex
	addrs []string
	next  []func() (net.Conn, error)
}

func (d *dialRecorder) dial(_ context.Context, _, addr string) (net.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.addrs = append(d.addrs, addr)
	if len(d.next) == 0 {
		return nil, &net.OpError{Op: "dial", Err: os.ErrDeadlineExceeded}
	}
	conn := d.next[0]
	d.next = d.next[1:]
	return conn()
}

func (d *dialRecorder) dialed() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.addrs...)
}

func writeRuntimeFile(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "userdata"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "userdata", "server-runtime.json"), []byte(body), 0o600))
	return home
}

// refusedDial asks a personal runner whose T3 Code home is home for one stream and returns its refusal and dials.
func refusedDial(t *testing.T, home string) (Frame, []string) {
	t.Helper()
	refused := make(chan Frame, 1)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameHarnessDial, ID: "s1"})
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameHarnessDialRefused {
				refused <- f
				return
			}
		}
	})
	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Personal = true
	c.cfg.T3Home = home
	rec := &dialRecorder{}
	c.dial = rec.dial
	cancel, done := runClient(t, c)
	got := <-refused
	cancel()
	<-done
	return got, rec.dialed()
}

func TestPersonalRunner_NeverDialsAnAddressThatIsNotLoopback(t *testing.T) {
	got, dialed := refusedDial(t, writeRuntimeFile(t, `{"version":1,"pid":4242,"host":"192.168.1.20","port":47180,"origin":"http://192.168.1.20:47180"}`))
	assert.Equal(t, "s1", got.ID)
	assert.Equal(t, "T3 Code listens on 192.168.1.20, which is not a loopback address", got.Error)
	assert.Empty(t, dialed, "no connection was attempted")
}

// A T3 Code home other than the default never falls back to 3773, where another T3 Code may be listening.
func TestPersonalRunner_CustomHomeWithoutARuntimeFileIsRefused(t *testing.T) {
	home := t.TempDir()
	got, dialed := refusedDial(t, home)
	assert.Equal(t, "T3 Code isn't running in "+home, got.Error)
	assert.Empty(t, dialed, "no connection was attempted")
}

func TestT3Address(t *testing.T) {
	tests := []struct {
		name        string
		runtime     string
		defaultHome bool
		want        string
		wantErr     string
	}{
		{name: "the default home without a runtime file is the default port", defaultHome: true, want: "127.0.0.1:3773"},
		{name: "another home without a runtime file is refused", wantErr: "T3 Code isn't running in"},
		{name: "loopback", runtime: `{"host":"127.0.0.1","port":47180}`, want: "127.0.0.1:47180"},
		{name: "no host", runtime: `{"port":47180}`, want: "127.0.0.1:47180"},
		{name: "ipv4 wildcard", runtime: `{"host":"0.0.0.0","port":47180}`, want: "127.0.0.1:47180"},
		{name: "ipv6 wildcard", runtime: `{"host":"::","port":47180}`, want: "[::1]:47180"},
		{name: "ipv6 loopback", runtime: `{"host":"[::1]","port":47180}`, want: "[::1]:47180"},
		{name: "localhost", runtime: `{"host":"localhost","port":47180}`, want: "127.0.0.1:47180"},
		{name: "lan address", runtime: `{"host":"10.0.0.5","port":47180}`, wantErr: "not a loopback address"},
		{name: "hostname", runtime: `{"host":"laptop.example.com","port":47180}`, wantErr: "not a loopback address"},
		{name: "no port", runtime: `{"host":"127.0.0.1"}`, wantErr: "names port 0"},
		{name: "garbage", runtime: `{`, wantErr: "decode T3 Code's runtime file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			if tt.runtime != "" {
				home = writeRuntimeFile(t, tt.runtime)
			}
			defaultHome := filepath.Join(t.TempDir(), ".t3")
			if tt.defaultHome {
				defaultHome = home + "/"
			}
			got, err := t3Address(home, defaultHome)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// panickyConn stands in for a T3 Code connection whose read panics mid-copy.
type panickyConn struct{ net.Conn }

func (panickyConn) Read([]byte) (int, error) { panic("read exploded") }

// echoConn is one end of a pipe whose other end echoes every byte back until it closes.
func echoConn() (net.Conn, error) {
	local, remote := net.Pipe()
	go func() {
		_, _ = io.Copy(remote, remote)
		_ = remote.Close()
	}()
	return local, nil
}

func TestRunner_APanicInOneStreamLeavesTheOthersRunning(t *testing.T) {
	f := newStreamFixture(t)
	alice := f.repo.personal("r-alice", "u-alice", "c-laptop")
	c := NewClient(ClientConfig{
		URL: wsURL(f.srv) + "/ws/runner", StreamURL: wsURL(f.srv) + "/api/runners/streams", Credential: alice, Name: "computer-r-alice",
		Logger: testLogger(), Executor: &fakeExecutor{}, HeartbeatInterval: 50 * time.Millisecond, Personal: true,
		T3Home: writeRuntimeFile(t, `{"host":"127.0.0.1","port":47180}`),
	})
	panicky := func() (net.Conn, error) {
		a, b := net.Pipe()
		_ = b.Close()
		return panickyConn{a}, nil
	}
	c.dial = (&dialRecorder{next: []func() (net.Conn, error){panicky, echoConn, echoConn}}).dial
	cancel, done := runClient(t, c)
	eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })
	ctx := t.Context()

	broken, err := f.h.DialComputer(ctx, "c-laptop")
	require.NoError(t, err)
	_, err = io.ReadAll(broken)
	if err != nil {
		require.ErrorContains(t, err, "EOF", "the stream whose copy panicked is closed, with or without its close frame")
	}
	_ = broken.Close()

	for range 2 {
		conn, err := f.h.DialComputer(ctx, "c-laptop")
		require.NoError(t, err, "the control connection still answers harness_dial")
		_, err = conn.Write([]byte("ping"))
		require.NoError(t, err)
		buf := make([]byte, 4)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "ping", string(buf), "a stream opened after the panic carries bytes both ways")
		_ = conn.Close()
	}
	cancel()
	<-done
}

func TestStream_IdleWebSocketStaysOpenForThirtyMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeRunnerRepo()
		h := newTestHandler(newFakeBus(), repo, func(c *HandlerConfig) { c.HeartbeatInterval = time.Hour })
		nexul := serveMem(t, streamMux(h))
		t3 := serveMem(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = ws.CloseNow() }()
			for {
				typ, data, err := ws.Read(r.Context())
				if err != nil {
					return
				}
				if err := ws.Write(r.Context(), typ, data); err != nil {
					return
				}
			}
		}))
		alice := repo.personal("r-alice", "u-alice", "c-laptop")
		ctx := t.Context()
		control := memDial(t, ctx, nexul, "ws://nexul.test/ws/runner", alice)
		synctest.Wait()
		runner := NewClient(ClientConfig{Logger: testLogger()})

		var spliced sync.WaitGroup
		spliced.Go(func() {
			var dial Frame
			require.NoError(t, wsjson.Read(ctx, control, &dial))
			stream := memDial(t, ctx, nexul, "ws://nexul.test/api/runners/streams/"+dial.ID, alice)
			local, err := t3.Transport.(*http.Transport).DialContext(ctx, "tcp", "t3.test:3773")
			require.NoError(t, err)
			runner.splice(dial.ID, websocket.NetConn(context.Background(), stream, websocket.MessageBinary), local)
		})

		var dials atomic.Int32
		harness := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dials.Add(1)
			return h.DialComputer(ctx, "c-laptop")
		}}}
		session, _, err := websocket.Dial(ctx, "ws://c-laptop.nexul-computer.invalid/ws", &websocket.DialOptions{HTTPClient: harness})
		require.NoError(t, err)

		time.Sleep(30 * time.Minute)
		synctest.Wait()

		require.NoError(t, session.Write(ctx, websocket.MessageText, []byte(`{"_tag":"Ping"}`)), "the socket through the runner is still open")
		_, data, err := session.Read(ctx)
		require.NoError(t, err)
		assert.JSONEq(t, `{"_tag":"Ping"}`, string(data))
		assert.Equal(t, int32(1), dials.Load())

		_ = session.CloseNow()
		spliced.Wait()
		_ = control.CloseNow()
	})
}

// serveMem serves handler over in-memory pipes, so it runs inside a synctest bubble, and returns a client for it.
func serveMem(t *testing.T, handler http.Handler) *http.Client {
	t.Helper()
	l := &memListener{conns: make(chan net.Conn), closed: make(chan struct{})}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	var served sync.WaitGroup
	served.Go(func() { _ = srv.Serve(l) })
	transport := &http.Transport{DialContext: l.dial}
	t.Cleanup(func() {
		transport.CloseIdleConnections()
		_ = srv.Close()
		served.Wait()
	})
	return &http.Client{Transport: transport}
}

func memDial(t *testing.T, ctx context.Context, client *http.Client, url, credential string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Authorization": {"Bearer " + credential}}})
	require.NoError(t, err)
	return conn
}

type memListener struct {
	conns  chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func (l *memListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *memListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *memListener) Addr() net.Addr { return memAddr{} }

func (l *memListener) dial(ctx context.Context, _, _ string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

type memAddr struct{}

func (memAddr) Network() string { return "mem" }
func (memAddr) String() string  { return "mem" }

package runner

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func eventually(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

// fakeRunnerRepo is an in-memory runner.Repo for handler tests.
type fakeRunnerRepo struct {
	mu         sync.Mutex
	runners    map[string]*Runner
	heartbeats int
	createErr  error
	listErr    error
	secret     string
	secretErr  error
}

// newFakeRunnerRepo defaults the secret to "s3cr3t" so every existing test
// that dials with that token keeps working now that auth reads it from the
// repo instead of a fixed HandlerConfig field.
func newFakeRunnerRepo() *fakeRunnerRepo {
	return &fakeRunnerRepo{runners: map[string]*Runner{}, secret: "s3cr3t"}
}

func (f *fakeRunnerRepo) Create(_ context.Context, r *Runner) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.runners[r.ID] = r
	return nil
}

func (f *fakeRunnerRepo) GetByID(_ context.Context, id string) (*Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runners[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	// Copies, like the SQLite repo: the handler mutates rows through
	// Heartbeat/SetConnected/SetVersion while tests read them.
	cp := *r
	return &cp, nil
}

func (f *fakeRunnerRepo) List(context.Context) ([]*Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*Runner, 0, len(f.runners))
	for _, r := range f.runners {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRunnerRepo) Heartbeat(_ context.Context, id string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runners[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	r.LastSeen = at
	f.heartbeats++
	return nil
}

func (f *fakeRunnerRepo) SetConnected(ctx context.Context, id string, connected bool) error {
	// Mirrors the SQLite repo, whose BeginTx refuses a cancelled ctx before the write starts.
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runners[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	r.Connected = connected
	return nil
}

func (f *fakeRunnerRepo) SetVersion(_ context.Context, id string, version string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runners[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	r.Version = version
	return nil
}

func (f *fakeRunnerRepo) SetMachine(_ context.Context, id string, machineID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runners[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	r.MachineID = machineID
	return nil
}

func (f *fakeRunnerRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.runners[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.runners, id)
	return nil
}

func (f *fakeRunnerRepo) Secret(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.secretErr != nil {
		return "", f.secretErr
	}
	return f.secret, nil
}

func (f *fakeRunnerRepo) SetSecret(_ context.Context, secret string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.secret = secret
	return nil
}

func (f *fakeRunnerRepo) connectedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.runners {
		if r.Connected {
			n++
		}
	}
	return n
}

// fakeMachineRepo is an in-memory runner.MachineRepo for handler/usecase tests.
type fakeMachineRepo struct {
	mu       sync.Mutex
	machines map[string]*Machine
}

func newFakeMachineRepo() *fakeMachineRepo {
	return &fakeMachineRepo{machines: map[string]*Machine{}}
}

func (f *fakeMachineRepo) Create(_ context.Context, m *Machine) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *m
	f.machines[m.ID] = &cp
	return nil
}

func (f *fakeMachineRepo) Get(_ context.Context, id string) (*Machine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.machines[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeMachineRepo) GetByName(_ context.Context, name string) (*Machine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.machines {
		if m.Name == name {
			cp := *m
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeMachineRepo) List(_ context.Context) ([]*Machine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*Machine, 0, len(f.machines))
	for _, m := range f.machines {
		cp := *m
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeMachineRepo) Rename(_ context.Context, id, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.machines[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	m.Name = name
	return nil
}

func (f *fakeMachineRepo) SetStackRoot(_ context.Context, id, stackRoot string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.machines[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	m.StackRoot = stackRoot
	return nil
}

func (f *fakeMachineRepo) Touch(_ context.Context, id, reportedHostname string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.machines[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	m.ReportedHostname = reportedHostname
	m.LastSeen = at
	return nil
}

// fakeUpgradeRepo is an in-memory runner.UpgradeRepo for usecase/handler tests.
type fakeUpgradeRepo struct {
	mu        sync.Mutex
	upgrades  map[string]*Upgrade
	createErr error
}

func newFakeUpgradeRepo() *fakeUpgradeRepo {
	return &fakeUpgradeRepo{upgrades: map[string]*Upgrade{}}
}

func (f *fakeUpgradeRepo) Create(_ context.Context, u *Upgrade) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	cp := *u
	f.upgrades[u.ID] = &cp
	return nil
}

func (f *fakeUpgradeRepo) GetByID(_ context.Context, id string) (*Upgrade, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.upgrades[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

// Latest returns the record with the newest CreatedAt; ties (same test-fixture timestamp) return either, which
// tests avoid by giving fixtures distinct timestamps.
func (f *fakeUpgradeRepo) Latest(_ context.Context) (*Upgrade, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var latest *Upgrade
	for _, u := range f.upgrades {
		if latest == nil || u.CreatedAt.After(latest.CreatedAt) {
			latest = u
		}
	}
	if latest == nil {
		return nil, apperrs.ErrNotFound
	}
	cp := *latest
	return &cp, nil
}

func (f *fakeUpgradeRepo) ListUnresolved(_ context.Context) ([]*Upgrade, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Upgrade
	for _, u := range f.upgrades {
		if u.unresolved() {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeUpgradeRepo) SetStatus(_ context.Context, id, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.upgrades[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	u.Status = status
	u.Error = errMsg
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// fakeBus records published events and dispatches to subscribed handlers
// synchronously, so handler tests are deterministic.
type fakeBus struct {
	mu         sync.Mutex
	handlers   map[string][]eventbus.Handler
	published  []eventbus.Event
	publishErr error
}

func newFakeBus() *fakeBus {
	return &fakeBus{handlers: map[string][]eventbus.Handler{}}
}

func (f *fakeBus) Publish(ctx context.Context, topic string, payload any) error {
	f.mu.Lock()
	if f.publishErr != nil {
		f.mu.Unlock()
		return f.publishErr
	}
	data, err := json.Marshal(payload)
	if err != nil {
		f.mu.Unlock()
		return err
	}
	ev := eventbus.Event{Topic: topic, Payload: data, Timestamp: time.Now()}
	f.published = append(f.published, ev)
	subs := append([]eventbus.Handler(nil), f.handlers[topic]...)
	f.mu.Unlock()
	// Dispatch outside the lock: a handler that publishes back (e.g. Cancel
	// reporting deploy.status_changed) must not re-enter the mutex.
	for _, h := range subs {
		if err := h(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeBus) Subscribe(_ context.Context, topic string, h eventbus.Handler) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[topic] = append(f.handlers[topic], h)
	return nil
}

func (f *fakeBus) handlerCount(topic string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.handlers[topic])
}

func (f *fakeBus) topicEvents(topic string) []eventbus.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []eventbus.Event
	for _, ev := range f.published {
		if ev.Topic == topic {
			out = append(out, ev)
		}
	}
	return out
}

// fakeEnvLookup is a test double for EnvLookup, keyed by service name.
type fakeEnvLookup struct {
	values map[string]map[string]string
	err    error
}

func (f *fakeEnvLookup) ResolveEnv(_ context.Context, service string) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.values[service], nil
}

// fakeExecutor is the client-side Executor whose Build/Deploy behavior is
// pluggable per test. It records the requests it received.
type fakeExecutor struct {
	mu           sync.Mutex
	builds       []DeployRequestedEvent
	deploys      []DeployRequestedEvent
	upgrades     []Frame
	joinCalls    []joinNetworksCall
	buildFn      func(ctx context.Context, req DeployRequestedEvent, send func(Frame))
	deployFn     func(ctx context.Context, req DeployRequestedEvent, send func(Frame))
	discoverFn   func(ctx context.Context) (DiscoverReport, error)
	joinNetworks func(ctx context.Context, gatewayContainer string, networks []string, send func(Frame))
	upgradeFn    func(ctx context.Context, req Frame, send func(Frame) error) error
}

func (f *fakeExecutor) Discover(ctx context.Context) (DiscoverReport, error) {
	f.mu.Lock()
	fn := f.discoverFn
	f.mu.Unlock()
	if fn == nil {
		return DiscoverReport{}, nil
	}
	return fn(ctx)
}

type joinNetworksCall struct {
	GatewayContainer string
	Networks         []string
}

func (f *fakeExecutor) Build(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	f.mu.Lock()
	f.builds = append(f.builds, req)
	fn := f.buildFn
	f.mu.Unlock()
	if fn == nil {
		fn = succeedBuild
	}
	fn(ctx, req, send)
}

func (f *fakeExecutor) Deploy(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	f.mu.Lock()
	f.deploys = append(f.deploys, req)
	fn := f.deployFn
	f.mu.Unlock()
	if fn == nil {
		fn = succeedDeploy
	}
	fn(ctx, req, send)
}

func (f *fakeExecutor) JoinNetworks(ctx context.Context, gatewayContainer string, networks []string, send func(Frame)) {
	f.mu.Lock()
	f.joinCalls = append(f.joinCalls, joinNetworksCall{GatewayContainer: gatewayContainer, Networks: networks})
	fn := f.joinNetworks
	f.mu.Unlock()
	if fn == nil {
		send(Frame{Type: FrameJoinNetworksResult, GatewayContainer: gatewayContainer, Status: BuildStatusSuccess})
		return
	}
	fn(ctx, gatewayContainer, networks, send)
}

func (f *fakeExecutor) Upgrade(ctx context.Context, req Frame, send func(Frame) error) error {
	f.mu.Lock()
	f.upgrades = append(f.upgrades, req)
	fn := f.upgradeFn
	f.mu.Unlock()
	if fn == nil {
		return send(Frame{Type: FrameUpgradeResult, ID: req.ID, Status: UpgradeStatusStarted, Log: "helper-id"})
	}
	return fn(ctx, req, send)
}

func (f *fakeExecutor) buildIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.builds))
	for _, b := range f.builds {
		out = append(out, b.ID)
	}
	return out
}

func succeedBuild(_ context.Context, req DeployRequestedEvent, send func(Frame)) {
	// A repo-driven build job ends with a deploy_result: the runner builds
	// where it deploys, so build_result is a phase terminal and the deploy
	// result frees the runner.
	send(Frame{Type: FrameBuildProgress, ID: req.ID, Step: 1, Total: 1})
	send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusSuccess})
	send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusHealthy})
}

func succeedDeploy(_ context.Context, req DeployRequestedEvent, send func(Frame)) {
	send(Frame{Type: FrameDeployProgress, ID: req.ID, Phase: DeployPhaseHealthy})
	send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusHealthy})
}

// wsTestServer accepts WebSocket connections and runs serve for each one.
func wsTestServer(t *testing.T, serve func(ctx context.Context, conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// The handler returns immediately, which cancels r.Context(); the
		// connection itself stays alive, so serve runs on an independent ctx.
		go func() {
			defer func() { _ = conn.CloseNow() }() // server-side handler goroutine outlives the test; the client usually already closed this conn
			serve(context.Background(), conn)
		}()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(srv *httptest.Server) string {
	return "ws" + srv.URL[len("http"):]
}

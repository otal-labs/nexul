package runner_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

// syncBus delivers each event to its subscribers before Publish returns.
type syncBus struct {
	mu       sync.Mutex
	handlers map[string][]eventbus.Handler
}

func (b *syncBus) Publish(ctx context.Context, topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	b.mu.Lock()
	hs := append([]eventbus.Handler(nil), b.handlers[topic]...)
	b.mu.Unlock()
	for _, h := range hs {
		if err := h(ctx, eventbus.Event{Topic: topic, Payload: data}); err != nil {
			return err
		}
	}
	return nil
}

func (b *syncBus) Subscribe(_ context.Context, topic string, h eventbus.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], h)
	return nil
}

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type allowGate struct{}

func (allowGate) RequireAnywhere(context.Context, permissions.Action) error { return nil }

type settings struct{ url string }

func (s settings) GetInstanceURL(context.Context) (string, error) { return s.url, nil }

// recordingExecutor finishes every job at once and records the builds and uninstalls it was asked for.
type recordingExecutor struct {
	mu         sync.Mutex
	builds     []string
	uninstalls []string
}

func (e *recordingExecutor) Build(_ context.Context, req runner.DeployRequestedEvent, send func(runner.Frame)) {
	e.mu.Lock()
	e.builds = append(e.builds, req.ID)
	e.mu.Unlock()
	send(runner.Frame{Type: runner.FrameDeployResult, ID: req.ID, Status: runner.DeployStatusHealthy})
}

func (e *recordingExecutor) Deploy(context.Context, runner.DeployRequestedEvent, func(runner.Frame)) {
}

func (e *recordingExecutor) Discover(context.Context) (runner.DiscoverReport, error) {
	return runner.DiscoverReport{}, nil
}

func (e *recordingExecutor) JoinNetworks(context.Context, string, []string, func(runner.Frame)) {}

func (e *recordingExecutor) Upgrade(context.Context, runner.Frame, func(runner.Frame) error) error {
	return nil
}

func (e *recordingExecutor) Uninstall(_ context.Context, name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.uninstalls = append(e.uninstalls, name)
	return nil
}

func (e *recordingExecutor) snapshot() (builds, uninstalls []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.builds...), append([]string(nil), e.uninstalls...)
}

type instance struct {
	store *storage.Store
	svc   *runner.Service
	bus   *syncBus
	srv   *httptest.Server
}

// newInstance serves the runner routes and the runner socket over a real, migrated SQLite database.
func newInstance(t *testing.T) *instance {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "nexul.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() }) // the process is done with it either way
	require.NoError(t, storage.Migrate(db))
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	bus := &syncBus{handlers: map[string][]eventbus.Handler{}}
	h := runner.NewHandler(runner.HandlerConfig{
		Bus: bus, Repo: store.Runners, Machines: store.Machines,
		HeartbeatInterval: 50 * time.Millisecond, WriteTimeout: time.Second, Logger: quiet,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = h.Run(ctx) }()
	svc := runner.NewService(store.Runners, h).WithMachines(store.Machines).
		WithInstall(runner.InstallConfig{Settings: settings{url: "https://nexul.example.com"}}).
		WithGate(allowGate{}).WithBus(bus)

	api := runner.NewHTTPHandler(svc)
	routes, public := api.Routes(), api.PublicRoutes()
	mux := http.NewServeMux()
	mux.Handle("/api/runners/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routes.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: "admin"})))
	}))
	mux.Handle("POST /api/runners/enroll", public)
	mux.Handle("/ws/runner", h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &instance{store: store, svc: svc, bus: bus, srv: srv}
}

func (in *instance) post(t *testing.T, path string, body any) (int, map[string]any) {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(in.srv.URL+path, "application/json", strings.NewReader(string(b)))
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return resp.StatusCode, out
}

// enroll mints a code for name on machine through the API and trades it, returning the enrolled record.
func (in *instance) enroll(t *testing.T, name, machine string) map[string]any {
	t.Helper()
	status, e := in.post(t, "/api/runners/enrollments", map[string]string{"name": name, "machine": machine})
	require.Equal(t, http.StatusCreated, status)
	status, enrolled := in.post(t, "/api/runners/enroll", map[string]string{"code": e["code"].(string), "name": name})
	require.Equal(t, http.StatusCreated, status)
	return enrolled
}

// start runs a real runner client with credential; the returned channel yields Run's result.
func (in *instance) start(t *testing.T, name, credential string, exec runner.Executor) <-chan error {
	t.Helper()
	client := runner.NewClient(runner.ClientConfig{
		URL: "ws" + strings.TrimPrefix(in.srv.URL, "http") + "/ws/runner", Credential: credential, Name: name,
		Executor: exec, Logger: quiet, HeartbeatInterval: 20 * time.Millisecond, ConnectTimeout: time.Second,
		BackoffBase: 5 * time.Millisecond, BackoffMax: 50 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- client.Run(ctx) }()
	return done
}

func (in *instance) connected(t *testing.T) map[string]bool {
	t.Helper()
	views, err := in.svc.ListRunners(context.Background())
	require.NoError(t, err)
	out := map[string]bool{}
	for _, v := range views {
		out[v.Name] = v.Connected
	}
	return out
}

func TestEnroll_CodeRefusals_RealSQLite(t *testing.T) {
	in := newInstance(t)
	status, e := in.post(t, "/api/runners/enrollments", map[string]string{"name": "build-box"})
	require.Equal(t, http.StatusCreated, status)
	code := e["code"].(string)

	status, body := in.post(t, "/api/runners/enroll", map[string]string{"code": code, "name": "other-box"})
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, "name_mismatch", body["code"])

	status, _ = in.post(t, "/api/runners/enroll", map[string]string{"code": code, "name": "build-box"})
	require.Equal(t, http.StatusCreated, status, "a name mismatch leaves the code usable")

	status, body = in.post(t, "/api/runners/enroll", map[string]string{"code": code, "name": "build-box"})
	assert.Equal(t, http.StatusUnauthorized, status, "a code enrolls once")
	assert.Equal(t, "invalid_code", body["code"])

	expired, hash, err := hostcred.MintCode()
	require.NoError(t, err)
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, in.store.Runners.CreateEnrollment(context.Background(), &runner.EnrollmentCode{
		CodeHash: hash, Name: "late-box", CreatedAt: past, ExpiresAt: past.Add(time.Hour),
	}))
	status, body = in.post(t, "/api/runners/enroll", map[string]string{"code": expired, "name": "late-box"})
	assert.Equal(t, http.StatusUnauthorized, status, "an expired code enrolls nothing")
	assert.Equal(t, "invalid_code", body["code"])
}

func TestRemoval_RealSQLite(t *testing.T) {
	in := newInstance(t)
	alpha := in.enroll(t, "alpha", "prod")
	beta := in.enroll(t, "beta", "prod")
	assert.Equal(t, "prod", alpha["machine"])

	alphaExec, betaExec := &recordingExecutor{}, &recordingExecutor{}
	alphaDone := in.start(t, "alpha", alpha["credential"].(string), alphaExec)
	in.start(t, "beta", beta["credential"].(string), betaExec)
	require.Eventually(t, func() bool {
		c := in.connected(t)
		return c["alpha"] && c["beta"]
	}, 3*time.Second, 10*time.Millisecond, "each credential connects as its own runner")

	req, err := http.NewRequest(http.MethodDelete, in.srv.URL+"/api/runners/"+alpha["id"].(string), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	select {
	case err := <-alphaDone:
		require.NoError(t, err, "the removed runner stops instead of reconnecting")
	case <-time.After(3 * time.Second):
		t.Fatal("the removed runner kept running")
	}
	_, uninstalls := alphaExec.snapshot()
	assert.Equal(t, []string{"alpha"}, uninstalls, "the uninstall frame runs nexul uninstall runner alpha")
	assert.Equal(t, map[string]bool{"beta": true}, in.connected(t))

	require.NoError(t, in.bus.Publish(context.Background(), runner.TopicDeployRequested, runner.DeployRequestedEvent{
		ID: "build-1", Kind: runner.RequestBuild, Repo: "org/app", Ref: "main", Target: "prod",
	}))
	require.Eventually(t, func() bool {
		builds, _ := betaExec.snapshot()
		return len(builds) == 1
	}, 3*time.Second, 10*time.Millisecond, "the other runner on the machine still takes jobs")
	builds, _ := alphaExec.snapshot()
	assert.Empty(t, builds)

	returning := &recordingExecutor{}
	select {
	case err := <-in.start(t, "alpha", alpha["credential"].(string), returning):
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("a removed runner coming back was not refused")
	}
	_, uninstalls = returning.snapshot()
	assert.Equal(t, []string{"alpha"}, uninstalls, "a runner that was offline when removed uninstalls itself on its return")
}

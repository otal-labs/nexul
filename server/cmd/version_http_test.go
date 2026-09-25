package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
)

// fakeUpgradeGate stands in for instanceAdminGate without a real auth.Service.
type fakeUpgradeGate struct {
	allow bool
	err   error
}

func (f fakeUpgradeGate) CanCreateWorkspace(context.Context, string) (bool, error) {
	return f.allow, f.err
}

// noopRunnerRepo satisfies runner.Repo without touching storage; UpgradeStatus/RequestUpgrade never call it.
type noopRunnerRepo struct{}

func (noopRunnerRepo) Create(context.Context, *runner.Runner) error { return nil }
func (noopRunnerRepo) GetByID(context.Context, string) (*runner.Runner, error) {
	return nil, apperrs.ErrNotFound
}
func (noopRunnerRepo) List(context.Context) ([]*runner.Runner, error)     { return nil, nil }
func (noopRunnerRepo) Heartbeat(context.Context, string, time.Time) error { return nil }
func (noopRunnerRepo) SetConnected(context.Context, string, bool) error   { return nil }
func (noopRunnerRepo) SetVersion(context.Context, string, string) error   { return nil }
func (noopRunnerRepo) SetMachine(context.Context, string, string) error   { return nil }
func (noopRunnerRepo) Delete(context.Context, string) error               { return nil }
func (noopRunnerRepo) Secret(context.Context) (string, error)             { return "", nil }
func (noopRunnerRepo) SetSecret(context.Context, string) error            { return nil }

// fakeUpgradeDispatch reports a fixed connected-runner view; only Runners() matters to UpgradeStatus.
type fakeUpgradeDispatch struct {
	runners []runner.RunnerStatus
}

func (f fakeUpgradeDispatch) Runners() []runner.RunnerStatus { return f.runners }
func (f fakeUpgradeDispatch) Queue() []runner.QueuedJob      { return nil }
func (f fakeUpgradeDispatch) Discover(context.Context, string, time.Duration) (runner.DiscoverReport, error) {
	return runner.DiscoverReport{}, nil
}

// memUpgradeRepo is an in-memory runner.UpgradeRepo for the HTTP handler tests.
type memUpgradeRepo struct {
	upgrades map[string]*runner.Upgrade
}

func newMemUpgradeRepo() *memUpgradeRepo {
	return &memUpgradeRepo{upgrades: map[string]*runner.Upgrade{}}
}

func (m *memUpgradeRepo) Create(_ context.Context, u *runner.Upgrade) error {
	cp := *u
	m.upgrades[u.ID] = &cp
	return nil
}

func (m *memUpgradeRepo) GetByID(_ context.Context, id string) (*runner.Upgrade, error) {
	u, ok := m.upgrades[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUpgradeRepo) Latest(_ context.Context) (*runner.Upgrade, error) {
	var latest *runner.Upgrade
	for _, u := range m.upgrades {
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

func (m *memUpgradeRepo) ListUnresolved(context.Context) ([]*runner.Upgrade, error) { return nil, nil }

func (m *memUpgradeRepo) SetStatus(_ context.Context, id, status, errMsg string) error {
	u, ok := m.upgrades[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	u.Status = status
	u.Error = errMsg
	return nil
}

// noopPublisher discards every event; the HTTP-layer tests assert on response bodies, not the live-push side effect.
type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, string, any) error { return nil }

// fakeReleaseServer stands in for GitHub's "latest release" lookup.
func fakeReleaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/otal-labs/nexul/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": tag, "html_url": "https://example.com/" + tag})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newTestUpgradeService(t *testing.T, tag string, connected, admin bool) *runner.Service {
	t.Helper()
	srv := fakeReleaseServer(t, tag)
	client := release.New(release.Config{APIBase: srv.URL})
	var runners []runner.RunnerStatus
	if connected {
		runners = []runner.RunnerStatus{{RunnerID: "instance"}}
	}
	return runner.NewService(noopRunnerRepo{}, fakeUpgradeDispatch{runners: runners}).
		WithInstall(runner.InstallConfig{Release: client}).
		WithUpgrades(newMemUpgradeRepo()).
		WithBus(noopPublisher{}).
		WithAdminGate(fakeUpgradeGate{allow: admin})
}

func withVersion(t *testing.T, v string) {
	t.Helper()
	orig := version.Version
	t.Cleanup(func() { version.Version = orig })
	version.Version = v
}

func TestInstanceUpgradeGetHandler(t *testing.T) {
	t.Run("no signed-in user is unauthorized", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		svc := newTestUpgradeService(t, "v0.2.1", true, true)
		req := httptest.NewRequest(http.MethodGet, "/api/instance/upgrade", nil)
		rec := httptest.NewRecorder()

		instanceUpgradeGetHandler(svc).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-admin is forbidden", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		svc := newTestUpgradeService(t, "v0.2.1", true, false)
		req := withActor(httptest.NewRequest(http.MethodGet, "/api/instance/upgrade", nil), "user-1")
		rec := httptest.NewRecorder()

		instanceUpgradeGetHandler(svc).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("admin gets the facts row", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		svc := newTestUpgradeService(t, "v0.2.1", true, true)
		req := withActor(httptest.NewRequest(http.MethodGet, "/api/instance/upgrade", nil), "user-1")
		rec := httptest.NewRecorder()

		instanceUpgradeGetHandler(svc).ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var status runner.UpgradeStatus
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &status))
		assert.Equal(t, "v0.2.0", status.Version)
		assert.True(t, status.CanUpgrade)
		assert.Equal(t, "v0.2.1", status.Latest.Version)
	})
}

func TestInstanceUpgradePostHandler(t *testing.T) {
	t.Run("202 with the new record", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		svc := newTestUpgradeService(t, "v0.2.1", true, true)
		req := withActor(httptest.NewRequest(http.MethodPost, "/api/instance/upgrade", nil), "user-1")
		rec := httptest.NewRecorder()

		instanceUpgradePostHandler(svc).ServeHTTP(rec, req)
		require.Equal(t, http.StatusAccepted, rec.Code)
		var got runner.Upgrade
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, "pending", got.Status)
		assert.Equal(t, "user-1", got.RequestedBy)
	})

	t.Run("409 with the reason when can_upgrade is false", func(t *testing.T) {
		withVersion(t, "dev")
		svc := newTestUpgradeService(t, "v0.2.1", true, true)
		req := withActor(httptest.NewRequest(http.MethodPost, "/api/instance/upgrade", nil), "user-1")
		rec := httptest.NewRecorder()

		instanceUpgradePostHandler(svc).ServeHTTP(rec, req)
		require.Equal(t, http.StatusConflict, rec.Code)
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "dev build", body["reason"])
	})

	t.Run("non-admin is forbidden", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		svc := newTestUpgradeService(t, "v0.2.1", true, false)
		req := withActor(httptest.NewRequest(http.MethodPost, "/api/instance/upgrade", nil), "user-1")
		rec := httptest.NewRecorder()

		instanceUpgradePostHandler(svc).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// withActor attaches an identity.Actor the way withIdentity middleware would after a real session authenticates,
// so these tests exercise requestUserID's fallback path without standing up a real auth.Service.
func withActor(r *http.Request, userID string) *http.Request {
	return r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: userID}))
}

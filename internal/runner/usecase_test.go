package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/release"
)

// fakeDispatch records the live view for use-case tests.
type fakeDispatch struct {
	runners    []RunnerStatus
	queue      []QueuedJob
	discoverFn func(ctx context.Context, machine string, timeout time.Duration) (DiscoverReport, error)
}

func (f *fakeDispatch) Runners() []RunnerStatus { return f.runners }
func (f *fakeDispatch) Queue() []QueuedJob      { return f.queue }
func (f *fakeDispatch) Discover(ctx context.Context, machine string, timeout time.Duration) (DiscoverReport, error) {
	if f.discoverFn == nil {
		return DiscoverReport{}, nil
	}
	return f.discoverFn(ctx, machine, timeout)
}

func TestService_ListRunners(t *testing.T) {
	t.Run("merges persisted presence with live running job", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		seen := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Runner{ID: "r-1", Name: "alpha", Version: "v0.1.6", LastSeen: seen, Connected: true, CreatedAt: seen}))
		require.NoError(t, repo.Create(context.Background(), &Runner{ID: "r-2", Name: "beta", LastSeen: seen, Connected: false, CreatedAt: seen}))

		svc := NewService(repo, &fakeDispatch{
			runners: []RunnerStatus{{RunnerID: "r-1", RunningJob: &RunningJob{ID: "d-9", Kind: RequestDeploy, Service: "api"}}},
		})
		got, err := svc.ListRunners(context.Background())
		require.NoError(t, err)
		require.Len(t, got, 2)

		byID := map[string]RunnerView{}
		for _, r := range got {
			byID[r.ID] = r
		}
		alpha := byID["r-1"]
		assert.True(t, alpha.Connected)
		assert.Equal(t, "v0.1.6", alpha.Version)
		require.NotNil(t, alpha.RunningJob)
		assert.Equal(t, "d-9", alpha.RunningJob.ID)
		assert.Equal(t, "api", alpha.RunningJob.Service)

		beta := byID["r-2"]
		assert.False(t, beta.Connected)
		assert.Empty(t, beta.Version)
		assert.Nil(t, beta.RunningJob)
	})

	t.Run("idle runner has no running job", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		require.NoError(t, repo.Create(context.Background(), &Runner{ID: "r-1"}))
		svc := NewService(repo, &fakeDispatch{runners: []RunnerStatus{{RunnerID: "r-1"}}})
		got, err := svc.ListRunners(context.Background())
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Nil(t, got[0].RunningJob)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.listErr = assert.AnError
		svc := NewService(repo, &fakeDispatch{})
		_, err := svc.ListRunners(context.Background())
		require.Error(t, err)
	})
}

func TestService_ListQueue(t *testing.T) {
	repo := newFakeRunnerRepo()
	svc := NewService(repo, &fakeDispatch{
		queue: []QueuedJob{{ID: "d-1", Kind: RequestDeploy, Service: "api"}},
	})
	got, err := svc.ListQueue(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "d-1", got[0].ID)
	assert.Equal(t, "api", got[0].Service)
}

// newUpgradeService wires a Service whose ReleaseClient hits apiBase, for UpgradeStatus/RequestUpgrade tests.
func newUpgradeService(apiBase string, upgrades UpgradeRepo, bus Publisher, dispatch *fakeDispatch) *Service {
	client := release.New(release.Config{APIBase: apiBase})
	return NewService(newFakeRunnerRepo(), dispatch).
		WithInstall(InstallConfig{Release: client}).
		WithUpgrades(upgrades).
		WithBus(bus).
		WithAdminGate(fakeAdminGate{admins: map[string]bool{"admin-1": true}})
}

// fakeAdminGate stands in for the auth-backed instance-admin fact.
type fakeAdminGate struct{ admins map[string]bool }

func (g fakeAdminGate) CanCreateWorkspace(_ context.Context, userID string) (bool, error) {
	return g.admins[userID], nil
}

// asAdmin is a context carrying the instance admin newUpgradeService recognizes.
func asAdmin() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "admin-1"})
}

func TestService_UpgradeStatus_ReasonPrecedence(t *testing.T) {
	t.Run("dev build", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0-beta-330", "x")
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "dev build", status.Reason)
		require.NotNil(t, status.Latest, "latest is still reported for display even on a dev build")
		assert.Equal(t, "v0.2.0-beta-330", status.Latest.Version)
	})

	t.Run("release lookup failed", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		srv.Close() // every lookup now fails to connect
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "release lookup failed", status.Reason)
		assert.Nil(t, status.Latest)
	})

	t.Run("already on the newest release", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.0", "x")
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "already on the newest release", status.Reason)
		assert.False(t, status.UpdateAvailable)
	})

	t.Run("an upgrade is already in progress takes precedence over runner connectivity", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		upgrades := newFakeUpgradeRepo()
		now := time.Now().UTC()
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
			ID: "u-1", FromVersion: "v0.2.0", ToVersion: "v0.2.1", Status: UpgradeStatusPending,
			CreatedAt: now, UpdatedAt: now,
		}))
		svc := newUpgradeService(srv.URL, upgrades, newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "an upgrade is already in progress", status.Reason)
	})

	t.Run("instance runner is not connected", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "instance runner is not connected", status.Reason)
	})

	t.Run("instance runner is busy", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: instanceRunnerID, RunningJob: &RunningJob{ID: "d-1"}}}}
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), dispatch)

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "instance runner is busy", status.Reason)
	})

	t.Run("can upgrade", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: instanceRunnerID}}}
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), dispatch)

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		assert.True(t, status.CanUpgrade)
		assert.Empty(t, status.Reason)
		assert.True(t, status.UpdateAvailable)
		assert.Equal(t, "v0.2.1", status.Latest.Version)
	})
}

func TestService_UpgradeStatus_LazyResolution(t *testing.T) {
	t.Run("a record matching the running version completes", func(t *testing.T) {
		withVersion(t, "v0.2.1")
		srv := fakeGitHub(t, "v0.2.1", "x")
		upgrades := newFakeUpgradeRepo()
		bus := newFakeBus()
		now := time.Now().UTC()
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
			ID: "u-1", FromVersion: "v0.2.0", ToVersion: "v0.2.1", Status: UpgradeStatusStarted,
			CreatedAt: now, UpdatedAt: now,
		}))
		svc := newUpgradeService(srv.URL, upgrades, bus, &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		require.NotNil(t, status.Upgrade)
		assert.Equal(t, UpgradeStatusCompleted, status.Upgrade.Status)

		stored, err := upgrades.GetByID(context.Background(), "u-1")
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusCompleted, stored.Status)
		assert.Len(t, bus.topicEvents(TopicInstanceUpgradeChanged), 1)
	})

	t.Run("a record stuck past the window fails with the journal hint", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		upgrades := newFakeUpgradeRepo()
		old := time.Now().UTC().Add(-20 * time.Minute)
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
			ID: "u-1", FromVersion: "v0.2.0", ToVersion: "v0.2.1", Status: UpgradeStatusStarted,
			CreatedAt: old, UpdatedAt: old,
		}))
		svc := newUpgradeService(srv.URL, upgrades, newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		require.NotNil(t, status.Upgrade)
		assert.Equal(t, UpgradeStatusFailed, status.Upgrade.Status)
		assert.Contains(t, status.Upgrade.Error, "instance is still on v0.2.0")
		assert.Contains(t, status.Upgrade.Error, "journalctl -u nexul-upgrade")
	})

	t.Run("a record still within the window stays unresolved", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		upgrades := newFakeUpgradeRepo()
		recent := time.Now().UTC().Add(-time.Minute)
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
			ID: "u-1", FromVersion: "v0.2.0", ToVersion: "v0.2.1", Status: UpgradeStatusStarted,
			CreatedAt: recent, UpdatedAt: recent,
		}))
		svc := newUpgradeService(srv.URL, upgrades, newFakeBus(), &fakeDispatch{})

		status, err := svc.UpgradeStatus(asAdmin())
		require.NoError(t, err)
		require.NotNil(t, status.Upgrade)
		assert.Equal(t, UpgradeStatusStarted, status.Upgrade.Status)
	})
}

func TestService_Upgrade_RequiresInstanceAdmin(t *testing.T) {
	withVersion(t, "v0.2.0")
	srv := fakeGitHub(t, "v0.2.1", "x")
	dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: instanceRunnerID}}}
	svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), dispatch)
	member := identity.WithActor(context.Background(), identity.Actor{ID: "member-1"})

	_, err := svc.UpgradeStatus(member)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = svc.RequestUpgrade(member, "member-1")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = svc.RequestUpgrade(context.Background(), "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestService_RequestUpgrade(t *testing.T) {
	t.Run("blocked can_upgrade wraps ErrConflict with the reason", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0", "x")
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})

		_, err := svc.RequestUpgrade(asAdmin(), "user-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		var blocked *UpgradeBlockedError
		require.True(t, errors.As(err, &blocked))
		assert.Equal(t, "dev build", blocked.Reason)
	})

	t.Run("writes the record pending and dispatches to the handler", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		upgrades := newFakeUpgradeRepo()
		bus := newFakeBus()
		dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: instanceRunnerID}}}
		svc := newUpgradeService(srv.URL, upgrades, bus, dispatch)

		got, err := svc.RequestUpgrade(asAdmin(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusPending, got.Status)
		assert.Equal(t, "v0.2.0", got.FromVersion)
		assert.Equal(t, "v0.2.1", got.ToVersion)
		assert.Equal(t, "user-1", got.RequestedBy)
		assert.NotEmpty(t, got.ID)

		stored, err := upgrades.GetByID(context.Background(), got.ID)
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusPending, stored.Status)

		require.Len(t, bus.topicEvents(TopicInstanceUpgradeRequested), 1)
		req := decodeEvent[InstanceUpgradeRequestedEvent](t, bus.topicEvents(TopicInstanceUpgradeRequested)[0])
		assert.Equal(t, got.ID, req.ID)
		assert.Equal(t, "v0.2.1", req.Version)
		assert.Len(t, bus.topicEvents(TopicInstanceUpgradeChanged), 1)
	})
}

func TestService_ResolvePendingUpgrade(t *testing.T) {
	t.Run("resolves every unresolved record", func(t *testing.T) {
		upgrades := newFakeUpgradeRepo()
		withVersion(t, "v0.2.1")
		now := time.Now().UTC()
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
			ID: "u-match", ToVersion: "v0.2.1", Status: UpgradeStatusStarted, CreatedAt: now, UpdatedAt: now,
		}))
		old := now.Add(-20 * time.Minute)
		require.NoError(t, upgrades.Create(context.Background(), &Upgrade{
			ID: "u-stuck", ToVersion: "v0.2.2", Status: UpgradeStatusPending, CreatedAt: old, UpdatedAt: old,
		}))
		svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithUpgrades(upgrades).WithBus(newFakeBus())

		require.NoError(t, svc.ResolvePendingUpgrade(context.Background()))

		matched, err := upgrades.GetByID(context.Background(), "u-match")
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusCompleted, matched.Status)

		stuck, err := upgrades.GetByID(context.Background(), "u-stuck")
		require.NoError(t, err)
		assert.Equal(t, UpgradeStatusFailed, stuck.Status)
	})

	t.Run("no upgrade repo configured is a no-op", func(t *testing.T) {
		svc := NewService(newFakeRunnerRepo(), &fakeDispatch{})
		require.NoError(t, svc.ResolvePendingUpgrade(context.Background()))
	})
}

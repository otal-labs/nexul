package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestDeploy(id string) *deploy.Deploy {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &deploy.Deploy{
		ID: id, Service: "api", Target: "10.0.0.1:22", Image: "ghcr.io/onik/api:v1",
		Status: deploy.StatusPending, Strategy: deploy.StrategyCompose, Log: "",
		CreatedAt: now, UpdatedAt: now,
	}
}

func TestDeploysRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Deploys.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	err := s.Deploys.Create(context.Background(), newTestDeploy("dep-1"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDeploysRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestDeploy("dep-1")
	require.NoError(t, s.Deploys.Create(context.Background(), want))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, "api", got.Service)
	assert.Equal(t, "10.0.0.1:22", got.Target)
	assert.Equal(t, deploy.StatusPending, got.Status)
	assert.Equal(t, deploy.StrategyCompose, got.Strategy)
}

func TestDeploysRepo_BuildKind_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	build := newTestDeploy("dep-b")
	build.Kind = deploy.KindBuild
	build.Image = ""
	require.NoError(t, s.Deploys.Create(context.Background(), build))

	got, err := s.Deploys.GetByID(context.Background(), "dep-b")
	require.NoError(t, err)
	assert.Equal(t, deploy.KindBuild, got.Kind, "a repo-driven build round-trips as kind build")
	assert.Equal(t, "", got.Image)

	// Legacy rows without a kind (migration default) read back as deploys.
	legacy := newTestDeploy("dep-legacy")
	require.NoError(t, s.Deploys.Create(context.Background(), legacy))
	gotLegacy, err := s.Deploys.GetByID(context.Background(), "dep-legacy")
	require.NoError(t, err)
	assert.Equal(t, deploy.KindDeploy, gotLegacy.Kind)
}

func TestDeploysRepo_List_ReturnsAll(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-2")))

	got, err := s.Deploys.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestDeploysRepo_ListByService_ScopesToService(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-2")))
	other := newTestDeploy("dep-3")
	other.Service = "worker"
	require.NoError(t, s.Deploys.Create(context.Background(), other))

	got, err := s.Deploys.ListByService(context.Background(), "api")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestDeploysRepo_ListByStatus_ScopesToStatus(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	running := newTestDeploy("dep-2")
	running.Status = deploy.StatusRunning
	require.NoError(t, s.Deploys.Create(context.Background(), running))

	got, err := s.Deploys.ListByStatus(context.Background(), deploy.StatusPending)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "dep-1", got[0].ID)
}

func TestDeploysRepo_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Deploys.UpdateStatus(context.Background(), "missing", deploy.StatusHealthy)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_UpdateStatus_Persists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.UpdateStatus(context.Background(), "dep-1", deploy.StatusHealthy))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusHealthy, got.Status)
}

func TestDeploysRepo_SetAddress_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Deploys.SetAddress(context.Background(), "missing", "172.18.0.4")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_SetAddress_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.SetAddress(context.Background(), "dep-1", "172.18.0.4"))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, "172.18.0.4", got.Address)
}

func TestDeploysRepo_AppendLog_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Deploys.AppendLog(context.Background(), "missing", "line")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeploysRepo_AppendLog_Appends(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Deploys.Create(context.Background(), newTestDeploy("dep-1")))
	require.NoError(t, s.Deploys.AppendLog(context.Background(), "dep-1", "pulling image\n"))
	require.NoError(t, s.Deploys.AppendLog(context.Background(), "dep-1", "container healthy\n"))

	got, err := s.Deploys.GetByID(context.Background(), "dep-1")
	require.NoError(t, err)
	assert.Equal(t, "pulling image\ncontainer healthy\n", got.Log)
}

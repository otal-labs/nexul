package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/runner"
)

func newTestUpgrade(id, status string) *runner.Upgrade {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	return &runner.Upgrade{
		ID: id, FromVersion: "v0.2.0", ToVersion: "v0.2.1", Status: status,
		RequestedBy: "user-1", RunnerID: "instance", CreatedAt: now, UpdatedAt: now,
	}
}

func TestInstanceUpgradesRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.InstanceUpgrades.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestInstanceUpgradesRepo_Latest_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.InstanceUpgrades.Latest(context.Background())
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestInstanceUpgradesRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestUpgrade("u-1", "pending")
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), want))

	got, err := s.InstanceUpgrades.GetByID(context.Background(), "u-1")
	require.NoError(t, err)
	assert.Equal(t, want.FromVersion, got.FromVersion)
	assert.Equal(t, want.ToVersion, got.ToVersion)
	assert.Equal(t, want.Status, got.Status)
	assert.Equal(t, want.RequestedBy, got.RequestedBy)
	assert.Equal(t, want.RunnerID, got.RunnerID)
	assert.Equal(t, "", got.Error)
	assert.True(t, want.CreatedAt.Equal(got.CreatedAt))
}

func TestInstanceUpgradesRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-1", "pending")))
	err := s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-1", "pending"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestInstanceUpgradesRepo_Latest_ReturnsMostRecentlyCreated(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	older := newTestUpgrade("u-1", "completed")
	older.CreatedAt = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), older))
	newer := newTestUpgrade("u-2", "pending")
	newer.CreatedAt = time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newer))

	got, err := s.InstanceUpgrades.Latest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "u-2", got.ID)
}

func TestInstanceUpgradesRepo_ListUnresolved_OnlyPendingAndStarted(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-pending", "pending")))
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-started", "started")))
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-completed", "completed")))
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-failed", "failed")))

	got, err := s.InstanceUpgrades.ListUnresolved(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	var ids []string
	for _, u := range got {
		ids = append(ids, u.ID)
	}
	assert.ElementsMatch(t, []string{"u-pending", "u-started"}, ids)
}

func TestInstanceUpgradesRepo_SetStatus_UpdatesStatusAndError(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.InstanceUpgrades.Create(context.Background(), newTestUpgrade("u-1", "started")))

	require.NoError(t, s.InstanceUpgrades.SetStatus(context.Background(), "u-1", "failed", "boom"))

	got, err := s.InstanceUpgrades.GetByID(context.Background(), "u-1")
	require.NoError(t, err)
	assert.Equal(t, "failed", got.Status)
	assert.Equal(t, "boom", got.Error)
	assert.True(t, got.UpdatedAt.After(got.CreatedAt) || got.UpdatedAt.Equal(got.CreatedAt))
}

func TestInstanceUpgradesRepo_SetStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.InstanceUpgrades.SetStatus(context.Background(), "missing", "failed", "boom")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

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

func newTestRunner(id string) *runner.Runner {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &runner.Runner{ID: id, Name: "build-box", LastSeen: now, Connected: false, CreatedAt: now}
}

func TestRunnersRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Runners.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	err := s.Runners.Create(context.Background(), newTestRunner("r-1"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestRunnersRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestRunner("r-1")
	want.Version = "v0.1.5"
	require.NoError(t, s.Runners.Create(context.Background(), want))

	got, err := s.Runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, "build-box", got.Name)
	assert.False(t, got.Connected)
	assert.Equal(t, "v0.1.5", got.Version)
}

func TestRunnersRepo_SetVersion_UpdatesVersion(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))

	require.NoError(t, s.Runners.SetVersion(context.Background(), "r-1", "v0.1.6"))

	got, err := s.Runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, "v0.1.6", got.Version)
}

func TestRunnersRepo_SetVersion_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Runners.SetVersion(context.Background(), "missing", "v0.1.6")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_List_ReturnsAll(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-2")))

	got, err := s.Runners.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestRunnersRepo_Heartbeat_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Runners.Heartbeat(context.Background(), "missing", time.Now())
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_Heartbeat_UpdatesSeenAndConnected(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))

	seen := time.Now().Add(-time.Minute).Truncate(time.Second)
	require.NoError(t, s.Runners.Heartbeat(context.Background(), "r-1", seen))

	got, err := s.Runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.True(t, got.Connected)
	assert.Equal(t, seen.UTC(), got.LastSeen)
}

func TestRunnersRepo_SetConnected_MarksDisconnected(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	require.NoError(t, s.Runners.Heartbeat(context.Background(), "r-1", time.Now()))
	require.NoError(t, s.Runners.SetConnected(context.Background(), "r-1", false))

	got, err := s.Runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.False(t, got.Connected)
}

func TestRunnersRepo_SetConnected_CancelledContext_LeavesRowUntouched(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	require.NoError(t, s.Runners.Heartbeat(context.Background(), "r-1", time.Now()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, s.Runners.SetConnected(ctx, "r-1", false), context.Canceled)

	got, err := s.Runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.True(t, got.Connected, "a cancelled ctx must not reach the database; this is why the handler detaches it")
}

func TestRunnersRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Runners.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_Delete_RemovesRunner(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	require.NoError(t, s.Runners.Delete(context.Background(), "r-1"))
	_, err := s.Runners.GetByID(context.Background(), "r-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_Secret_GeneratedOnceAndStable(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	first, err := s.Runners.Secret(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, first)

	second, err := s.Runners.Secret(context.Background())
	require.NoError(t, err)
	assert.Equal(t, first, second, "secret must be generated once and stay stable across calls")
}

func TestRunnersRepo_SetSecret_ThenSecret_ReturnsSeededValue(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.SetSecret(context.Background(), "seeded-secret"))

	got, err := s.Runners.Secret(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "seeded-secret", got)
}

func TestRunnersRepo_SetSecret_Empty_ReturnsValidationError(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Runners.SetSecret(context.Background(), "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestRunnersRepo_SetMachine_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))

	require.NoError(t, s.Runners.SetMachine(context.Background(), "r-1", "m-1"))

	got, err := s.Runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, "m-1", got.MachineID)
}

func TestRunnersRepo_SetMachine_UnknownRunner_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Runners.SetMachine(context.Background(), "missing", "m-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

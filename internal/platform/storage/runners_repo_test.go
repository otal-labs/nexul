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
	return &runner.Runner{ID: id, Name: "box-" + id, LastSeen: now, Connected: false, CreatedAt: now}
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
	assert.Equal(t, "box-r-1", got.Name)
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

func TestRunnersRepo_Create_DuplicateName_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	other := newTestRunner("r-2")
	other.Name = "box-r-1"
	require.ErrorIs(t, s.Runners.Create(context.Background(), other), apperrs.ErrConflict)
}

func TestRunnersRepo_GetByName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Runners.GetByName(context.Background(), "box-r-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	got, err := s.Runners.GetByName(context.Background(), "box-r-1")
	require.NoError(t, err)
	assert.Equal(t, "r-1", got.ID)
}

var enrollNow = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func mustCreateEnrollment(t *testing.T, s *Store, hash, name string, expiresAt time.Time) {
	t.Helper()
	require.NoError(t, s.Runners.CreateEnrollment(context.Background(), &runner.EnrollmentCode{
		CodeHash: hash, Name: name, Machine: "m", CreatedAt: enrollNow, ExpiresAt: expiresAt,
	}))
}

func TestRunnersRepo_GetEnrollment_UnknownOrExpired_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateEnrollment(t, s, "h-live", "a", enrollNow.Add(time.Hour))
	mustCreateEnrollment(t, s, "h-old", "b", enrollNow.Add(time.Minute))

	_, err := s.Runners.GetEnrollment(context.Background(), "h-missing", enrollNow)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.Runners.GetEnrollment(context.Background(), "h-old", enrollNow.Add(time.Minute))
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	got, err := s.Runners.GetEnrollment(context.Background(), "h-live", enrollNow)
	require.NoError(t, err)
	assert.Equal(t, runner.EnrollmentCode{CodeHash: "h-live", Name: "a", Machine: "m", CreatedAt: enrollNow, ExpiresAt: enrollNow.Add(time.Hour)}, *got)
}

func TestRunnersRepo_CreateEnrollment_PrunesExpiredCodes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateEnrollment(t, s, "h-expired", "a", enrollNow.Add(-time.Second))
	mustCreateEnrollment(t, s, "h-new", "b", enrollNow.Add(time.Hour))

	var n int
	require.NoError(t, s.db.QueryRow(`SELECT COUNT(*) FROM runner_enrollment_codes`).Scan(&n))
	assert.Equal(t, 1, n)
}

func TestRunnersRepo_Enroll_ConsumesCodeOnce(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateEnrollment(t, s, "h-1", "box-r-1", enrollNow.Add(time.Hour))

	require.NoError(t, s.Runners.Enroll(context.Background(), "h-1", newTestRunner("r-1"), "cred-hash", enrollNow))

	got, err := s.Runners.GetByName(context.Background(), "box-r-1")
	require.NoError(t, err)
	assert.Equal(t, "r-1", got.ID)
	cred, err := s.Runners.GetCredential(context.Background(), "cred-hash")
	require.NoError(t, err)
	assert.Equal(t, runner.Credential{RunnerID: "r-1", RunnerName: "box-r-1", CreatedAt: enrollNow}, *cred)

	err = s.Runners.Enroll(context.Background(), "h-1", newTestRunner("r-2"), "cred-hash-2", enrollNow)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a used code is gone")
}

func TestRunnersRepo_Enroll_ExpiredCode_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateEnrollment(t, s, "h-1", "box-r-1", enrollNow.Add(time.Minute))
	err := s.Runners.Enroll(context.Background(), "h-1", newTestRunner("r-1"), "cred-hash", enrollNow.Add(time.Hour))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_Enroll_NameTaken_ConflictAndCodeKept(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Runners.Create(context.Background(), newTestRunner("r-1")))
	mustCreateEnrollment(t, s, "h-1", "box-r-1", enrollNow.Add(time.Hour))

	again := newTestRunner("r-1b")
	again.Name = "box-r-1"
	err := s.Runners.Enroll(context.Background(), "h-1", again, "cred-hash", enrollNow)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	_, err = s.Runners.GetEnrollment(context.Background(), "h-1", enrollNow)
	require.NoError(t, err, "the failed enrollment rolls back, leaving the code usable")
	_, err = s.Runners.GetCredential(context.Background(), "cred-hash")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestRunnersRepo_Remove_RevokesCredentialAndDeletesRunner(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateEnrollment(t, s, "h-1", "box-r-1", enrollNow.Add(time.Hour))
	require.NoError(t, s.Runners.Enroll(context.Background(), "h-1", newTestRunner("r-1"), "cred-hash", enrollNow))

	require.NoError(t, s.Runners.Remove(context.Background(), "r-1", enrollNow))

	_, err := s.Runners.GetByID(context.Background(), "r-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	cred, err := s.Runners.GetCredential(context.Background(), "cred-hash")
	require.NoError(t, err, "the credential stays as a tombstone")
	assert.True(t, cred.Revoked)

	require.ErrorIs(t, s.Runners.Remove(context.Background(), "r-1", enrollNow), apperrs.ErrNotFound)
}

func TestRunnersRepo_GetCredential_Unknown_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Runners.GetCredential(context.Background(), "nope")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

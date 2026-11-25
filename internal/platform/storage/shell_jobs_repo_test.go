package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func shellEvent(id string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: id, Topic: pairing.TopicShellJobChanged, Payload: pairing.ShellJobChangedEvent{JobID: id, MembersOnly: true}}
}

func finishedShellJob(t *testing.T, s *Store, id string, ended time.Time) {
	t.Helper()
	ctx := t.Context()
	job := pairing.ShellJob{ID: id, ComputerID: "c1", OwnerID: "u1", RequesterID: "u1", Adapter: pairing.AdapterMCP, Command: "cat ~/.env",
		TimeoutSeconds: 60, Status: pairing.ShellQueued, CreatedAt: ended.Add(-time.Minute)}
	require.NoError(t, s.Pairing.InsertShellJob(ctx, job, shellEvent(id+"-q")))
	started, err := s.Pairing.StartShellJob(ctx, id, ended.Add(-time.Minute), shellEvent(id+"-r"))
	require.NoError(t, err)
	require.True(t, started)
	code := 0
	job.Status, job.ExitCode, job.Output, job.EndedAt = pairing.ShellSucceeded, &code, "TOKEN=abc\n", &ended
	require.NoError(t, s.Pairing.FinishShellJob(ctx, job, shellEvent(id+"-e")))
}

func TestShellJobs_OutputPurgedAfter30DaysKeepsTheRecord(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now()
	finishedShellJob(t, s, "j-month", now.Add(-31*24*time.Hour))
	finishedShellJob(t, s, "j-year", now.Add(-365*24*time.Hour))
	finishedShellJob(t, s, "j-fresh", now.Add(-24*time.Hour))

	n, err := s.Pairing.PurgeShellOutput(ctx, now.Add(-pairing.ShellOutputRetention), 500)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)
	_, err = s.Audit.DeleteBefore(ctx, now.Add(-45*24*time.Hour), 500)
	require.NoError(t, err)

	for _, id := range []string{"j-month", "j-year"} {
		job, err := s.Pairing.GetShellJob(ctx, id)
		require.NoError(t, err, id)
		assert.Empty(t, job.Output, id)
		assert.True(t, job.OutputPurged, id)
		assert.Equal(t, "cat ~/.env", job.Command, id)
		assert.Equal(t, "u1", job.RequesterID, id)
		require.NotNil(t, job.ExitCode, id)
		assert.Equal(t, 0, *job.ExitCode, id)
		assert.NotNil(t, job.EndedAt, id)
	}
	fresh, err := s.Pairing.GetShellJob(ctx, "j-fresh")
	require.NoError(t, err)
	assert.Equal(t, "TOKEN=abc\n", fresh.Output)
}

func TestShellJobs_ListPagesNewestFirstWithoutOutputAndNarrowsToARequester(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	base := time.Now().Add(-time.Hour)
	for i, who := range []string{"u1", "u2", "u1"} {
		job := pairing.ShellJob{ID: string(rune('a' + i)), ComputerID: "c1", OwnerID: "u1", RequesterID: who, Adapter: pairing.AdapterWeb,
			Command: "ls", TimeoutSeconds: 60, Status: pairing.ShellQueued, CreatedAt: base.Add(time.Duration(i) * time.Minute)}
		require.NoError(t, s.Pairing.InsertShellJob(ctx, job, shellEvent(job.ID)))
	}

	all, total, err := s.Pairing.ListShellJobs(ctx, "c1", "", paging.Window{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, []string{"c", "b"}, []string{all[0].ID, all[1].ID})
	mine, total, err := s.Pairing.ListShellJobs(ctx, "c1", "u2", paging.Window{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "b", mine[0].ID)
}

func TestShellJobs_ARestartFailsWhatWasLeftAndAnEndedJobStaysEnded(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	finishedShellJob(t, s, "j-done", time.Now())
	left := pairing.ShellJob{ID: "j-left", ComputerID: "c1", OwnerID: "u1", RequesterID: "u1", Adapter: pairing.AdapterWeb, Command: "make",
		TimeoutSeconds: 60, Status: pairing.ShellQueued, CreatedAt: time.Now()}
	require.NoError(t, s.Pairing.InsertShellJob(ctx, left, shellEvent("j-left")))

	n, err := s.Pairing.FailUnfinishedShellJobs(ctx, time.Now())
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	got, err := s.Pairing.GetShellJob(ctx, "j-left")
	require.NoError(t, err)
	assert.Equal(t, pairing.ShellFailed, got.Status)

	left.Status = pairing.ShellCancelled
	require.NoError(t, s.Pairing.FinishShellJob(ctx, left, shellEvent("j-left-2")))
	got, err = s.Pairing.GetShellJob(ctx, "j-left")
	require.NoError(t, err)
	assert.Equal(t, pairing.ShellFailed, got.Status, "a job that ended is never ended again")
}

func TestPairingRepo_SetComputerShell_OwnerOnlyAndOnByDefault(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.True(t, got.ShellEnabled)
	require.NoError(t, s.Pairing.SetComputerShell(ctx, "u1", "c1", false, time.Now()))
	got, err = s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.False(t, got.ShellEnabled)
	require.ErrorIs(t, s.Pairing.SetComputerShell(ctx, "u2", "c1", true, time.Now()), apperrs.ErrNotFound)
}

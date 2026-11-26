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
	"github.com/otal-labs/nexul/internal/platform/shelljob"
	"github.com/otal-labs/nexul/internal/runner"
)

func shellEvent(id string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: id, Topic: pairing.TopicShellJobChanged, Payload: pairing.ShellJobChangedEvent{JobID: id, MembersOnly: true}}
}

func finishedShellJob(t *testing.T, s *Store, id string, ended time.Time) {
	t.Helper()
	ctx := t.Context()
	job := shelljob.Job{ID: id, ComputerID: "c1", OwnerID: "u1", RequesterID: "u1", Adapter: shelljob.AdapterMCP, Command: "cat ~/.env",
		TimeoutSeconds: 60, Status: shelljob.Queued, CreatedAt: ended.Add(-time.Minute)}
	require.NoError(t, s.Pairing.InsertShellJob(ctx, job, shellEvent(id+"-q")))
	started, err := s.Pairing.StartShellJob(ctx, id, ended.Add(-time.Minute), shellEvent(id+"-r"))
	require.NoError(t, err)
	require.True(t, started)
	code := 0
	job.Status, job.ExitCode, job.Output, job.EndedAt = shelljob.Succeeded, &code, "TOKEN=abc\n", &ended
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

	n, err := s.Pairing.PurgeShellOutput(ctx, now.Add(-shelljob.OutputRetention), 500)
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
		job := shelljob.Job{ID: string(rune('a' + i)), ComputerID: "c1", OwnerID: "u1", RequesterID: who, Adapter: shelljob.AdapterWeb,
			Command: "ls", TimeoutSeconds: 60, Status: shelljob.Queued, CreatedAt: base.Add(time.Duration(i) * time.Minute)}
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
	left := shelljob.Job{ID: "j-left", ComputerID: "c1", OwnerID: "u1", RequesterID: "u1", Adapter: shelljob.AdapterWeb, Command: "make",
		TimeoutSeconds: 60, Status: shelljob.Queued, CreatedAt: time.Now()}
	require.NoError(t, s.Pairing.InsertShellJob(ctx, left, shellEvent("j-left")))

	n, err := s.Pairing.FailUnfinishedShellJobs(ctx, time.Now())
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	got, err := s.Pairing.GetShellJob(ctx, "j-left")
	require.NoError(t, err)
	assert.Equal(t, shelljob.Failed, got.Status)

	left.Status = shelljob.Cancelled
	require.NoError(t, s.Pairing.FinishShellJob(ctx, left, shellEvent("j-left-2")))
	got, err = s.Pairing.GetShellJob(ctx, "j-left")
	require.NoError(t, err)
	assert.Equal(t, shelljob.Failed, got.Status, "a job that ended is never ended again")
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

func TestShellJobs_AMachinesListNamesTheRunnerAndLeavesComputersOut(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, s.Machines.Create(ctx, &runner.Machine{ID: "m1", Name: "prod", FirstSeen: now, LastSeen: now}))
	require.NoError(t, s.Runners.Create(ctx, &runner.Runner{ID: "r1", Name: "edge", MachineID: "m1", CreatedAt: now, LastSeen: now}))
	for i, job := range []shelljob.Job{
		{ID: "on-machine", RunnerID: "r1", MachineID: "m1"},
		{ID: "on-computer", ComputerID: "c1", OwnerID: "u1"},
	} {
		job.RequesterID, job.Adapter, job.Command, job.TimeoutSeconds, job.Status = "u1", shelljob.AdapterWeb, "uptime", 60, shelljob.Queued
		job.CreatedAt = now.Add(time.Duration(i) * time.Second)
		require.NoError(t, s.ShellJobs.InsertShellJob(ctx, job, shellEvent(job.ID)))
	}

	jobs, total, err := s.ShellJobs.ListMachineShellJobs(ctx, "m1", paging.Window{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, jobs, 1)
	assert.Equal(t, "on-machine", jobs[0].ID)
	assert.Equal(t, "edge", jobs[0].RunnerName)
	assert.Equal(t, "r1", jobs[0].RunnerID)
	got, err := s.ShellJobs.GetShellJob(ctx, "on-machine")
	require.NoError(t, err)
	assert.Equal(t, "m1", got.MachineID)
}

func TestMachines_SetShell(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, s.Machines.Create(ctx, &runner.Machine{ID: "m1", Name: "prod", FirstSeen: now, LastSeen: now}))
	m, err := s.Machines.Get(ctx, "m1")
	require.NoError(t, err)
	assert.False(t, m.ShellEnabled, "a machine starts with commands off")

	require.NoError(t, s.Machines.SetShell(ctx, "m1", true))
	m, err = s.Machines.Get(ctx, "m1")
	require.NoError(t, err)
	assert.True(t, m.ShellEnabled)
	require.ErrorIs(t, s.Machines.SetShell(ctx, "m-gone", true), apperrs.ErrNotFound)
}

func TestPairingRepo_SetComputerPlaysOff_OwnerOnlyAndTakesPlaysByDefault(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("c1", "u1", "home")))

	got, err := s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.False(t, got.PlaysOff)
	require.NoError(t, s.Pairing.SetComputerPlaysOff(ctx, "u1", "c1", true, time.Now()))
	got, err = s.Pairing.GetComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.True(t, got.PlaysOff)
	require.ErrorIs(t, s.Pairing.SetComputerPlaysOff(ctx, "u2", "c1", false, time.Now()), apperrs.ErrNotFound)
}

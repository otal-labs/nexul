package runner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// shellGate holds, per person, what they hold on machine m-1; a person it does not name can't see it.
type shellGate struct {
	mu   sync.Mutex
	held map[string][]permissions.Action
}

func (g *shellGate) actions(userID, machineID string) ([]permissions.Action, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	held, ok := g.held[userID]
	return held, ok && machineID == "m-1"
}

func (g *shellGate) set(userID string, held ...permissions.Action) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.held[userID] = held
}

func (g *shellGate) RequireAnywhere(context.Context, permissions.Action) error { return nil }

func (g *shellGate) RequireMachine(ctx context.Context, machineID string, action permissions.Action) error {
	actor, _ := identity.ActorFromCtx(ctx)
	held, seen := g.actions(actor.ID, machineID)
	if !seen {
		return fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, machineID)
	}
	if !slices.Contains(held, action) {
		return fmt.Errorf("%w: %s", apperrs.ErrForbidden, action)
	}
	return nil
}

func (g *shellGate) CallerMachines(context.Context) ([]string, bool, error) {
	return []string{"m-1"}, false, nil
}

func (g *shellGate) HoldsAnywhere(context.Context, string, permissions.Action) (bool, error) {
	return false, nil
}

func (g *shellGate) MachinePermissions(_ context.Context, userID, machineID string) (permissions.Set, error) {
	held, seen := g.actions(userID, machineID)
	if !seen {
		return nil, apperrs.ErrNotFound
	}
	return permissions.SetOf(held...), nil
}

// fakeShellJobs keeps machines' jobs in memory.
type fakeShellJobs struct {
	mu   sync.Mutex
	jobs map[string]shelljob.Job
	evts []eventbus.OutboxEvent
}

func (f *fakeShellJobs) InsertShellJob(_ context.Context, job shelljob.Job, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID], f.evts = job, append(f.evts, evt)
	return nil
}

func (f *fakeShellJobs) StartShellJob(_ context.Context, id string, at time.Time, evt eventbus.OutboxEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[id]
	if !ok || job.Status != shelljob.Queued {
		return false, nil
	}
	job.Status, job.StartedAt = shelljob.Running, &at
	f.jobs[id], f.evts = job, append(f.evts, evt)
	return true, nil
}

func (f *fakeShellJobs) FinishShellJob(_ context.Context, job shelljob.Job, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID], f.evts = job, append(f.evts, evt)
	return nil
}

func (f *fakeShellJobs) GetShellJob(_ context.Context, id string) (*shelljob.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &job, nil
}

func (f *fakeShellJobs) ListMachineShellJobs(_ context.Context, machineID string, _ paging.Window) ([]shelljob.Job, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []shelljob.Job
	for _, j := range f.jobs {
		if j.MachineID == machineID {
			out = append(out, j)
		}
	}
	return out, len(out), nil
}

func (f *fakeShellJobs) job(id string) shelljob.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jobs[id]
}

// fakeMachineShells stands in for the runners' connections: it keeps each run's sink until the test ends it.
type fakeMachineShells struct {
	mu      sync.Mutex
	sinks   map[string]ShellSink
	offline bool
	started chan string
}

func (f *fakeMachineShells) RunMachineShell(_ context.Context, runnerID string, run ShellRun, sink ShellSink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.offline {
		return apperrs.Retryable(fmt.Errorf("runner %s is not connected", runnerID))
	}
	f.sinks[run.ID] = sink
	f.started <- run.ID
	return nil
}

func (f *fakeMachineShells) OpenMachineTerminal(context.Context, string, string, shelljob.TerminalSize) (shelljob.TerminalStream, error) {
	return nil, errors.New("the runner is not connected")
}

func (f *fakeMachineShells) CancelShell(_ context.Context, jobID string) error {
	f.mu.Lock()
	sink := f.sinks[jobID]
	f.mu.Unlock()
	if sink != nil {
		sink.Done(ShellResult{Status: ShellStatusCancelled})
	}
	return nil
}

func (f *fakeMachineShells) sink(id string) ShellSink {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sinks[id]
}

type machineShellFixture struct {
	svc    *Service
	gate   *shellGate
	jobs   *fakeShellJobs
	shells *fakeMachineShells
}

// newMachineShellFixture is runner edge on machine m-1 with commands on, which ops holds Run commands on and dev sees
// but holds Deploy to only; ci-laptop is a person's computer runner.
func newMachineShellFixture(t *testing.T) machineShellFixture {
	t.Helper()
	repo := newFakeRunnerRepo()
	repo.runners["r-edge"] = &Runner{ID: "r-edge", Name: "edge", MachineID: "m-1"}
	repo.runners["r-laptop"] = &Runner{ID: "r-laptop", Name: "computer-1", MachineID: "m-1", OwnerUserID: "ops", ComputerID: "c-1"}
	machines := newFakeMachineRepo()
	machines.machines["m-1"] = &Machine{ID: "m-1", Name: "prod", ShellEnabled: true}
	gate := &shellGate{held: map[string][]permissions.Action{
		"ops": {permissions.MachinesSee, permissions.MachinesShell},
		"dev": {permissions.MachinesSee, permissions.MachinesDeploy},
	}}
	jobs := &fakeShellJobs{jobs: map[string]shelljob.Job{}}
	shells := &fakeMachineShells{sinks: map[string]ShellSink{}, started: make(chan string, 16)}
	svc := NewService(repo, &fakeDispatch{}).WithMachines(machines).WithGate(gate).WithShells(jobs, shells)
	t.Cleanup(svc.DrainShells)
	return machineShellFixture{svc: svc, gate: gate, jobs: jobs, shells: shells}
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func (f machineShellFixture) waitStarted(t *testing.T) string {
	t.Helper()
	select {
	case id := <-f.shells.started:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("no command reached the runner")
		return ""
	}
}

func TestMachineShell_RefusalsComeBeforeAnythingIsRecorded(t *testing.T) {
	tests := []struct {
		name, user, runner string
		in                 shelljob.Input
		want               error
	}{
		{"a person who can't see the machine", "mallory", "r-edge", shelljob.Input{Command: "id"}, apperrs.ErrNotFound},
		{"a person without Run commands", "dev", "r-edge", shelljob.Input{Command: "id"}, apperrs.ErrForbidden},
		{"a person's computer runner", "ops", "r-laptop", shelljob.Input{Command: "id"}, apperrs.ErrNotFound},
		{"a runner that does not exist", "ops", "r-gone", shelljob.Input{Command: "id"}, apperrs.ErrNotFound},
		{"a blank command", "ops", "r-edge", shelljob.Input{Command: " "}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMachineShellFixture(t)
			_, err := f.svc.RunCommand(as(tt.user), tt.runner, tt.in)
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, f.jobs.jobs)
		})
	}
}

func TestMachineShell_WithoutRunCommandsNoneAreListedOrRead(t *testing.T) {
	f := newMachineShellFixture(t)
	job, err := f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "uptime"})
	require.NoError(t, err)
	f.waitStarted(t)

	_, _, err = f.svc.ListShellJobs(as("dev"), "m-1", paging.Window{})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.svc.GetShellJob(as("dev"), job.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.svc.GetShellJob(as("mallory"), job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.svc.CancelShellJob(as("dev"), job.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.svc.SetMachineShell(as("dev"), "m-1", false)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = f.svc.OpenShellOutput(as("dev"), job.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	jobs, total, err := f.svc.ListShellJobs(as("ops"), "m-1", paging.Window{})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "uptime", jobs[0].Command)
}

func TestMachineShell_RecordsWhoRanWhatAndHowItEnded(t *testing.T) {
	f := newMachineShellFixture(t)
	job, err := f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "df -h", Adapter: shelljob.AdapterMCP})
	require.NoError(t, err)
	assert.Equal(t, shelljob.Job{
		ID: job.ID, RunnerID: "r-edge", MachineID: "m-1", RequesterID: "ops", Adapter: shelljob.AdapterMCP, Command: "df -h",
		TimeoutSeconds: 60, Status: shelljob.Queued, CreatedAt: job.CreatedAt,
	}, *job)
	id := f.waitStarted(t)
	f.shells.sink(id).Output([]ContainerLogLine{{Stream: LogStreamStdout, Line: "/dev/sda1 40G"}})
	f.shells.sink(id).Done(ShellResult{Status: ShellStatusSucceeded})

	got, err := f.svc.WaitShellJob(as("ops"), id)
	require.NoError(t, err)
	assert.Equal(t, shelljob.Succeeded, got.Status)
	assert.Equal(t, "/dev/sda1 40G\n", got.Output)
	evt := f.jobs.evts[len(f.jobs.evts)-1]
	assert.Equal(t, TopicShellJobChanged, evt.Topic)
	assert.Equal(t, ShellJobChangedEvent{JobID: id, RunnerID: "r-edge", MachineID: "m-1", UserID: "ops", Status: shelljob.Succeeded}, evt.Payload)
}

func TestMachineShell_TheMachinesSwitch(t *testing.T) {
	f := newMachineShellFixture(t)
	running, err := f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "sleep 99"})
	require.NoError(t, err)
	f.waitStarted(t)

	m, err := f.svc.SetMachineShell(as("ops"), "m-1", false)
	require.NoError(t, err)
	assert.False(t, m.ShellEnabled)
	got, err := f.svc.WaitShellJob(as("ops"), running.ID)
	require.NoError(t, err)
	assert.Equal(t, shelljob.Cancelled, got.Status, "turning it off stops what runs there")

	_, err = f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "id"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "off on machine prod")

	_, err = f.svc.SetMachineShell(as("ops"), "m-1", true)
	require.NoError(t, err)
	_, err = f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "id"})
	require.NoError(t, err)
}

func TestMachineShell_ARunnerOfflineFailsTheCommand(t *testing.T) {
	f := newMachineShellFixture(t)
	f.shells.offline = true
	job, err := f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "id"})
	require.NoError(t, err)

	got, err := f.svc.WaitShellJob(as("ops"), job.ID)
	require.NoError(t, err)
	assert.Equal(t, shelljob.Failed, got.Status)
	assert.Equal(t, "the runner is not connected", got.Error)
}

func TestMachineShell_AQueuedCommandIsCheckedAgainAtTheFront(t *testing.T) {
	f := newMachineShellFixture(t)
	for range shelljob.AtOnce {
		_, err := f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "sleep 99"})
		require.NoError(t, err)
		f.waitStarted(t)
	}
	queued, err := f.svc.RunCommand(as("ops"), "r-edge", shelljob.Input{Command: "id"})
	require.NoError(t, err)
	assert.Equal(t, shelljob.Queued, f.jobs.job(queued.ID).Status)

	f.gate.set("ops", permissions.MachinesSee)
	f.shells.mu.Lock()
	sinks := slices.Collect(maps.Values(f.shells.sinks))
	f.shells.mu.Unlock()
	for _, sink := range sinks {
		sink.Done(ShellResult{Status: ShellStatusSucceeded})
	}
	f.svc.DrainShells()
	assert.Equal(t, shelljob.Cancelled, f.jobs.job(queued.ID).Status)
	assert.Equal(t, "access revoked", f.jobs.job(queued.ID).Error)
}

func TestMachineShell_AComputersCommandIsNotAMachines(t *testing.T) {
	f := newMachineShellFixture(t)
	f.jobs.jobs["j-computer"] = shelljob.Job{ID: "j-computer", ComputerID: "c-1", OwnerID: "ops", RequesterID: "ops"}
	_, err := f.svc.GetShellJob(as("ops"), "j-computer")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.svc.GetShellJob(as("ops"), "j-none")
	require.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestMachineShellStartError(t *testing.T) {
	assert.Contains(t, machineShellStartError(ErrCannotRunShell), "too old")
	assert.Equal(t, "the runner is not connected", machineShellStartError(errors.New("dial")))
}

func TestMachineTerminal_RefusesWhatCommandsRefuse(t *testing.T) {
	size := shelljob.TerminalSize{Cols: 80, Rows: 24}
	tests := []struct {
		name, user, runner string
		want               error
	}{
		{"a person who can't see the machine", "mallory", "r-edge", apperrs.ErrNotFound},
		{"a person without Run commands", "dev", "r-edge", apperrs.ErrForbidden},
		{"a person's computer runner", "ops", "r-laptop", apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMachineShellFixture(t)
			_, err := f.svc.OpenTerminal(as(tt.user), tt.runner, size)
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, f.jobs.jobs)
		})
	}
	t.Run("an offline runner is a recorded refusal", func(t *testing.T) {
		f := newMachineShellFixture(t)
		_, err := f.svc.OpenTerminal(as("ops"), "r-edge", size)
		require.ErrorIs(t, err, apperrs.ErrConflict)
		require.Len(t, f.jobs.jobs, 1)
		for _, job := range f.jobs.jobs {
			assert.Equal(t, shelljob.Refused, job.Status)
			assert.Equal(t, "ops", job.RequesterID)
			assert.Equal(t, "the runner is not connected", job.Error)
		}
	})
}

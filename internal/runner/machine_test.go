package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureMachine_NewRunner_CreatesAndLinksMachine(t *testing.T) {
	runners := newFakeRunnerRepo()
	machines := newFakeMachineRepo()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, runners.Create(context.Background(), &Runner{ID: "r-1", Name: "r-1", CreatedAt: now, LastSeen: now}))

	machineID, err := ensureMachine(context.Background(), machines, runners, "r-1", "prod-box", "", now)
	require.NoError(t, err)
	require.NotEmpty(t, machineID)

	m, err := machines.Get(context.Background(), machineID)
	require.NoError(t, err)
	assert.Equal(t, "prod-box", m.Name)
	assert.Equal(t, "prod-box", m.ReportedHostname)
	assert.Equal(t, defaultStackRoot, m.StackRoot)

	r, err := runners.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, machineID, r.MachineID)
}

func TestEnsureMachine_NoReportedName_FallsBackToRunnerID(t *testing.T) {
	runners := newFakeRunnerRepo()
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, runners.Create(context.Background(), &Runner{ID: "r-1", CreatedAt: now, LastSeen: now}))

	machineID, err := ensureMachine(context.Background(), machines, runners, "r-1", "", "", now)
	require.NoError(t, err)

	m, err := machines.Get(context.Background(), machineID)
	require.NoError(t, err)
	assert.Equal(t, "r-1", m.Name)
}

func TestEnsureMachine_TwoRunnersReportingSameName_SharesOneMachine(t *testing.T) {
	runners := newFakeRunnerRepo()
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, runners.Create(context.Background(), &Runner{ID: "r-1", CreatedAt: now, LastSeen: now}))
	require.NoError(t, runners.Create(context.Background(), &Runner{ID: "r-2", CreatedAt: now, LastSeen: now}))

	m1, err := ensureMachine(context.Background(), machines, runners, "r-1", "prod-box", "", now)
	require.NoError(t, err)
	m2, err := ensureMachine(context.Background(), machines, runners, "r-2", "prod-box", "", now)
	require.NoError(t, err)

	assert.Equal(t, m1, m2)
	ms, err := machines.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, ms, 1)
}

// A reconnecting runner keeps its machine and id even if the reported hostname changes; renaming is a UI
// action only (issue 05).
func TestEnsureMachine_ExistingRunner_KeepsMachineAndRecordsReportedHint(t *testing.T) {
	runners := newFakeRunnerRepo()
	machines := newFakeMachineRepo()
	first := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, runners.Create(context.Background(), &Runner{ID: "r-1", CreatedAt: first, LastSeen: first}))

	machineID, err := ensureMachine(context.Background(), machines, runners, "r-1", "prod-box", "", first)
	require.NoError(t, err)

	// Owner renames the machine through the UI; the id must survive a reconnect.
	require.NoError(t, machines.Rename(context.Background(), machineID, "prod-primary"))

	second := first.Add(time.Hour)
	gotID, err := ensureMachine(context.Background(), machines, runners, "r-1", "prod-box-renamed-by-dhcp", "", second)
	require.NoError(t, err)
	assert.Equal(t, machineID, gotID)

	m, err := machines.Get(context.Background(), machineID)
	require.NoError(t, err)
	assert.Equal(t, "prod-primary", m.Name, "the UI-chosen name must survive a reconnect")
	assert.Equal(t, "prod-box-renamed-by-dhcp", m.ReportedHostname)
	assert.Equal(t, second, m.LastSeen)
}

func TestEnsureMachine_UnknownRunner_Errors(t *testing.T) {
	runners := newFakeRunnerRepo()
	machines := newFakeMachineRepo()
	_, err := ensureMachine(context.Background(), machines, runners, "missing", "box", "", time.Now())
	require.Error(t, err)
}

func TestService_MachineCRUD(t *testing.T) {
	machines := newFakeMachineRepo()
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(machines)
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", StackRoot: defaultStackRoot, FirstSeen: now, LastSeen: now}))

	list, err := svc.ListMachines(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)

	got, err := svc.GetMachine(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, "prod", got.Name)

	updated, err := svc.UpdateMachine(context.Background(), "m-1", "prod-renamed", "/srv/nexul")
	require.NoError(t, err)
	assert.Equal(t, "prod-renamed", updated.Name)
	assert.Equal(t, "/srv/nexul", updated.StackRoot)

	// An empty field leaves that setting unchanged.
	unchanged, err := svc.UpdateMachine(context.Background(), "m-1", "", "")
	require.NoError(t, err)
	assert.Equal(t, "prod-renamed", unchanged.Name)
	assert.Equal(t, "/srv/nexul", unchanged.StackRoot)
}

func TestService_MachineCRUD_Unconfigured(t *testing.T) {
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{})
	_, err := svc.GetMachine(context.Background(), "m-1")
	require.Error(t, err)
	_, err = svc.UpdateMachine(context.Background(), "m-1", "x", "")
	require.Error(t, err)
	_, err = svc.Discover(context.Background(), "m-1")
	require.Error(t, err)
	list, err := svc.ListMachines(context.Background())
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestService_ListRunners_ResolvesMachineName(t *testing.T) {
	runners := newFakeRunnerRepo()
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
	require.NoError(t, runners.Create(context.Background(), &Runner{ID: "r-1", Name: "r-1", MachineID: "m-1", CreatedAt: now, LastSeen: now}))

	svc := NewService(runners, &fakeDispatch{}).WithMachines(machines)
	views, err := svc.ListRunners(context.Background())
	require.NoError(t, err)
	require.Len(t, views, 1)
	assert.Equal(t, "prod", views[0].Machine)
}

func TestService_Discover_ResolvesMachineNameAndDelegates(t *testing.T) {
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
	var gotMachine string
	dispatch := &fakeDispatch{discoverFn: func(_ context.Context, machine string, _ time.Duration) (DiscoverReport, error) {
		gotMachine = machine
		return DiscoverReport{Containers: []DiscoveredContainer{{Name: "web"}}}, nil
	}}
	svc := NewService(newFakeRunnerRepo(), dispatch).WithMachines(machines)

	report, err := svc.Discover(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, "prod", gotMachine)
	require.Len(t, report.Containers, 1)
}

type fakeManagedLookup struct{ names []string }

func (f fakeManagedLookup) ListContainerNamesByMachine(context.Context, string) ([]string, error) {
	return f.names, nil
}

func TestService_DiscoverUnmanaged_DropsTrackedContainers(t *testing.T) {
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
	dispatch := &fakeDispatch{discoverFn: func(context.Context, string, time.Duration) (DiscoverReport, error) {
		return DiscoverReport{Containers: []DiscoveredContainer{{Name: "web"}, {Name: "api-web-1"}, {Name: "hand-run"}}}, nil
	}}
	svc := NewService(newFakeRunnerRepo(), dispatch).WithMachines(machines).WithManaged(fakeManagedLookup{names: []string{"web", "api-web-1"}})

	report, err := svc.DiscoverUnmanaged(context.Background(), "m-1")
	require.NoError(t, err)
	require.Len(t, report.Containers, 1)
	assert.Equal(t, "hand-run", report.Containers[0].Name)

	raw, err := svc.Discover(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Len(t, raw.Containers, 3, "Discover stays raw for the import use case")
}

func TestEnsureMachine_NewMachineTakesTheReportedStackRoot(t *testing.T) {
	machines, runners := newFakeMachineRepo(), newFakeRunnerRepo()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	require.NoError(t, runners.Create(t.Context(), &Runner{ID: "r-1", Name: "r-1", CreatedAt: now, LastSeen: now}))

	machineID, err := ensureMachine(t.Context(), machines, runners, "r-1", "laptop", "/Users/onik/nexul", now)

	require.NoError(t, err)
	m, err := machines.Get(t.Context(), machineID)
	require.NoError(t, err)
	assert.Equal(t, "/Users/onik/nexul", m.StackRoot)
}

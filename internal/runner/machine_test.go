package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

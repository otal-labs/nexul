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
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(NewService(newFakeRunnerRepo(), &fakeDispatch{}))
	require.Len(t, tools, 6)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{
		"runner_list", "runner_queue", "machine_list", "machine_discover",
		"instance_upgrade_status", "instance_upgrade",
	}, names)
}

func TestMCPTools_RunnerList(t *testing.T) {
	repo := newFakeRunnerRepo()
	require.NoError(t, repo.Create(context.Background(), &Runner{ID: "r-1"}))
	svc := NewService(repo, &fakeDispatch{runners: []RunnerStatus{{RunnerID: "r-1"}}})
	call := toolByName(t, MCPTools(svc), "runner_list").Call
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.Len(t, got.([]RunnerView), 1)
}

func TestMCPTools_RunnerQueue(t *testing.T) {
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{queue: []QueuedJob{{ID: "d-1", Kind: RequestDeploy, Service: "api"}}})
	call := toolByName(t, MCPTools(svc), "runner_queue").Call
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	require.Len(t, got.([]QueuedJob), 1)
	assert.Equal(t, "api", got.([]QueuedJob)[0].Service)
}

func TestMCPTools_MachineList(t *testing.T) {
	machines := newFakeMachineRepo()
	now := time.Now().UTC()
	require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
	svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(machines)
	call := toolByName(t, MCPTools(svc), "machine_list").Call
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	require.Len(t, got.([]*Machine), 1)
	assert.Equal(t, "prod", got.([]*Machine)[0].Name)
}

func TestMCPTools_MachineDiscover(t *testing.T) {
	t.Run("happy path groups the report", func(t *testing.T) {
		machines := newFakeMachineRepo()
		now := time.Now().UTC()
		require.NoError(t, machines.Create(context.Background(), &Machine{ID: "m-1", Name: "prod", FirstSeen: now, LastSeen: now}))
		dispatch := &fakeDispatch{discoverFn: func(context.Context, string, time.Duration) (DiscoverReport, error) {
			return DiscoverReport{Containers: []DiscoveredContainer{{Name: "web", Labels: map[string]string{composeProjectLabel: "myapp"}}}}, nil
		}}
		svc := NewService(newFakeRunnerRepo(), dispatch).WithMachines(machines)
		call := toolByName(t, MCPTools(svc), "machine_discover").Call
		got, err := call(context.Background(), map[string]any{"machine_id": "m-1"})
		require.NoError(t, err)
		grouped, ok := got.(GroupedDiscovery)
		require.True(t, ok)
		require.Len(t, grouped.Stacks, 1)
		assert.Equal(t, "myapp", grouped.Stacks[0].Project)
	})
	t.Run("missing machine id is invalid", func(t *testing.T) {
		svc := NewService(newFakeRunnerRepo(), &fakeDispatch{})
		call := toolByName(t, MCPTools(svc), "machine_discover").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown machine is not found", func(t *testing.T) {
		svc := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(newFakeMachineRepo())
		call := toolByName(t, MCPTools(svc), "machine_discover").Call
		_, err := call(context.Background(), map[string]any{"machine_id": "ghost"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_InstanceUpgradeStatus(t *testing.T) {
	withVersion(t, "dev")
	srv := fakeGitHub(t, "v0.2.0", "x")
	svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})
	call := toolByName(t, MCPTools(svc), "instance_upgrade_status").Call

	_, err := call(identity.WithActor(context.Background(), identity.Actor{ID: "member-1"}), map[string]any{})
	require.ErrorIs(t, err, apperrs.ErrForbidden, "a non-admin gets the same answer the settings page gives")

	got, err := call(asAdmin(), map[string]any{})
	require.NoError(t, err)
	status, ok := got.(UpgradeStatus)
	require.True(t, ok)
	assert.Equal(t, "dev build", status.Reason)
}

func TestMCPTools_InstanceUpgrade(t *testing.T) {
	t.Run("provenance carries the mcp suffix", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: instanceRunnerID}}}
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), dispatch)
		call := toolByName(t, MCPTools(svc), "instance_upgrade").Call

		got, err := call(asAdmin(), map[string]any{})
		require.NoError(t, err)
		upgrade, ok := got.(Upgrade)
		require.True(t, ok)
		assert.Equal(t, "admin-1:mcp", upgrade.RequestedBy)
	})

	t.Run("blocked can_upgrade surfaces as a conflict", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0", "x")
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})
		call := toolByName(t, MCPTools(svc), "instance_upgrade").Call

		_, err := call(asAdmin(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})

	t.Run("a non-admin cannot upgrade", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: instanceRunnerID}}}
		svc := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), dispatch)
		call := toolByName(t, MCPTools(svc), "instance_upgrade").Call

		_, err := call(identity.WithActor(context.Background(), identity.Actor{ID: "member-1"}), map[string]any{})
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
}

func toolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

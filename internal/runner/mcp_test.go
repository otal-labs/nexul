package runner

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

func callTool(ctx context.Context, t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func asMember() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "member-1"})
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(NewService(newFakeRunnerRepo(), &fakeDispatch{})) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
	}
	assert.Equal(t, []string{"machine_list", "machine_discover", "instance_get", "instance_upgrade"}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	withVersion(t, "dev")
	srv := fakeGitHub(t, "v0.2.0", "x")
	upgrades := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{}).WithMachines(newFakeMachineRepo())
	tests := []struct {
		name string
		ctx  context.Context
		tool string
		args string
		want error
		msg  string
	}{
		{"machine_list rejects an unknown key", asAdmin(), "machine_list", `{"machine":"prod"}`, apperrs.ErrInvalid, ""},
		{"machine_discover needs an id", asAdmin(), "machine_discover", `{}`, apperrs.ErrInvalid, ""},
		{"machine_discover of a missing machine", asAdmin(), "machine_discover", `{"id":"ghost"}`, apperrs.ErrNotFound, "machine_list"},
		{"instance_get needs instance:read", asMember(), "instance_get", `{}`, apperrs.ErrForbidden, ""},
		{"instance_upgrade needs instance:write", asMember(), "instance_upgrade", `{}`, apperrs.ErrForbidden, ""},
		{"instance_upgrade refuses a dev build", asAdmin(), "instance_upgrade", `{}`, apperrs.ErrConflict, "dev build"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(tt.ctx, t, upgrades, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

func TestMachineList_GroupsRunnersUnderTheirMachine(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	machines := newFakeMachineRepo()
	require.NoError(t, machines.Create(ctx, &Machine{ID: "m-1", Name: "prod", StackRoot: "/data/nexul", LastSeen: now}))
	require.NoError(t, machines.Create(ctx, &Machine{ID: "m-2", Name: "edge", LastSeen: now}))
	runners := newFakeRunnerRepo()
	require.NoError(t, runners.Create(ctx, &Runner{ID: "r-1", Name: "alpha", Version: "v0.1.6", Connected: true, MachineID: "m-1"}))
	require.NoError(t, runners.Create(ctx, &Runner{ID: "r-new", Name: "fresh"}))
	dispatch := &fakeDispatch{
		runners: []RunnerStatus{{RunnerID: "r-1", RunningJob: &RunningJob{ID: "d-1", Kind: RequestDeploy, Service: "api"}}},
		queue:   []QueuedJob{{ID: "d-2", Kind: RequestBuild, Service: "web", Target: "prod"}},
	}
	s := NewService(runners, dispatch).WithMachines(machines)

	got, err := callTool(ctx, t, s, "machine_list", `{}`)
	require.NoError(t, err)
	list := got.(machineList)
	require.Equal(t, 2, list.Total)
	byName := map[string]machineResult{}
	for _, m := range list.Items {
		byName[m.Name] = m
	}
	require.Len(t, byName["prod"].Runners, 1)
	assert.Equal(t, "v0.1.6", byName["prod"].Runners[0].Version)
	assert.Equal(t, &jobResult{ID: "d-1", Kind: RequestDeploy, Stack: "api"}, byName["prod"].Runners[0].RunningJob)
	assert.Empty(t, byName["edge"].Runners)
	require.Len(t, list.UnassignedRunners, 1)
	assert.Equal(t, "r-new", list.UnassignedRunners[0].ID)
	assert.Equal(t, []jobResult{{ID: "d-2", Kind: RequestBuild, Stack: "web", Machine: "prod"}}, list.Queue)
	b, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"target"`)
	assert.NotContains(t, string(b), `"service"`)
}

type fakeAutomationsHosts struct {
	hosts []AutomationsHost
	err   error
}

func (f fakeAutomationsHosts) ListAutomationsHosts(context.Context) ([]AutomationsHost, error) {
	return f.hosts, f.err
}

func TestMachineList_FilesAutomationsHostsUnderTheirMachine(t *testing.T) {
	ctx := t.Context()
	machines := newFakeMachineRepo()
	require.NoError(t, machines.Create(ctx, &Machine{ID: "m-1", Name: "prod"}))
	require.NoError(t, machines.Create(ctx, &Machine{ID: "m-2", Name: "edge"}))
	base := NewService(newFakeRunnerRepo(), &fakeDispatch{}).WithMachines(machines)

	_, err := callTool(ctx, t, base.WithAutomationsHosts(fakeAutomationsHosts{err: errors.New("db down")}), "machine_list", `{}`)
	require.Error(t, err, "a failed host lookup fails the list")

	hosts := fakeAutomationsHosts{hosts: []AutomationsHost{
		{ID: "h-1", Name: "jobs", Machine: "prod", Version: "v0.3.0", Connected: true},
		{ID: "h-2", Name: "laptop", Machine: "laptop"},
	}}
	got, err := callTool(ctx, t, base.WithAutomationsHosts(hosts), "machine_list", `{}`)
	require.NoError(t, err)
	list := got.(machineList)
	byName := map[string]machineResult{}
	for _, m := range list.Items {
		byName[m.Name] = m
	}
	assert.Equal(t, []automationsHostResult{{ID: "h-1", Name: "jobs", Version: "v0.3.0", Connected: true}}, byName["prod"].AutomationsHosts)
	assert.Equal(t, []automationsHostResult{}, byName["edge"].AutomationsHosts)
	assert.Equal(t, []automationsHostResult{{ID: "h-2", Name: "laptop"}}, list.UnassignedAutomationsHosts)
}

func TestMachineList_EmptyQueueIsAList(t *testing.T) {
	got, err := callTool(t.Context(), t, NewService(newFakeRunnerRepo(), &fakeDispatch{}), "machine_list", `{}`)
	require.NoError(t, err)
	b, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{"items":[],"total":0,"has_more":false,"queue":[]}`, string(b))
}

func TestMachineDiscover_DropsDockerLabels(t *testing.T) {
	machines := newFakeMachineRepo()
	require.NoError(t, machines.Create(t.Context(), &Machine{ID: "m-1", Name: "prod"}))
	dispatch := &fakeDispatch{discoverFn: func(context.Context, string, time.Duration) (DiscoverReport, error) {
		return DiscoverReport{Containers: []DiscoveredContainer{
			{Name: "web", Image: "nginx", Labels: map[string]string{composeProjectLabel: "myapp", "com.docker.compose.config-hash": "abc"}},
			{Name: "solo", Image: "redis", Labels: map[string]string{"maintainer": "x"}},
		}}, nil
	}}
	s := NewService(newFakeRunnerRepo(), dispatch).WithMachines(machines)

	got, err := callTool(t.Context(), t, s, "machine_discover", `{"id":"m-1"}`)
	require.NoError(t, err)
	grouped := got.(GroupedDiscovery)
	require.Len(t, grouped.Stacks, 1)
	assert.Equal(t, "myapp", grouped.Stacks[0].Project)
	require.Len(t, grouped.Standalone, 1)
	b, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "labels")
}

func TestInstanceTools_AsAdmin(t *testing.T) {
	t.Run("instance_get reports why no upgrade can start", func(t *testing.T) {
		withVersion(t, "dev")
		srv := fakeGitHub(t, "v0.2.0", "x")
		s := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})
		got, err := callTool(asAdmin(), t, s, "instance_get", `{}`)
		require.NoError(t, err)
		status := got.(UpgradeStatus)
		assert.False(t, status.CanUpgrade)
		assert.Equal(t, "dev build", status.Reason)
	})
	t.Run("instance_get with refresh reads past the release cache", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		s := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), &fakeDispatch{})
		got, err := callTool(asAdmin(), t, s, "instance_get", `{"refresh":true}`)
		require.NoError(t, err)
		assert.Equal(t, "v0.2.1", got.(UpgradeStatus).Latest.Version)
	})
	t.Run("refreshing without a release client is a no-op", func(t *testing.T) {
		assert.NotPanics(t, func() { (&Service{}).RefreshReleases() })
	})
	t.Run("instance_upgrade records the mcp source", func(t *testing.T) {
		withVersion(t, "v0.2.0")
		srv := fakeGitHub(t, "v0.2.1", "x")
		dispatch := &fakeDispatch{runners: []RunnerStatus{{RunnerID: "instance-id", Name: instanceRunnerName}}}
		s := newUpgradeService(srv.URL, newFakeUpgradeRepo(), newFakeBus(), dispatch)
		got, err := callTool(asAdmin(), t, s, "instance_upgrade", `{}`)
		require.NoError(t, err)
		upgrade := got.(Upgrade)
		assert.Equal(t, "admin-1:mcp", upgrade.RequestedBy)
		assert.Equal(t, "v0.2.1", upgrade.ToVersion)
	})
}

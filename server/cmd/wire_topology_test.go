package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/workspace"
)

// topologyFixture adds a second workspace with its own stack to the permission fixture, a container for each stack,
// and the infra stack as a gateway routing a hostname to the default workspace's web container.
func topologyFixture(t *testing.T) permFixture {
	t.Helper()
	f := newPermFixture(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, f.store.Workspaces.Create(ctx, &tenancy.Workspace{ID: "workspace-other", Name: "Other", Slug: "other", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: "project-other", Name: "Other", Prefix: "OTH", WorkspaceID: "workspace-other", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-other", ProjectID: "project-other", Name: "api", Slug: "api", Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	for id, stack := range map[string]string{"c-web": f.stack, "c-cf": f.infra, "c-other": "stack-other"} {
		require.NoError(t, f.store.Services.Create(ctx, &deploy.Container{ID: id, StackID: stack, Name: id, Status: deploy.ServiceStatusRunning}))
	}
	require.NoError(t, f.store.DNS.SaveGateway(ctx, dns.Gateway{ID: "gw-1", Kind: dns.GatewayTunnel, DockerNetwork: "net", Machine: "m1", ServiceID: f.infra, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.DNS.SaveExposure(ctx, dns.Exposure{ID: "exp-1", GatewayID: "gw-1", Hostname: "web.example.com", ServiceID: "c-web", Port: 80, ZoneID: "z", Zone: "example.com", CreatedAt: now, UpdatedAt: now}))
	return f
}

func TestTopologyScope_AWorkspaceSeesItsServicesAndTheGatewaysRoutingToThem(t *testing.T) {
	f := topologyFixture(t)
	scope := topologyScope{stacks: f.store.Stacks, services: f.store.Services, dns: f.store.DNS}

	own, err := scope.Services(t.Context(), "workspace-default")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{f.stack: true, "c-web": true, f.infra: true, "c-cf": true}, own)

	other, err := scope.Services(t.Context(), "workspace-other")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"stack-other": true, "c-other": true}, other, "the gateway routes nothing here")
}

func TestTopologyLiveHandler_ARegistryChangePushesEachWorkspaceItsOwnCanvas(t *testing.T) {
	f := topologyFixture(t)
	topo := f.svc.topoSvc
	for _, id := range []string{"c-web", "c-other"} {
		_, err := topo.AddServiceNode(t.Context(), topology.Node{ID: id, Data: topology.NodeData{Name: id, Status: topology.ServiceRunning}})
		require.NoError(t, err)
	}
	pushed := map[string][]string{}
	publish := func(_ context.Context, topic string, payload any) error {
		frame := payload.(topologyFrame)
		assert.Equal(t, topicTopologyCanvas, topic)
		for _, n := range frame.Nodes {
			pushed[frame.WorkspaceID] = append(pushed[frame.WorkspaceID], n.ID)
		}
		return nil
	}
	handle := topologyLiveHandler(topo, f.store.Workspaces, publish)

	raw, err := json.Marshal(topology.UpdatedEvent{Environment: "default"})
	require.NoError(t, err)
	require.NoError(t, handle(t.Context(), eventbus.Event{Topic: topology.TopicUpdated, Payload: raw}))
	assert.Equal(t, []string{"c-web"}, pushed["workspace-default"])
	assert.Equal(t, []string{"c-other"}, pushed["workspace-other"])

	clear(pushed)
	raw, err = json.Marshal(topology.UpdatedEvent{Environment: "workspace-other", WorkspaceID: "workspace-other", Canvas: topology.Canvas{Nodes: []topology.Node{{ID: "net"}}}})
	require.NoError(t, err)
	require.NoError(t, handle(t.Context(), eventbus.Event{Topic: topology.TopicUpdated, Payload: raw}))
	assert.Equal(t, map[string][]string{"workspace-other": {"net"}}, pushed, "a workspace's own change reaches only it")

	assert.Error(t, handle(t.Context(), eventbus.Event{Topic: topology.TopicUpdated, Payload: []byte("{")}))
}

func TestLiveAudience_ATopologyFrameReachesItsWorkspacesReaders(t *testing.T) {
	f := newPermFixture(t)
	a := liveAudience{access: f.svc.accessSvc}
	raw, err := json.Marshal(topologyFrame{WorkspaceID: "workspace-default"})
	require.NoError(t, err)
	for user, want := range map[string]bool{uOwner: true, uReader: true, uPlain: false, uOutsider: false} {
		assert.Equal(t, want, a.allows(as(user), topicTopologyCanvas, json.RawMessage(raw)), "as %s", user)
	}
	assert.False(t, a.allows(as(uOwner), topicTopologyCanvas, json.RawMessage(`{"nodes":[]}`)), "a frame naming no workspace reaches nobody")
}

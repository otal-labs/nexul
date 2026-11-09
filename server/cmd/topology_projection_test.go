package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/topology"
)

func TestContainerNodeName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		containers int
		want       string
	}{
		{"one container is the stack alone", 1, "shop"},
		{"none left is the stack alone", 0, "shop"},
		{"several containers are stack/service", 2, "shop/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containerNodeName("shop", "api", tt.containers))
		})
	}
}

func TestContainerNodeStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		container    deploy.Container
		deployStatus deploy.Status
		want         topology.ServiceStatus
		wantAddress  string
	}{
		{"healthy", deploy.Container{Status: deploy.ServiceStatusHealthy}, deploy.StatusHealthy, topology.ServiceHealthy, ""},
		{"running", deploy.Container{Status: deploy.ServiceStatusRunning}, deploy.StatusHealthy, topology.ServiceRunning, ""},
		{"stopped", deploy.Container{Status: deploy.ServiceStatusStopped}, deploy.StatusHealthy, topology.ServiceStopped, ""},
		{"exited", deploy.Container{Status: deploy.ServiceStatusExited}, deploy.StatusHealthy, topology.ServiceFailed, ""},
		{"never reported under a healthy deploy", deploy.Container{Status: deploy.ServiceStatusPending}, deploy.StatusHealthy, topology.ServiceRunning, ""},
		{"never reported under a failed deploy", deploy.Container{Status: deploy.ServiceStatusPending}, deploy.StatusFailed, topology.ServiceFailed, ""},
		{"first network's address", deploy.Container{Status: deploy.ServiceStatusRunning, Networks: []deploy.Network{{Name: "a", Address: "10.0.0.2"}, {Name: "b", Address: "10.0.1.2"}}}, deploy.StatusHealthy, topology.ServiceRunning, "10.0.0.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, address := containerNodeStatus(&tt.container, tt.deployStatus)
			assert.Equal(t, tt.want, status)
			assert.Equal(t, tt.wantAddress, address)
		})
	}
}

// projectionFixture seeds stack "stack-shop" in the default workspace; its containers are added per test.
func projectionFixture(t *testing.T, strategy deploy.Strategy) permFixture {
	t.Helper()
	f := newPermFixture(t)
	now := time.Now()
	require.NoError(t, f.store.Stacks.Create(t.Context(), &deploy.Stack{ID: "stack-shop", ProjectID: "project-general", Name: "shop", Slug: "shop", Machine: "m1", Strategy: strategy, CreatedAt: now, UpdatedAt: now}))
	return f
}

func addContainer(t *testing.T, f permFixture, id, name string, status deploy.ServiceStatus, networks ...deploy.Network) {
	t.Helper()
	require.NoError(t, f.store.Services.Create(t.Context(), &deploy.Container{ID: id, StackID: "stack-shop", Name: name, Status: status, Networks: networks}))
}

func addDeploy(t *testing.T, f permFixture, id string, status deploy.Status) {
	t.Helper()
	now := time.Now()
	require.NoError(t, f.store.Deploys.Create(t.Context(), &deploy.Deploy{ID: id, StackID: "stack-shop", Service: "shop", Status: status, CreatedAt: now, UpdatedAt: now}))
}

// canvasNodes is what the default workspace's canvas shows, keyed by container id.
func canvasNodes(t *testing.T, f permFixture) map[string]topology.NodeData {
	t.Helper()
	canvas, err := f.svc.topoSvc.Get(t.Context(), "workspace-default")
	require.NoError(t, err)
	shown := make(map[string]topology.NodeData, len(canvas.Nodes))
	for _, n := range canvas.Nodes {
		shown[n.ID] = n.Data
	}
	return shown
}

// badge is what a node shows: its status and, once reported, its address.
type badge struct {
	status  topology.ServiceStatus
	address string
}

// canvasBadges is what the default workspace's canvas shows, keyed by node name.
func canvasBadges(t *testing.T, f permFixture) map[string]badge {
	t.Helper()
	shown := map[string]badge{}
	for _, data := range canvasNodes(t, f) {
		shown[data.Name] = badge{data.Status, data.Address}
	}
	return shown
}

func event(t *testing.T, topic string, payload any) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return eventbus.Event{Topic: topic, Payload: raw}
}

func TestTopologyStackCreatedHandler(t *testing.T) {
	t.Parallel()
	one := []deploy.Container{{ID: "c-web", Name: "web", Status: deploy.ServiceStatusPending}}
	several := []deploy.Container{
		{ID: "c-api", Name: "api", Status: deploy.ServiceStatusHealthy, Networks: []deploy.Network{{Name: "shop_default", Address: "172.30.0.2"}}},
		{ID: "c-web", Name: "web", Status: deploy.ServiceStatusStopped},
	}
	tests := []struct {
		name     string
		services []deploy.Container
		want     map[string]topology.NodeData
	}{
		{"a lone pending container is named for the stack and shows running", one, map[string]topology.NodeData{
			"c-web": {Name: "shop", Runtime: "docker", Status: topology.ServiceRunning, ServiceID: "c-web"},
		}},
		{"adopted containers keep their observed status and address under stack/service names", several, map[string]topology.NodeData{
			"c-api": {Name: "shop/api", Runtime: "docker", Status: topology.ServiceHealthy, Address: "172.30.0.2", ServiceID: "c-api"},
			"c-web": {Name: "shop/web", Runtime: "docker", Status: topology.ServiceStopped, ServiceID: "c-web"},
		}},
		{"a stack with no containers adds nothing", nil, map[string]topology.NodeData{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := projectionFixture(t, deploy.StrategyCompose)
			for _, c := range tt.services {
				addContainer(t, f, c.ID, c.Name, c.Status, c.Networks...)
			}
			ev := event(t, deploy.TopicStackCreated, deploy.StackEvent{Stack: deploy.Stack{ID: "stack-shop", Name: "shop"}, Services: tt.services})
			handle := topologyStackCreatedHandler(f.svc.topoSvc)

			require.NoError(t, handle(t.Context(), ev))
			assert.Equal(t, tt.want, canvasNodes(t, f))

			require.NoError(t, handle(t.Context(), ev), "a redelivery finds the nodes already anchored")
			assert.Equal(t, tt.want, canvasNodes(t, f))
		})
	}
}

func TestTopologyStackCreatedHandler_ABadPayloadIsFatal(t *testing.T) {
	t.Parallel()
	f := projectionFixture(t, deploy.StrategyRun)
	err := topologyStackCreatedHandler(f.svc.topoSvc)(t.Context(), eventbus.Event{Topic: deploy.TopicStackCreated, Payload: []byte("{")})
	require.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestTopologyStackCreatedHandler_ANodeTheRegistryRefusesFailsTheEvent(t *testing.T) {
	t.Parallel()
	f := projectionFixture(t, deploy.StrategyRun)
	ev := event(t, deploy.TopicStackCreated, deploy.StackEvent{Stack: deploy.Stack{ID: "stack-shop", Name: "shop"}, Services: []deploy.Container{{ID: "", Name: "web"}}})
	require.Error(t, topologyStackCreatedHandler(f.svc.topoSvc)(t.Context(), ev), "a container with no id cannot be anchored")
}

func TestTopologyStackUpdatedHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		containers map[string]string
		want       map[string]string
	}{
		{"a rename reaches a lone container", map[string]string{"c-web": "web"}, map[string]string{"c-web": "store"}},
		{"a rename reaches every container of a stack", map[string]string{"c-api": "api", "c-web": "web"}, map[string]string{"c-api": "store/api", "c-web": "store/web"}},
		{"a stack with no containers renames nothing", nil, map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := projectionFixture(t, deploy.StrategyCompose)
			for id, name := range tt.containers {
				addContainer(t, f, id, name, deploy.ServiceStatusRunning)
				_, err := f.svc.topoSvc.AddServiceNode(t.Context(), topology.Node{ID: id, Data: topology.NodeData{Name: "shop", Runtime: "docker", Status: topology.ServiceRunning}})
				require.NoError(t, err)
			}
			ev := event(t, deploy.TopicStackUpdated, deploy.StackEvent{Stack: deploy.Stack{ID: "stack-shop", Name: "store"}})
			handle := topologyStackUpdatedHandler(f.svc.deploySvc, f.svc.topoSvc)

			require.NoError(t, handle(t.Context(), ev))
			require.NoError(t, handle(t.Context(), ev), "a redelivery renames to the same names")
			got := map[string]string{}
			for id, data := range canvasNodes(t, f) {
				got[id] = data.Name
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTopologyStackUpdatedHandler_ErrorPaths(t *testing.T) {
	t.Parallel()
	f := projectionFixture(t, deploy.StrategyRun)
	handle := topologyStackUpdatedHandler(f.svc.deploySvc, f.svc.topoSvc)

	require.ErrorIs(t, handle(t.Context(), eventbus.Event{Topic: deploy.TopicStackUpdated, Payload: []byte("{")}), apperrs.ErrFatal)

	addContainer(t, f, "c-web", "web", deploy.ServiceStatusRunning)
	err := handle(t.Context(), event(t, deploy.TopicStackUpdated, deploy.StackEvent{Stack: deploy.Stack{ID: "stack-shop", Name: ""}}))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a stack renamed to nothing leaves its node's name alone and fails the event")
}

func TestTopologyStackDeletedHandler(t *testing.T) {
	t.Parallel()
	f := projectionFixture(t, deploy.StrategyCompose)
	for _, id := range []string{"c-api", "c-web"} {
		addContainer(t, f, id, id, deploy.ServiceStatusRunning)
		_, err := f.svc.topoSvc.AddServiceNode(t.Context(), topology.Node{ID: id, Data: topology.NodeData{Name: id, Runtime: "docker", Status: topology.ServiceRunning}})
		require.NoError(t, err)
	}
	handle := topologyStackDeletedHandler(f.svc.topoSvc)

	ev := event(t, deploy.TopicStackDeleted, deploy.StackDeletedEvent{ID: "stack-shop", ServiceIDs: []string{"c-api", "never-anchored"}})
	require.NoError(t, handle(t.Context(), ev), "a container that never had a node is skipped")
	assert.Equal(t, []string{"c-web"}, mapKeys(canvasNodes(t, f)))

	require.NoError(t, handle(t.Context(), ev), "a redelivery finds the nodes already gone")
	assert.Equal(t, []string{"c-web"}, mapKeys(canvasNodes(t, f)))

	require.ErrorIs(t, handle(t.Context(), eventbus.Event{Topic: deploy.TopicStackDeleted, Payload: []byte("{")}), apperrs.ErrFatal)
	require.ErrorIs(t, handle(t.Context(), event(t, deploy.TopicStackDeleted, deploy.StackDeletedEvent{ID: "stack-shop", ServiceIDs: []string{""}})), apperrs.ErrInvalid)
}

func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestTopologyDeployStatusHandler_Transitions(t *testing.T) {
	t.Parallel()
	net := func(addr string) []deploy.Network { return []deploy.Network{{Name: "shop_default", Address: addr}} }
	tests := []struct {
		name string
		// before is the stack's containers as the registry knew them (all anchored on the canvas).
		before     []deploy.Container
		strategy   deploy.Strategy
		from       deploy.Status
		event      deploy.DeployStatusChangedEvent
		wantStatus deploy.Status
		want       map[string]badge
	}{
		{
			name:       "healthy with a report badges each container with its own status and address",
			before:     []deploy.Container{{ID: "c-api", Name: "api", Status: deploy.ServiceStatusPending}, {ID: "c-web", Name: "web", Status: deploy.ServiceStatusPending}},
			strategy:   deploy.StrategyCompose,
			from:       deploy.StatusRunning,
			event:      deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "healthy", Services: []deploy.ObservedService{{Name: "api", ContainerName: "shop-api-1", Status: "healthy", Networks: net("172.30.0.2")}, {Name: "web", ContainerName: "shop-web-1", Status: "running", Networks: net("172.30.0.3")}}},
			wantStatus: deploy.StatusHealthy,
			want: map[string]badge{
				"shop/api": {topology.ServiceHealthy, "172.30.0.2"},
				"shop/web": {topology.ServiceRunning, "172.30.0.3"},
			},
		},
		{
			name:       "healthy straight from pending with no report shows the container running",
			before:     []deploy.Container{{ID: "c-web", Name: "web", Status: deploy.ServiceStatusPending}},
			strategy:   deploy.StrategyRun,
			from:       deploy.StatusPending,
			event:      deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "healthy"},
			wantStatus: deploy.StatusHealthy,
			want:       map[string]badge{"shop": {status: topology.ServiceRunning}},
		},
		{
			name:       "failed with no report falls back to the deploy's own status on every container",
			before:     []deploy.Container{{ID: "c-api", Name: "api", Status: deploy.ServiceStatusPending}, {ID: "c-web", Name: "web", Status: deploy.ServiceStatusPending}},
			strategy:   deploy.StrategyCompose,
			from:       deploy.StatusRunning,
			event:      deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "failed", Error: "build failed"},
			wantStatus: deploy.StatusFailed,
			want: map[string]badge{
				"shop/api": {status: topology.ServiceFailed},
				"shop/web": {status: topology.ServiceFailed},
			},
		},
		{
			name:       "failed with a report keeps the running container running and fails the one that exited",
			before:     []deploy.Container{{ID: "c-api", Name: "api", Status: deploy.ServiceStatusRunning}, {ID: "c-web", Name: "web", Status: deploy.ServiceStatusRunning}},
			strategy:   deploy.StrategyCompose,
			from:       deploy.StatusRunning,
			event:      deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "failed", Services: []deploy.ObservedService{{Name: "api", ContainerName: "shop-api-1", Status: "running"}, {Name: "web", ContainerName: "shop-web-1", Status: "exited"}}},
			wantStatus: deploy.StatusFailed,
			want: map[string]badge{
				"shop/api": {status: topology.ServiceRunning},
				"shop/web": {status: topology.ServiceFailed},
			},
		},
		{
			name:       "a container missing from the report is shown stopped",
			before:     []deploy.Container{{ID: "c-api", Name: "api", Status: deploy.ServiceStatusRunning}, {ID: "c-web", Name: "web", Status: deploy.ServiceStatusRunning}},
			strategy:   deploy.StrategyCompose,
			from:       deploy.StatusRunning,
			event:      deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "healthy", Services: []deploy.ObservedService{{Name: "api", ContainerName: "shop-api-1", Status: "healthy"}}},
			wantStatus: deploy.StatusHealthy,
			want: map[string]badge{
				"shop/api": {status: topology.ServiceHealthy},
				"shop/web": {status: topology.ServiceStopped},
			},
		},
		{
			name:       "the compose placeholder a report proves a guess leaves the canvas and the real containers arrive",
			before:     []deploy.Container{{ID: "c-guess", Name: "shop", Status: deploy.ServiceStatusPending}},
			strategy:   deploy.StrategyCompose,
			from:       deploy.StatusRunning,
			event:      deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "healthy", Services: []deploy.ObservedService{{Name: "api", ContainerName: "shop-api-1", Status: "healthy", Networks: net("172.30.0.2")}, {Name: "web", ContainerName: "shop-web-1", Status: "running"}}},
			wantStatus: deploy.StatusHealthy,
			want: map[string]badge{
				"shop/api": {topology.ServiceHealthy, "172.30.0.2"},
				"shop/web": {status: topology.ServiceRunning},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := projectionFixture(t, tt.strategy)
			for _, c := range tt.before {
				addContainer(t, f, c.ID, c.Name, c.Status)
				_, err := f.svc.topoSvc.AddServiceNode(t.Context(), topology.Node{ID: c.ID, Data: topology.NodeData{Name: "shop", Runtime: "docker", Status: topology.ServiceRunning}})
				require.NoError(t, err)
			}
			addDeploy(t, f, "deploy-shop", tt.from)
			ev := event(t, deploy.TopicDeployStatusChanged, tt.event)
			handle := topologyDeployStatusHandler(f.svc.deploySvc, f.svc.topoSvc)

			require.NoError(t, handle(t.Context(), ev))
			d, err := f.svc.deploySvc.Get(t.Context(), "deploy-shop")
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, deploy.Status(d.Status))
			assert.Equal(t, tt.want, canvasBadges(t, f))

			require.NoError(t, handle(t.Context(), ev), "a redelivery changes nothing")
			assert.Equal(t, tt.want, canvasBadges(t, f))
		})
	}
}

func TestTopologyDeployStatusHandler_ErrorPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		deploy  deploy.Status
		payload func(t *testing.T) eventbus.Event
		wantErr error
	}{
		{"an unreadable payload is fatal", deploy.StatusRunning, func(*testing.T) eventbus.Event {
			return eventbus.Event{Topic: deploy.TopicDeployStatusChanged, Payload: []byte("{")}
		}, apperrs.ErrFatal},
		{"an unknown deploy is not found", deploy.StatusRunning, func(t *testing.T) eventbus.Event {
			return event(t, deploy.TopicDeployStatusChanged, deploy.DeployStatusChangedEvent{ID: "deploy-missing", Status: "healthy"})
		}, apperrs.ErrNotFound},
		{"a status a runner never sends is fatal", deploy.StatusRunning, func(t *testing.T) eventbus.Event {
			return event(t, deploy.TopicDeployStatusChanged, deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "running"})
		}, apperrs.ErrFatal},
		{"a deploy that already failed cannot turn healthy", deploy.StatusFailed, func(t *testing.T) eventbus.Event {
			return event(t, deploy.TopicDeployStatusChanged, deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "healthy"})
		}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := projectionFixture(t, deploy.StrategyRun)
			addContainer(t, f, "c-web", "web", deploy.ServiceStatusRunning)
			_, err := f.svc.topoSvc.AddServiceNode(t.Context(), topology.Node{ID: "c-web", Data: topology.NodeData{Name: "shop", Runtime: "docker", Status: topology.ServiceRunning}})
			require.NoError(t, err)
			addDeploy(t, f, "deploy-shop", tt.deploy)

			err = topologyDeployStatusHandler(f.svc.deploySvc, f.svc.topoSvc)(t.Context(), tt.payload(t))
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, topology.ServiceRunning, canvasNodes(t, f)["c-web"].Status, "a refused event leaves the canvas as it was")
		})
	}
}

// A sync the registry refuses after the deploy's rows landed is redelivered, and the retry finishes the projection.
func TestTopologyDeployStatusHandler_ARetryAfterAFailedSyncShowsTheReportedContainers(t *testing.T) {
	t.Parallel()
	f := projectionFixture(t, deploy.StrategyCompose)
	addContainer(t, f, "c-guess", "shop", deploy.ServiceStatusPending)
	_, err := f.svc.topoSvc.AddServiceNode(t.Context(), topology.Node{ID: "c-guess", Data: topology.NodeData{Name: "shop", Runtime: "docker", Status: topology.ServiceRunning}})
	require.NoError(t, err)
	addDeploy(t, f, "deploy-shop", deploy.StatusRunning)
	ev := event(t, deploy.TopicDeployStatusChanged, deploy.DeployStatusChangedEvent{ID: "deploy-shop", Status: "healthy", Services: []deploy.ObservedService{
		{Name: "web", ContainerName: "shop-web-1", Status: "running"},
	}})
	handle := topologyDeployStatusHandler(f.svc.deploySvc, f.svc.topoSvc)

	// A reader may read deploys but not write the registry, so the reconcile lands and the sync is refused.
	require.Error(t, handle(as(uReader), ev))
	services, err := f.svc.deploySvc.ListServices(t.Context(), "stack-shop")
	require.NoError(t, err)
	require.Len(t, services, 1, "the reconcile already replaced the placeholder with the real container")
	assert.Empty(t, canvasBadges(t, f), "the placeholder's row is gone and the real container has no node yet")

	require.NoError(t, handle(t.Context(), ev))
	assert.Equal(t, map[string]badge{"shop": {status: topology.ServiceRunning}}, canvasBadges(t, f), "the retry anchors the reported container")
}

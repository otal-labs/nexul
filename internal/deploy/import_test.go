package deploy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeMachineLookup is an in-memory deploy.MachineLookup for import tests.
type fakeMachineLookup struct {
	names map[string]string
}

func (f fakeMachineLookup) MachineName(_ context.Context, id string) (string, error) {
	name, ok := f.names[id]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return name, nil
}

// fakeMachineDiscoverer is an in-memory deploy.MachineDiscoverer for import tests.
type fakeMachineDiscoverer struct {
	containers []DiscoveredContainer
	err        error
}

func (f fakeMachineDiscoverer) DiscoverContainers(context.Context, string) ([]DiscoveredContainer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.containers, nil
}

func newImportTestService(discovered []DiscoveredContainer) (*Service, *fakeStackRepo, *fakeContainerRepo) {
	stacks := newFakeStackRepo()
	containers := newFakeContainerRepo()
	s := NewService(newFakeRepo(), stacks, containers, newFakeProjects())
	s.now = func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	s.SetMachineLookup(fakeMachineLookup{names: map[string]string{"m-1": "prod"}})
	s.SetMachineDiscoverer(fakeMachineDiscoverer{containers: discovered})
	return s, stacks, containers
}

func TestImport_CreatesOneStackPerComposeProjectWithObservedFacts(t *testing.T) {
	s, stacks, containers := newImportTestService([]DiscoveredContainer{
		{Name: "myapp-web-1", Image: "nginx:latest", Status: "running",
			Networks: []ImportNetwork{{Name: "myapp_default", Address: "172.20.0.2"}}, Ports: []string{"8080:80/tcp"}},
		{Name: "myapp-db-1", Image: "postgres:16", Status: "running"},
	})

	result, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID: "proj-1",
		Stacks:    []ImportStackGroup{{Project: "myapp", Containers: []string{"myapp-web-1", "myapp-db-1"}}},
	})
	require.NoError(t, err)
	require.Len(t, result.Stacks, 1)

	stack := result.Stacks[0]
	assert.Equal(t, "myapp", stack.Slug)
	assert.Equal(t, "prod", stack.Machine)
	assert.Equal(t, StrategyCompose, stack.Strategy)
	assert.False(t, stack.Managed, "an imported stack starts unmanaged (issue 08)")

	stored, err := stacks.GetBySlugAndMachine(context.Background(), "myapp", "prod")
	require.NoError(t, err)
	assert.Equal(t, stack.ID, stored.ID)

	svcs, err := containers.ListByStack(context.Background(), stack.ID)
	require.NoError(t, err)
	require.Len(t, svcs, 2)
	byName := map[string]*Container{}
	for _, c := range svcs {
		byName[c.Name] = c
	}
	web := byName["myapp-web-1"]
	require.NotNil(t, web)
	assert.Equal(t, "nginx:latest", web.Image)
	assert.Equal(t, ServiceStatusRunning, web.Status)
	require.Len(t, web.Networks, 1)
	assert.Equal(t, "172.20.0.2", web.Networks[0].Address)
	assert.Equal(t, []string{"8080:80/tcp"}, web.Ports)
}

func TestImport_StandaloneContainer_CreatesStackOfOne(t *testing.T) {
	s, stacks, containers := newImportTestService([]DiscoveredContainer{
		{Name: "redis-standalone", Image: "redis:7", Status: "running"},
	})

	result, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID:  "proj-1",
		Standalone: []string{"redis-standalone"},
	})
	require.NoError(t, err)
	require.Len(t, result.Stacks, 1)
	assert.Equal(t, "redis-standalone", result.Stacks[0].Slug)

	stored, err := stacks.GetBySlugAndMachine(context.Background(), "redis-standalone", "prod")
	require.NoError(t, err)
	svcs, err := containers.ListByStack(context.Background(), stored.ID)
	require.NoError(t, err)
	require.Len(t, svcs, 1)
	assert.Equal(t, "redis:7", svcs[0].Image)
}

// Gateways are only echoed back as "recognised, not adopted" (issue 08, ticket 14: dns owns gateway records).
// fakeGatewayAdopter records the adoption it was asked for and answers with a fixed outcome.
type fakeGatewayAdopter struct {
	in  *AdoptGatewayInput
	out *AdoptedGateway
	err error
}

func (f *fakeGatewayAdopter) AdoptTunnelGateway(_ context.Context, in AdoptGatewayInput) (*AdoptedGateway, error) {
	f.in = &in
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func TestImport_Gateway_AdoptedAsStackAndHandedToDNSWithTargets(t *testing.T) {
	s, stacks, containers := newImportTestService([]DiscoveredContainer{
		{Name: "hello-api", Image: "hello", Status: "running", Networks: []ImportNetwork{{Name: "app_default", Address: "10.0.0.2"}}},
		{Name: "cloudflared-local", Image: "cloudflare/cloudflared:latest", Status: "running", TunnelID: "tun-1",
			Networks: []ImportNetwork{{Name: "app_default"}, {Name: "other_default"}}},
	})
	adopter := &fakeGatewayAdopter{out: &AdoptedGateway{GatewayID: "gw-1", Exposed: []string{"api.example.com"}, Unmatched: []string{"web.example.com"}}}
	s.SetGatewayAdopter(adopter)

	result, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID:  "proj-1",
		Standalone: []string{"hello-api"},
		Gateways:   []string{"cloudflared-local"},
	})
	require.NoError(t, err)
	require.Len(t, result.Stacks, 2, "the gateway container is adopted as a stack-of-one so the canvas has a node for it")
	require.Len(t, result.Gateways, 1)
	assert.Equal(t, GatewayAdoption{Name: "cloudflared-local", GatewayID: "gw-1", Exposed: []string{"api.example.com"}, Unmatched: []string{"web.example.com"}}, result.Gateways[0])

	gwStack, err := stacks.GetBySlugAndMachine(context.Background(), "cloudflared-local", "prod")
	require.NoError(t, err)
	gwRows, err := containers.ListByStack(context.Background(), gwStack.ID)
	require.NoError(t, err)
	apiStack, err := stacks.GetBySlugAndMachine(context.Background(), "hello-api", "prod")
	require.NoError(t, err)
	apiRows, err := containers.ListByStack(context.Background(), apiStack.ID)
	require.NoError(t, err)

	require.NotNil(t, adopter.in)
	assert.Equal(t, "tun-1", adopter.in.TunnelID)
	assert.Equal(t, "prod", adopter.in.Machine)
	assert.Equal(t, gwRows[0].ID, adopter.in.ServiceID, "the gateway backs onto the adopted cloudflared row")
	assert.Equal(t, []string{"app_default", "other_default"}, adopter.in.Networks)
	assert.Equal(t, apiRows[0].ID, adopter.in.Targets["hello-api"], "routes resolve by container name to a services.id")
}

func TestImport_Gateway_FailuresLandOnTheRow(t *testing.T) {
	s, _, _ := newImportTestService([]DiscoveredContainer{
		{Name: "cf-no-token", Image: "cloudflare/cloudflared:latest", Status: "running"},
		{Name: "cf-broken", Image: "cloudflare/cloudflared:latest", Status: "running", TunnelID: "tun-2"},
	})
	s.SetGatewayAdopter(&fakeGatewayAdopter{err: errors.New("invalid: Cloudflare is not connected")})

	result, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID: "proj-1",
		Gateways:  []string{"cf-no-token", "cf-broken"},
	})
	require.NoError(t, err, "a gateway that cannot be adopted must not fail the import")
	require.Len(t, result.Gateways, 2)
	assert.Contains(t, result.Gateways[0].Error, "no tunnel token")
	assert.Contains(t, result.Gateways[1].Error, "Cloudflare is not connected")
	assert.Len(t, result.Stacks, 2, "the containers themselves are still adopted")
}

// Re-import matches the existing stack and container by name, updates observed facts, and never deletes.
func TestImport_ReImport_UpdatesObservedFactsWithoutDuplicating(t *testing.T) {
	s, stacks, containers := newImportTestService([]DiscoveredContainer{
		{Name: "myapp-web-1", Image: "nginx:1.25", Status: "running"},
	})
	first, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID: "proj-1",
		Stacks:    []ImportStackGroup{{Project: "myapp", Containers: []string{"myapp-web-1"}}},
	})
	require.NoError(t, err)
	firstStackID := first.Stacks[0].ID

	svcsBefore, err := containers.ListByStack(context.Background(), firstStackID)
	require.NoError(t, err)
	require.Len(t, svcsBefore, 1)
	firstContainerID := svcsBefore[0].ID

	// Second discovery run reports a new image tag on the same container name.
	s.discoverer = fakeMachineDiscoverer{containers: []DiscoveredContainer{
		{Name: "myapp-web-1", Image: "nginx:1.26", Status: "running"},
	}}
	second, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID: "proj-1",
		Stacks:    []ImportStackGroup{{Project: "myapp", Containers: []string{"myapp-web-1"}}},
	})
	require.NoError(t, err)
	require.Len(t, second.Stacks, 1)
	assert.Equal(t, firstStackID, second.Stacks[0].ID, "re-import must reuse the same stack, not duplicate it")

	all, err := stacks.ListByProject(context.Background(), "proj-1")
	require.NoError(t, err)
	assert.Len(t, all, 1, "re-import must never create a duplicate stack")

	svcsAfter, err := containers.ListByStack(context.Background(), firstStackID)
	require.NoError(t, err)
	require.Len(t, svcsAfter, 1, "re-import must never create a duplicate container")
	assert.Equal(t, firstContainerID, svcsAfter[0].ID)
	assert.Equal(t, "nginx:1.26", svcsAfter[0].Image)
}

func TestImport_MissingProjectID_ReturnsInvalid(t *testing.T) {
	s, _, _ := newImportTestService(nil)
	_, err := s.Import(context.Background(), "m-1", ImportRequest{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestImport_Unconfigured_ReturnsConflict(t *testing.T) {
	s := NewService(newFakeRepo(), newFakeStackRepo(), newFakeContainerRepo(), newFakeProjects())
	_, err := s.Import(context.Background(), "m-1", ImportRequest{ProjectID: "proj-1"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
}

func TestImport_UnknownMachine_Errors(t *testing.T) {
	s, _, _ := newImportTestService(nil)
	_, err := s.Import(context.Background(), "missing", ImportRequest{ProjectID: "proj-1"})
	require.Error(t, err)
}

// adoptedEvents returns the payloads of every service.created event the stack repo enqueued.
func adoptedEvents(t *testing.T, stacks *fakeStackRepo) []StackEvent {
	t.Helper()
	var out []StackEvent
	for _, evt := range stacks.outbox {
		if evt.Topic != TopicStackCreated {
			continue
		}
		payload, ok := evt.Payload.(StackEvent)
		require.True(t, ok, "service.created payload must be a StackEvent")
		out = append(out, payload)
	}
	return out
}

func TestImport_PublishesStackCreatedWithAdoptedRows(t *testing.T) {
	s, stacks, containers := newImportTestService([]DiscoveredContainer{
		{Name: "myapp-web-1", Image: "nginx:1.25", Status: "running", Networks: []ImportNetwork{{Name: "myapp_default", Address: "172.20.0.2"}}},
		{Name: "myapp-db-1", Image: "postgres:16", Status: "exited"},
	})
	res, err := s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID: "proj-1",
		Stacks:    []ImportStackGroup{{Project: "myapp", Containers: []string{"myapp-web-1", "myapp-db-1"}}},
	})
	require.NoError(t, err)

	events := adoptedEvents(t, stacks)
	require.Len(t, events, 1, "one service.created per adopted stack, published after the rows exist")
	assert.Equal(t, res.Stacks[0].ID, events[0].Stack.ID)
	require.Len(t, events[0].Services, 2, "the payload must carry the adopted containers, or the topology anchors nothing")

	rows, err := containers.ListByStack(context.Background(), res.Stacks[0].ID)
	require.NoError(t, err)
	byID := map[string]*Container{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	for _, svc := range events[0].Services {
		stored, ok := byID[svc.ID]
		require.True(t, ok, "payload ids must be the stored row ids")
		assert.Equal(t, stored.Status, svc.Status)
		assert.Equal(t, stored.Networks, svc.Networks)
	}
}

func TestImport_ReImport_PublishesOnlyNewlyAdoptedContainers(t *testing.T) {
	s, stacks, _ := newImportTestService([]DiscoveredContainer{
		{Name: "myapp-web-1", Image: "nginx:1.25", Status: "running"},
	})
	req := ImportRequest{
		ProjectID: "proj-1",
		Stacks:    []ImportStackGroup{{Project: "myapp", Containers: []string{"myapp-web-1"}}},
	}
	_, err := s.Import(context.Background(), "m-1", req)
	require.NoError(t, err)

	// Same selection again: nothing new was adopted, so no event.
	_, err = s.Import(context.Background(), "m-1", req)
	require.NoError(t, err)
	require.Len(t, adoptedEvents(t, stacks), 1)

	// A container that joined the compose project since: only it rides the event.
	s.discoverer = fakeMachineDiscoverer{containers: []DiscoveredContainer{
		{Name: "myapp-web-1", Image: "nginx:1.25", Status: "running"},
		{Name: "myapp-worker-1", Image: "myapp/worker", Status: "running"},
	}}
	_, err = s.Import(context.Background(), "m-1", ImportRequest{
		ProjectID: "proj-1",
		Stacks:    []ImportStackGroup{{Project: "myapp", Containers: []string{"myapp-web-1", "myapp-worker-1"}}},
	})
	require.NoError(t, err)
	events := adoptedEvents(t, stacks)
	require.Len(t, events, 2)
	require.Len(t, events[1].Services, 1)
	assert.Equal(t, "myapp-worker-1", events[1].Services[0].Name)
}

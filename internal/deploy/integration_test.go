package deploy_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// openStore returns a migrated real-SQLite store on a temp file.
func openStore(t *testing.T) *storage.Store {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
}

func newService(t *testing.T, store *storage.Store) *deploy.Service {
	t.Helper()
	return deploy.NewService(store.Deploys, store.Stacks, store.Services, &testProjects{store: store})
}

// outboxTopics returns the topics of every unpublished outbox row, so tests
// can assert an event reached the transactional outbox without
// waiting on the relay.
func outboxTopics(t *testing.T, store *storage.Store) map[string]bool {
	t.Helper()
	pending, err := store.Outbox.Unpublished(context.Background(), 100)
	require.NoError(t, err)
	topics := make(map[string]bool, len(pending))
	for _, e := range pending {
		topics[e.Topic] = true
	}
	return topics
}

// testProjects adapts the storage projects repo to the deploy ProjectStore
// seam for the integration test.
type testProjects struct {
	store *storage.Store
}

func (p *testProjects) ProjectExists(ctx context.Context, projectID string) (bool, error) {
	_, err := p.store.Projects.Get(ctx, projectID)
	return err == nil, nil
}

func (p *testProjects) LinkRepo(_ context.Context, _ string, _ string, _ string) error { return nil }

func (p *testProjects) RepoInProject(_ context.Context, _ string, _ string, _ string) (bool, error) {
	return true, nil
}

func seedStack(t *testing.T, store *storage.Store) *deploy.Stack {
	t.Helper()
	stack := &deploy.Stack{
		ID:          "svc-1",
		ProjectID:   "project-general",
		Name:        "api",
		Slug:        "api",
		Machine:     "10.0.0.1:22",
		Strategy:    deploy.StrategyCompose,
		ComposePath: "/srv/api/docker-compose.yml",
		Managed:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	require.NoError(t, store.Stacks.Create(context.Background(), stack))
	return stack
}

// TestIntegration_RunnerOnlyFlow covers the runner-only wiring: Deploy validates,
// persists a pending record, and enqueues deploy.requested; the terminal state
// lands exclusively via deploy.status_changed, all over real SQLite.
func TestIntegration_RunnerOnlyFlow(t *testing.T) {
	store := openStore(t)
	svc := newService(t, store)
	seedStack(t, store)

	got, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "ghcr.io/onik/api:v1"})
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusPending, got.Status)
	assert.Equal(t, "svc-1", got.StackID)
	assert.True(t, outboxTopics(t, store)[deploy.TopicDeployRequested], "deploy.requested written via the outbox")

	stored, err := store.Deploys.GetByID(context.Background(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusPending, stored.Status)

	require.NoError(t, svc.HandleStatusChanged(context.Background(), eventbus.Event{
		Payload: json.RawMessage(`{"id":"` + got.ID + `","status":"healthy"}`),
	}))

	stored, err = store.Deploys.GetByID(context.Background(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusHealthy, stored.Status)
}

// TestIntegration_OneActivePerStack proves the one-active-deploy guard over real SQLite:
// a second deploy while the first is still active is rejected, and the partial
// unique index is the backstop.
func TestIntegration_OneActivePerStack(t *testing.T) {
	store := openStore(t)
	svc := newService(t, store)
	seedStack(t, store)

	first, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "img:v1"})
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusPending, first.Status)

	_, err = svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "img:v2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active deploy")

	require.NoError(t, svc.HandleStatusChanged(context.Background(), eventbus.Event{
		Payload: json.RawMessage(`{"id":"` + first.ID + `","status":"healthy"}`),
	}))

	second, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "img:v2"})
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusPending, second.Status)
}

// TestIntegration_RunnerFailurePath lands a runner-reported failure: the streamed output stays in time order
// and the error detail lands as the last log line.
func TestIntegration_RunnerFailurePath(t *testing.T) {
	store := openStore(t)
	svc := newService(t, store)
	seedStack(t, store)

	got, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "ghcr.io/onik/api:v1"})
	require.NoError(t, err)

	require.NoError(t, svc.HandleLog(context.Background(), eventbus.Event{
		Payload: json.RawMessage(`{"id":"` + got.ID + `","phase":"deploy","log":"docker pull ghcr.io/onik/api:v1\ncontainer exited 1\n","ts":1695379028112}`),
	}))
	require.NoError(t, svc.HandleStatusChanged(context.Background(), eventbus.Event{
		Payload: json.RawMessage(`{"id":"` + got.ID + `","status":"failed","error":"container exited 1"}`),
	}))

	stored, err := store.Deploys.GetByID(context.Background(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, deploy.StatusFailed, stored.Status)
	lines, err := svc.Log(context.Background(), got.ID)
	require.NoError(t, err)
	require.Len(t, lines, 3)
	assert.Equal(t, "docker pull ghcr.io/onik/api:v1", lines[0].Text)
	assert.Equal(t, "deploy", lines[0].Phase)
	assert.Equal(t, "container exited 1", lines[1].Text)
	assert.Equal(t, "deploy failed: container exited 1", lines[2].Text)
	assert.Equal(t, "", lines[2].Phase)
	assert.True(t, outboxTopics(t, store)[deploy.TopicDeployUpdated], "deploy.updated written via the outbox")
}

// TestIntegration_RollbackAndStackEvents covers the one-click rollback
// and the stack lifecycle events written through the outbox.
func TestIntegration_RollbackAndStackEvents(t *testing.T) {
	store := openStore(t)
	svc := newService(t, store)
	seedStack(t, store)

	first, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "img:v1", TriggeredBy: "user-1"})
	require.NoError(t, err)
	require.NoError(t, svc.HandleStatusChanged(context.Background(), eventbus.Event{
		Payload: json.RawMessage(`{"id":"` + first.ID + `","status":"healthy"}`),
	}))
	second, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "img:v2"})
	require.NoError(t, err)
	require.NoError(t, svc.HandleStatusChanged(context.Background(), eventbus.Event{
		Payload: json.RawMessage(`{"id":"` + second.ID + `","status":"failed","error":"bad image"}`),
	}))

	rolled, err := svc.Rollback(context.Background(), "svc-1", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "img:v1", rolled.Image, "rollback uses the last healthy image, not the failed one")

	created, err := svc.CreateStack(context.Background(), deploy.Stack{
		ProjectID:     "project-general",
		Name:          "web",
		Machine:       "10.0.0.1:22",
		Strategy:      deploy.StrategyRun,
		DockerNetwork: "net1",
	}, nil)
	require.NoError(t, err)

	created2Svcs, err := svc.ListServices(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, created2Svcs, 1, "a run stack gets exactly one pending service named after its slug")
	assert.Equal(t, created.Slug, created2Svcs[0].Name)

	require.NoError(t, svc.DeleteStack(context.Background(), created.ID))

	topics := outboxTopics(t, store)
	assert.True(t, topics[deploy.TopicDeployRequested], "deploy.requested written via the outbox")
	assert.True(t, topics[deploy.TopicStackCreated], "stack.created (service.created) written via the outbox")
	assert.True(t, topics[deploy.TopicStackDeleted], "stack.deleted (service.deleted) written via the outbox")
}

// TestIntegration_DeployRequestedEnvNeverOnTheWire is ticket 14's end-to-end
// guard: a secret in a stack's Env must not appear in the outbox row's raw
// bytes — the exact payload every bus subscriber (automation, webhooks-out,
// dead-letter storage) would see.
func TestIntegration_DeployRequestedEnvNeverOnTheWire(t *testing.T) {
	store := openStore(t)
	svc := newService(t, store)
	def := seedStack(t, store)
	def.Env = map[string]string{"API_TOKEN": "sup3rSecretValue"}
	require.NoError(t, store.Stacks.Update(context.Background(), def))

	_, err := svc.Deploy(context.Background(), deploy.DeployRequest{StackID: "svc-1", Image: "ghcr.io/onik/api:v1"})
	require.NoError(t, err)

	pending, err := store.Outbox.Unpublished(context.Background(), 100)
	require.NoError(t, err)
	var raw []byte
	for _, e := range pending {
		if e.Topic == deploy.TopicDeployRequested {
			raw = e.Payload
		}
	}
	require.NotEmpty(t, raw, "deploy.requested must be in the outbox")
	assert.NotContains(t, string(raw), "sup3rSecretValue", "the raw outbox row is what every subscriber sees; it must never carry the secret")
	assert.Contains(t, string(raw), "API_TOKEN", "the key still travels")
}

// TestIntegration_StatusChanged_FansOutToAllConsumers is the ws-33 production
// wiring regression: with the real bus + dedupe (the production configuration),
// one deploy.status_changed event must reach all four consumers — the deploy
// domain, live push, automations, and integrations fan-out — not just whichever
// one wins the dedupe race.
func TestIntegration_StatusChanged_FansOutToAllConsumers(t *testing.T) {
	store := openStore(t)
	svc := newService(t, store)
	seedStack(t, store)
	ctx := context.Background()

	got, err := svc.Deploy(ctx, deploy.DeployRequest{StackID: "svc-1", Image: "ghcr.io/onik/api:v1"})
	require.NoError(t, err)

	bus := inprocess.New(inprocess.Options{
		Logger:      testutil.DiscardLogger(),
		DedupeStore: store.ProcessedEvents,
	})
	defer func() { require.NoError(t, bus.Close()) }()

	var live, automations, integrations atomic.Int32
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "deploy.live_status", deploy.TopicDeployStatusChanged, func(ctx context.Context, ev eventbus.Event) error {
		return svc.HandleStatusChanged(ctx, ev)
	}))
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "live.push", deploy.TopicDeployStatusChanged, func(ctx context.Context, ev eventbus.Event) error {
		live.Add(1)
		return nil
	}))
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "automations", deploy.TopicDeployStatusChanged, func(ctx context.Context, ev eventbus.Event) error {
		automations.Add(1)
		return nil
	}))
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "integrations.fanout", deploy.TopicDeployStatusChanged, func(ctx context.Context, ev eventbus.Event) error {
		integrations.Add(1)
		return nil
	}))

	require.NoError(t, bus.Publish(ctx, deploy.TopicDeployStatusChanged, deploy.DeployStatusChangedEvent{
		ID: got.ID, Status: string(deploy.StatusHealthy),
	}))

	require.Eventually(t, func() bool {
		if live.Load() != 1 || automations.Load() != 1 || integrations.Load() != 1 {
			return false
		}
		stored, err := store.Deploys.GetByID(ctx, got.ID)
		return err == nil && stored.Status == deploy.StatusHealthy
	}, 2*time.Second, 10*time.Millisecond, "all four consumers must process the event")
}

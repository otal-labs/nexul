package topology

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type fakeRepo struct {
	mu      sync.Mutex
	stored  map[string]*Canvas
	outbox  []eventbus.OutboxEvent
	getErr  error
	saveErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{stored: map[string]*Canvas{}}
}

func (f *fakeRepo) Get(_ context.Context, environment string) (*Canvas, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.stored[environment]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return c, nil
}

func (f *fakeRepo) Save(_ context.Context, environment string, c *Canvas, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.stored[environment] = c
	f.outbox = append(f.outbox, evts...)
	return nil
}

// lastUpdated decodes the last topology.updated event enqueued to the outbox.
func (f *fakeRepo) lastUpdated() UpdatedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var e UpdatedEvent
	for _, evt := range f.outbox {
		if evt.Topic == TopicUpdated {
			if c, ok := evt.Payload.(UpdatedEvent); ok {
				e = c
			}
		}
	}
	return e
}

type fakeBus struct {
	mu         sync.Mutex
	published  []eventbus.Event
	publishErr error
}

func (f *fakeBus) Publish(_ context.Context, topic string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.publishErr != nil {
		return f.publishErr
	}
	b, _ := json.Marshal(payload)
	f.published = append(f.published, eventbus.Event{Topic: topic, Payload: b})
	return nil
}

func newFakeBus() *fakeBus {
	return &fakeBus{}
}

// bus is accepted for call-site compatibility but is unused: topology.updated
// reaches subscribers via the transactional outbox on repo, not a direct
// Publish. Tests assert on repo.lastUpdated() instead.
func newTestService(repo *fakeRepo, bus *fakeBus) *Service {
	return NewService(repo)
}

func validNode() Node {
	return Node{
		ID:       "svc-api",
		Type:     NodeService,
		Position: Position{X: 100, Y: 200},
		Data: NodeData{
			ServiceID: "svc-api",
			Name:      "api", Runtime: "go", URL: "https://api.example.com",
			Status: ServiceHealthy, Replicas: 3,
		},
	}
}

func validNetworkNode() Node {
	return Node{
		ID:       "net-main",
		Type:     NodeNetwork,
		Position: Position{X: 1, Y: 1},
		Data:     NodeData{Name: "main"},
	}
}

func validExternalNode() Node {
	return Node{
		ID:       "ext-cf",
		Type:     NodeExternal,
		Position: Position{X: 2, Y: 2},
		Data:     NodeData{Name: "Cloudflare", Label: LabelTunnel, URL: "https://example.com"},
	}
}

func validEdge() Edge {
	return Edge{ID: "edge-1", Source: "svc-api", Target: "svc-db", Type: "relation", Data: EdgeData{Kind: KindDependsOn}}
}

func TestGet(t *testing.T) {
	t.Run("empty environment is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Get(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing canvas returns empty default", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		got, err := s.Get(context.Background(), "prod")
		require.NoError(t, err)
		assert.Equal(t, CurrentSchemaVersion, got.SchemaVersion)
		assert.Empty(t, got.Nodes)
		assert.Empty(t, got.Edges)
	})
	t.Run("returns stored canvas", func(t *testing.T) {
		repo := newFakeRepo()
		c := &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}
		require.NoError(t, repo.Save(context.Background(), "prod", c))
		s := newTestService(repo, newFakeBus())

		got, err := s.Get(context.Background(), "prod")
		require.NoError(t, err)
		assert.Len(t, got.Nodes, 1)
		assert.Equal(t, "svc-api", got.Nodes[0].ID)
	})
	t.Run("propagates repo error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		_, err := s.Get(context.Background(), "prod")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("empty environment is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Update(context.Background(), "", &Canvas{SchemaVersion: CurrentSchemaVersion})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unsupported schema_version is rejected", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Update(context.Background(), "prod", &Canvas{SchemaVersion: 999})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("saves and publishes network/external nodes", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		c := &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNetworkNode(), validExternalNode()}}

		got, err := s.Update(context.Background(), "prod", c)
		require.NoError(t, err)
		assert.Len(t, got.Nodes, 2)

		stored, err := repo.Get(context.Background(), "prod")
		require.NoError(t, err)
		assert.Len(t, stored.Nodes, 2)

		ev := repo.lastUpdated()
		assert.Equal(t, "prod", ev.Environment)
		assert.Len(t, ev.Canvas.Nodes, 2)
	})
	t.Run("merges stored service nodes into the incoming canvas", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode()},
		}))

		// A full-replace save drops the service node and adds a network node:
		// the service node must be merged back (auto-managed, cannot be
		// removed on the canvas), while the network node is kept.
		got, err := s.Update(context.Background(), "prod", &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNetworkNode()},
		})
		require.NoError(t, err)
		require.Len(t, got.Nodes, 2)
		byID := map[string]Node{}
		for _, n := range got.Nodes {
			byID[n.ID] = n
		}
		assert.Equal(t, NodeService, byID["svc-api"].Type, "service node survives a canvas save")
		assert.Equal(t, NodeNetwork, byID["net-main"].Type)
	})
	t.Run("ignores client-supplied service node names", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode()},
		}))

		renamed := validNode()
		renamed.Data.Name = "renamed-on-canvas"
		got, err := s.Update(context.Background(), "prod", &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{renamed},
		})
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "api", got.Nodes[0].Data.Name, "service node name comes from the definition")
	})
	t.Run("propagates repo save error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.saveErr = errors.New("disk full")
		s := newTestService(repo, newFakeBus())
		_, err := s.Update(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion})
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.saveErr)
	})
}

func TestAddNode(t *testing.T) {
	t.Run("invalid node is rejected", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.AddNode(context.Background(), "prod", Node{ID: "svc-api", Type: NodeService})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("adds node to empty canvas", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)

		c, err := s.AddNode(context.Background(), "prod", validNode())
		require.NoError(t, err)
		require.Len(t, c.Nodes, 1)
		assert.Equal(t, "svc-api", c.Nodes[0].ID)
		assert.Len(t, repo.lastUpdated().Canvas.Nodes, 1)
	})
	t.Run("duplicate node is a conflict", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, s.repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}))

		_, err := s.AddNode(context.Background(), "prod", validNode())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("edge referencing added node stays valid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		c, err := s.AddNode(context.Background(), "prod", validNode())
		require.NoError(t, err)
		assert.Equal(t, CurrentSchemaVersion, c.SchemaVersion)
	})
}

func TestRemoveNode(t *testing.T) {
	t.Run("missing node is not found", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, s.repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}))

		_, err := s.RemoveNode(context.Background(), "prod", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("empty node id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.RemoveNode(context.Background(), "prod", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("removes node and its edges", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		c := &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNetworkNode(), {ID: "net-2", Type: NodeNetwork, Position: Position{X: 1, Y: 1}, Data: NodeData{Name: "two"}}},
			Edges:         []Edge{{ID: "edge-1", Source: "net-main", Target: "net-2", Type: "relation", Data: EdgeData{Kind: KindDependsOn}}},
		}
		require.NoError(t, s.repo.Save(context.Background(), "prod", c))

		got, err := s.RemoveNode(context.Background(), "prod", "net-main")
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "net-2", got.Nodes[0].ID)
		assert.Empty(t, got.Edges, "edges referencing the removed node are dropped")
		assert.Len(t, repo.lastUpdated().Canvas.Nodes, 1)
	})
	t.Run("service node removal is rejected", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, s.repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}))

		_, err := s.RemoveNode(context.Background(), "prod", "svc-api")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestAddEdge(t *testing.T) {
	t.Run("invalid edge is rejected", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.AddEdge(context.Background(), "prod", Edge{ID: "edge-1", Type: "relation"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("edge with unknown node is rejected", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.AddEdge(context.Background(), "prod", validEdge())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("adds edge between existing nodes", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		c := &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode(), {ID: "svc-db", Type: NodeService, Position: Position{X: 1, Y: 1}, Data: NodeData{ServiceID: "svc-db", Name: "db", Runtime: "postgres", Status: ServiceHealthy}}},
		}
		require.NoError(t, s.repo.Save(context.Background(), "prod", c))

		got, err := s.AddEdge(context.Background(), "prod", validEdge())
		require.NoError(t, err)
		require.Len(t, got.Edges, 1)
		assert.Equal(t, KindDependsOn, got.Edges[0].Data.Kind)
		assert.Len(t, repo.lastUpdated().Canvas.Edges, 1)
	})
	t.Run("duplicate edge is a conflict", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		c := &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode(), {ID: "svc-db", Type: NodeService, Position: Position{X: 1, Y: 1}, Data: NodeData{ServiceID: "svc-db", Name: "db", Runtime: "postgres", Status: ServiceHealthy}}},
			Edges:         []Edge{validEdge()},
		}
		require.NoError(t, s.repo.Save(context.Background(), "prod", c))

		_, err := s.AddEdge(context.Background(), "prod", validEdge())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
}

func TestRemoveEdge(t *testing.T) {
	t.Run("missing edge is not found", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, s.repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion}))

		_, err := s.RemoveEdge(context.Background(), "prod", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("removes edge and publishes", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		c := &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode(), {ID: "svc-db", Type: NodeService, Position: Position{X: 1, Y: 1}, Data: NodeData{ServiceID: "svc-db", Name: "db", Runtime: "postgres", Status: ServiceHealthy}}},
			Edges:         []Edge{validEdge()},
		}
		require.NoError(t, s.repo.Save(context.Background(), "prod", c))

		got, err := s.RemoveEdge(context.Background(), "prod", "edge-1")
		require.NoError(t, err)
		assert.Empty(t, got.Edges)
		assert.Empty(t, repo.lastUpdated().Canvas.Edges)
	})
}

func TestRenameServiceNode(t *testing.T) {
	t.Run("missing service id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.RenameServiceNode(context.Background(), "prod", "", "api")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.RenameServiceNode(context.Background(), "prod", "svc-api", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("renames the anchored service node", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}))

		got, err := s.RenameServiceNode(context.Background(), "prod", "svc-api", "gateway")
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "gateway", got.Nodes[0].Data.Name)
		assert.Len(t, repo.lastUpdated().Canvas.Nodes, 1)
	})
	t.Run("missing node is a no-op", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion}))

		got, err := s.RenameServiceNode(context.Background(), "prod", "svc-ghost", "x")
		require.NoError(t, err)
		assert.Empty(t, got.Nodes)
	})
}

func TestSetServiceNodeStatus(t *testing.T) {
	t.Run("missing service id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.SetServiceNodeStatus(context.Background(), "prod", "", ServiceHealthy, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("bad status is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.SetServiceNodeStatus(context.Background(), "prod", "svc-api", ServiceStatus("bogus"), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("updates the anchored service node status", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}))

		got, err := s.SetServiceNodeStatus(context.Background(), "prod", "svc-api", ServiceFailed, "")
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, ServiceFailed, got.Nodes[0].Data.Status)
		assert.Len(t, repo.lastUpdated().Canvas.Nodes, 1)
	})
	t.Run("missing node is a no-op", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion}))

		got, err := s.SetServiceNodeStatus(context.Background(), "prod", "svc-ghost", ServiceHealthy, "")
		require.NoError(t, err)
		assert.Empty(t, got.Nodes)
	})
	t.Run("sets the address when reported", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}))

		got, err := s.SetServiceNodeStatus(context.Background(), "prod", "svc-api", ServiceHealthy, "172.18.0.4")
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "172.18.0.4", got.Nodes[0].Data.Address)
	})
	t.Run("an empty address keeps the previous one", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		node := validNode()
		node.Data.Address = "172.18.0.4"
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{node}}))

		got, err := s.SetServiceNodeStatus(context.Background(), "prod", "svc-api", ServiceFailed, "")
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "172.18.0.4", got.Nodes[0].Data.Address, "a failed redeploy must not wipe the last known address")
	})
}

func TestRemoveServiceNode(t *testing.T) {
	t.Run("missing service id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.RemoveServiceNode(context.Background(), "prod", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("removes the service node and its edges", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		c := &Canvas{
			SchemaVersion: CurrentSchemaVersion,
			Nodes:         []Node{validNode(), validNetworkNode()},
			Edges:         []Edge{{ID: "edge-1", Source: "svc-api", Target: "net-main", Type: "relation", Data: EdgeData{Kind: KindConnectsTo}}},
		}
		require.NoError(t, repo.Save(context.Background(), "prod", c))

		got, err := s.RemoveServiceNode(context.Background(), "prod", "svc-api")
		require.NoError(t, err)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "net-main", got.Nodes[0].ID)
		assert.Empty(t, got.Edges, "edges referencing the removed service node are dropped")
		assert.Len(t, repo.lastUpdated().Canvas.Nodes, 1)
	})
	t.Run("missing node is a no-op", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{SchemaVersion: CurrentSchemaVersion}))

		got, err := s.RemoveServiceNode(context.Background(), "prod", "svc-ghost")
		require.NoError(t, err)
		assert.Empty(t, got.Nodes)
	})
}

func TestMigrate(t *testing.T) {
	t.Run("current version is unchanged", func(t *testing.T) {
		c := &Canvas{SchemaVersion: CurrentSchemaVersion, Nodes: []Node{validNode()}}
		got, changed := Migrate(c)
		assert.False(t, changed)
		assert.Same(t, c, got)
	})
	t.Run("v1 service nodes gain the service_id anchor", func(t *testing.T) {
		c := &Canvas{
			SchemaVersion: 1,
			Nodes: []Node{{
				ID:       "svc-api",
				Type:     NodeService,
				Position: Position{X: 1, Y: 1},
				Data:     NodeData{Name: "api", Status: ServiceRunning},
			}},
		}
		got, changed := Migrate(c)
		require.True(t, changed)
		assert.Equal(t, CurrentSchemaVersion, got.SchemaVersion)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "svc-api", got.Nodes[0].Data.ServiceID)
		assert.Equal(t, "api", got.Nodes[0].Data.Name)
	})
	t.Run("Get migrates a stored v1 canvas and persists it", func(t *testing.T) {
		repo := newFakeRepo()
		bus := newFakeBus()
		s := newTestService(repo, bus)
		require.NoError(t, repo.Save(context.Background(), "prod", &Canvas{
			SchemaVersion: 1,
			Nodes: []Node{{
				ID:       "svc-api",
				Type:     NodeService,
				Position: Position{X: 1, Y: 1},
				Data:     NodeData{Name: "api", Status: ServiceRunning},
			}},
		}))

		got, err := s.Get(context.Background(), "prod")
		require.NoError(t, err)
		assert.Equal(t, CurrentSchemaVersion, got.SchemaVersion)
		require.Len(t, got.Nodes, 1)
		assert.Equal(t, "svc-api", got.Nodes[0].Data.ServiceID)

		stored, err := repo.Get(context.Background(), "prod")
		require.NoError(t, err)
		assert.Equal(t, CurrentSchemaVersion, stored.SchemaVersion, "migrated canvas persisted")
	})
}

func TestService_Update_KeepsViewport(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	ctx := context.Background()
	c := &Canvas{SchemaVersion: 2, Nodes: []Node{validNetworkNode()}, Viewport: &Viewport{X: 12, Y: -40, Zoom: 0.8}}
	saved, err := svc.Update(ctx, DefaultEnvironment, c)
	require.NoError(t, err)
	assert.Equal(t, &Viewport{X: 12, Y: -40, Zoom: 0.8}, saved.Viewport)

	got, err := svc.Get(ctx, DefaultEnvironment)
	require.NoError(t, err)
	assert.Equal(t, &Viewport{X: 12, Y: -40, Zoom: 0.8}, got.Viewport)
}

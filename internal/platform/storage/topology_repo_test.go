package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/topology"
)

func newTestCanvas() *topology.Canvas {
	return &topology.Canvas{
		SchemaVersion: 2,
		Nodes: []topology.Node{
			{
				ID:       "svc-api",
				Type:     topology.NodeService,
				Position: topology.Position{X: 100, Y: 200},
				Data: topology.NodeData{ServiceID: "svc-api",
					Name: "api", Runtime: "go", URL: "https://api.example.com",
					Status: topology.ServiceHealthy, Replicas: 3,
				},
			},
		},
		Edges: []topology.Edge{
			{ID: "edge-1", Source: "svc-api", Target: "svc-db", Type: "relation", Data: topology.EdgeData{Kind: topology.KindDependsOn}},
		},
	}
}

func TestTopologyRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Topology.Get(context.Background(), "prod")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTopologyRepo_Get_CorruptJSON_Error(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.db.ExecContext(context.Background(),
		`INSERT INTO topology (environment, schema_version, canvas, updated_at) VALUES ('prod', 1, 'not json', 1)`)
	require.NoError(t, err)

	_, err = s.Topology.Get(context.Background(), "prod")
	require.Error(t, err)
}

func TestTopologyRepo_Save_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestCanvas()
	require.NoError(t, s.Topology.Save(context.Background(), "prod", want))

	got, err := s.Topology.Get(context.Background(), "prod")
	require.NoError(t, err)
	assert.Equal(t, 2, got.SchemaVersion)
	require.Len(t, got.Nodes, 1)
	assert.Equal(t, "svc-api", got.Nodes[0].ID)
	assert.Equal(t, float64(100), got.Nodes[0].Position.X)
	assert.Equal(t, topology.ServiceHealthy, got.Nodes[0].Data.Status)
	require.Len(t, got.Edges, 1)
	assert.Equal(t, topology.KindDependsOn, got.Edges[0].Data.Kind)
}

func TestTopologyRepo_Save_UpsertsEnvironment(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Topology.Save(context.Background(), "prod", newTestCanvas()))

	updated := newTestCanvas()
	updated.Nodes[0].Data.Replicas = 5
	require.NoError(t, s.Topology.Save(context.Background(), "prod", updated))

	got, err := s.Topology.Get(context.Background(), "prod")
	require.NoError(t, err)
	assert.Equal(t, 5, got.Nodes[0].Data.Replicas)
	assert.Len(t, got.Nodes, 1, "upsert replaces, not appends")
}

func TestTopologyRepo_Save_EmptyCanvas(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	empty := &topology.Canvas{SchemaVersion: 1}
	require.NoError(t, s.Topology.Save(context.Background(), "prod", empty))

	got, err := s.Topology.Get(context.Background(), "prod")
	require.NoError(t, err)
	assert.Empty(t, got.Nodes)
	assert.Empty(t, got.Edges)
}

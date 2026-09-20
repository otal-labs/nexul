package topology

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo(), newFakeBus()))
	require.Len(t, tools, 5)
	names := []string{tools[0].Name, tools[1].Name, tools[2].Name, tools[3].Name, tools[4].Name}
	assert.ElementsMatch(t, []string{"topology_get", "topology_add_node", "topology_remove_node", "topology_add_edge", "topology_remove_edge"}, names)
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
}

func TestMCPTools_Get(t *testing.T) {
	t.Run("missing environment defaults to canonical", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		_, err := svc.AddNode(context.Background(), DefaultEnvironment, validNode())
		require.NoError(t, err)
		call := toolByName(t, MCPTools(svc), "topology_get").Call
		got, err := call(context.Background(), map[string]any{})
		require.NoError(t, err)
		c, ok := got.(*Canvas)
		require.True(t, ok)
		assert.Equal(t, "svc-api", c.Nodes[0].ID)
	})
	t.Run("unsaved environment returns empty canvas", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "topology_get").Call
		got, err := call(context.Background(), map[string]any{"environment": "prod"})
		require.NoError(t, err)
		c, ok := got.(*Canvas)
		require.True(t, ok)
		assert.Equal(t, CurrentSchemaVersion, c.SchemaVersion)
		assert.Empty(t, c.Nodes)
	})
	t.Run("repo failure surfaces", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("boom")
		call := toolByName(t, MCPTools(newTestService(repo, newFakeBus())), "topology_get").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

func TestMCPTools_AddNode(t *testing.T) {
	call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "topology_add_node").Call

	t.Run("network node happy path", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{
			"type": "network", "id": "net-1", "name": "main",
		})
		require.NoError(t, err)
		c := got.(*Canvas)
		assert.Equal(t, NodeNetwork, c.Nodes[0].Type)
		assert.Equal(t, "main", c.Nodes[0].Data.Name)
	})
	t.Run("external node happy path", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		call := toolByName(t, MCPTools(svc), "topology_add_node").Call
		got, err := call(context.Background(), map[string]any{
			"type": "external", "id": "ext-1", "name": "Cloudflare", "label": "tunnel", "url": "https://example.com",
		})
		require.NoError(t, err)
		c := got.(*Canvas)
		require.Len(t, c.Nodes, 1)
		assert.Equal(t, NodeExternal, c.Nodes[0].Type)
		assert.Equal(t, ExternalLabel("tunnel"), c.Nodes[0].Data.Label)
	})
	t.Run("service type is rejected as auto-managed", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"type": "service", "id": "svc-1", "name": "api"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing type is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"name": "api"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing label for external is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"type": "external", "id": "ext-1", "name": "Cloudflare"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("duplicate id is conflict", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		_, err := svc.AddNode(context.Background(), DefaultEnvironment, validNetworkNode())
		require.NoError(t, err)
		call := toolByName(t, MCPTools(svc), "topology_add_node").Call
		_, err = call(context.Background(), map[string]any{"type": "network", "id": "net-main", "name": "dup"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
}

func TestMCPTools_RemoveNode(t *testing.T) {
	t.Run("removes node and drops edges", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		_, err := svc.AddNode(context.Background(), DefaultEnvironment, validNetworkNode())
		require.NoError(t, err)
		call := toolByName(t, MCPTools(svc), "topology_remove_node").Call
		_, err = call(context.Background(), map[string]any{"node_id": "net-main"})
		require.NoError(t, err)
		c, _ := svc.Get(context.Background(), DefaultEnvironment)
		assert.Empty(t, c.Nodes)
	})
	t.Run("service node removal is rejected", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		_, err := svc.AddNode(context.Background(), DefaultEnvironment, validNode())
		require.NoError(t, err)
		call := toolByName(t, MCPTools(svc), "topology_remove_node").Call
		_, err = call(context.Background(), map[string]any{"node_id": "svc-api"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing node id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "topology_remove_node").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown node is not found", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "topology_remove_node").Call
		_, err := call(context.Background(), map[string]any{"node_id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_AddEdge(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		_, err := svc.AddNode(context.Background(), DefaultEnvironment, validNetworkNode())
		require.NoError(t, err)
		other := validNetworkNode()
		other.ID = "net-2"
		other.Data.Name = "two"
		_, err = svc.AddNode(context.Background(), DefaultEnvironment, other)
		require.NoError(t, err)
		call := toolByName(t, MCPTools(svc), "topology_add_edge").Call
		_, err = call(context.Background(), map[string]any{"id": "e1", "source": "net-main", "target": "net-2", "kind": "depends_on"})
		require.NoError(t, err)
		c, _ := svc.Get(context.Background(), DefaultEnvironment)
		assert.Equal(t, "net-main", c.Edges[0].Source)
	})
	t.Run("missing kind is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "topology_add_edge").Call
		_, err := call(context.Background(), map[string]any{"id": "e1", "source": "a", "target": "b"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_RemoveEdge(t *testing.T) {
	t.Run("removes edge", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newTestService(repo, newFakeBus())
		_, err := svc.AddNode(context.Background(), DefaultEnvironment, validNode())
		require.NoError(t, err)
		other := validNode()
		other.ID = "db"
		_, err = svc.AddNode(context.Background(), DefaultEnvironment, other)
		require.NoError(t, err)
		_, err = svc.AddEdge(context.Background(), DefaultEnvironment, Edge{ID: "e1", Source: "svc-api", Target: "db", Type: "relation", Data: EdgeData{Kind: KindDependsOn}})
		require.NoError(t, err)
		call := toolByName(t, MCPTools(svc), "topology_remove_edge").Call
		_, err = call(context.Background(), map[string]any{"edge_id": "e1"})
		require.NoError(t, err)
		c, _ := svc.Get(context.Background(), DefaultEnvironment)
		assert.Empty(t, c.Edges)
	})
	t.Run("missing edge id is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(newTestService(newFakeRepo(), newFakeBus())), "topology_remove_edge").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
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

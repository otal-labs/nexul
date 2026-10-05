package topology

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func callTool(t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(t.Context(), json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

// seededService has the service node svc-api and the network node net-main on workspace ws-a's canvas.
func seededService(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	s := newTestService(repo, newFakeBus())
	_, err := s.AddServiceNode(t.Context(), validNode())
	require.NoError(t, err)
	_, err = s.AddNode(t.Context(), "ws-a", validNetworkNode())
	require.NoError(t, err)
	return s, repo
}

func nodeIDs(c canvasResult) []string {
	var out []string
	for _, n := range c.Nodes {
		out = append(out, n.ID)
	}
	return out
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(newFakeRepo(), newFakeBus())) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
	}
	assert.Equal(t, []string{"topology_get", "topology_update"}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	tests := []struct {
		name, tool, args string
		want             error
		msg              string
	}{
		{"topology_get needs a workspace", "topology_get", `{}`, apperrs.ErrInvalid, "workspace_id"},
		{"topology_update needs a workspace", "topology_update", `{"remove_edge_ids":["e1"]}`, apperrs.ErrInvalid, "workspace_id"},
		{"topology_update with nothing to change", "topology_update", `{"workspace_id":"ws-a"}`, apperrs.ErrInvalid, "add_nodes"},
		{"topology_update rejects an unknown key", "topology_update", `{"workspace_id":"ws-a","node_id":"x"}`, apperrs.ErrInvalid, ""},
		{"a node needs a name", "topology_update", `{"workspace_id":"ws-a","add_nodes":[{"id":"n","type":"network"}]}`, apperrs.ErrInvalid, ""},
		{"a service node cannot be added", "topology_update", `{"workspace_id":"ws-a","add_nodes":[{"id":"svc-2","type":"service","name":"api"}]}`, apperrs.ErrInvalid, "managed from stacks"},
		{"an external node needs a label", "topology_update", `{"workspace_id":"ws-a","add_nodes":[{"id":"ext","type":"external","name":"CF"}]}`, apperrs.ErrInvalid, "label"},
		{"a duplicate node id", "topology_update", `{"workspace_id":"ws-a","add_nodes":[{"id":"net-main","type":"network","name":"dup"}]}`, apperrs.ErrConflict, "nothing was applied"},
		{"a service node cannot be removed", "topology_update", `{"workspace_id":"ws-a","remove_node_ids":["svc-api"]}`, apperrs.ErrInvalid, "auto-managed"},
		{"removing a missing node", "topology_update", `{"workspace_id":"ws-a","remove_node_ids":["ghost"]}`, apperrs.ErrNotFound, "topology_get"},
		{"removing a missing edge", "topology_update", `{"workspace_id":"ws-a","remove_edge_ids":["ghost"]}`, apperrs.ErrNotFound, "topology_get"},
		{"an edge with an unknown kind", "topology_update", `{"workspace_id":"ws-a","add_edges":[{"id":"e","source":"svc-api","target":"net-main","kind":"calls"}]}`, apperrs.ErrInvalid, "kind"},
		{"an edge to a missing node", "topology_update", `{"workspace_id":"ws-a","add_edges":[{"id":"e","source":"svc-api","target":"ghost","kind":"depends_on"}]}`, apperrs.ErrInvalid, "ghost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := seededService(t)
			_, err := callTool(t, s, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

func TestTopologyGet(t *testing.T) {
	t.Run("reads the workspace's canvas", func(t *testing.T) {
		s, _ := seededService(t)
		got, err := callTool(t, s, "topology_get", `{"workspace_id":"ws-a"}`)
		require.NoError(t, err)
		c := got.(canvasResult)
		assert.Equal(t, "ws-a", c.WorkspaceID)
		assert.Equal(t, []string{"net-main", "svc-api"}, nodeIDs(c))
	})
	t.Run("another workspace has its own drawing", func(t *testing.T) {
		s, _ := seededService(t)
		got, err := callTool(t, s, "topology_get", `{"workspace_id":"ws-b"}`)
		require.NoError(t, err)
		assert.Equal(t, []string{"svc-api"}, nodeIDs(got.(canvasResult)))
	})
	t.Run("a storage failure surfaces", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("boom")
		_, err := callTool(t, newTestService(repo, newFakeBus()), "topology_get", `{"workspace_id":"ws-a"}`)
		require.Error(t, err)
	})
}

func TestTopologyUpdate_AppliesInOrder(t *testing.T) {
	s, repo := seededService(t)
	_, err := s.AddEdge(t.Context(), "ws-a", Edge{ID: "old", Type: "relation", Source: "svc-api", Target: "net-main", Data: EdgeData{Kind: KindConnectsTo}})
	require.NoError(t, err)

	got, err := callTool(t, s, "topology_update", `{
		"workspace_id":"ws-a",
		"remove_edge_ids":["old"],
		"remove_node_ids":["net-main"],
		"add_nodes":[{"id":"db","type":"external","name":"Postgres","label":"database","x":10,"y":20},{"id":"net-2","type":"network","name":"back"}],
		"add_edges":[{"id":"api-db","source":"svc-api","target":"db","kind":"depends_on"}]}`)
	require.NoError(t, err)
	c := got.(canvasResult)
	assert.Equal(t, []string{"db", "net-2", "svc-api"}, nodeIDs(c))
	assert.Equal(t, Position{X: 10, Y: 20}, c.Nodes[0].Position)
	assert.Equal(t, LabelDatabase, c.Nodes[0].Data.Label)
	require.Len(t, c.Edges, 1)
	assert.Equal(t, "api-db", c.Edges[0].ID)
	assert.Equal(t, TopicUpdated, repo.outbox[len(repo.outbox)-1].Topic)
}

func TestTopologyUpdate_StopsAtTheFirstFailureAndSaysWhatApplied(t *testing.T) {
	s, _ := seededService(t)
	_, err := callTool(t, s, "topology_update", `{
		"workspace_id":"ws-a",
		"add_nodes":[{"id":"net-2","type":"network","name":"back"}],
		"add_edges":[{"id":"e1","source":"svc-api","target":"ghost","kind":"depends_on"},{"id":"e2","source":"svc-api","target":"net-2","kind":"depends_on"}]}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "already applied: added node net-2")
	c, err := s.Get(t.Context(), "ws-a")
	require.NoError(t, err)
	assert.Len(t, c.Nodes, 3, "the node added before the failure stays")
	assert.Empty(t, c.Edges, "nothing after the failure runs")
}

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

// seededService has the service node svc-api and the network node net-main on the workspace canvas.
func seededService(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	s := newTestService(repo, newFakeBus())
	_, err := s.AddNode(t.Context(), DefaultEnvironment, validNode())
	require.NoError(t, err)
	_, err = s.AddNode(t.Context(), DefaultEnvironment, validNetworkNode())
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
		{"topology_get of an environment never saved", "topology_get", `{"environment":"prdo"}`, apperrs.ErrNotFound, "omit environment"},
		{"topology_update of an environment never saved", "topology_update", `{"environment":"prdo","remove_edge_ids":["e1"]}`, apperrs.ErrNotFound, "omit environment"},
		{"topology_update with nothing to change", "topology_update", `{}`, apperrs.ErrInvalid, "add_nodes"},
		{"topology_update rejects an unknown key", "topology_update", `{"node_id":"x"}`, apperrs.ErrInvalid, ""},
		{"a node needs a name", "topology_update", `{"add_nodes":[{"id":"n","type":"network"}]}`, apperrs.ErrInvalid, ""},
		{"a service node cannot be added", "topology_update", `{"add_nodes":[{"id":"svc-2","type":"service","name":"api"}]}`, apperrs.ErrInvalid, "managed from stacks"},
		{"an external node needs a label", "topology_update", `{"add_nodes":[{"id":"ext","type":"external","name":"CF"}]}`, apperrs.ErrInvalid, "label"},
		{"a duplicate node id", "topology_update", `{"add_nodes":[{"id":"net-main","type":"network","name":"dup"}]}`, apperrs.ErrConflict, "nothing was applied"},
		{"a service node cannot be removed", "topology_update", `{"remove_node_ids":["svc-api"]}`, apperrs.ErrInvalid, "auto-managed"},
		{"removing a missing node", "topology_update", `{"remove_node_ids":["ghost"]}`, apperrs.ErrNotFound, "topology_get"},
		{"removing a missing edge", "topology_update", `{"remove_edge_ids":["ghost"]}`, apperrs.ErrNotFound, "topology_get"},
		{"an edge with an unknown kind", "topology_update", `{"add_edges":[{"id":"e","source":"svc-api","target":"net-main","kind":"calls"}]}`, apperrs.ErrInvalid, "kind"},
		{"an edge to a missing node", "topology_update", `{"add_edges":[{"id":"e","source":"svc-api","target":"ghost","kind":"depends_on"}]}`, apperrs.ErrInvalid, "ghost"},
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
	t.Run("defaults to the workspace canvas", func(t *testing.T) {
		s, _ := seededService(t)
		got, err := callTool(t, s, "topology_get", `{}`)
		require.NoError(t, err)
		c := got.(canvasResult)
		assert.Equal(t, DefaultEnvironment, c.Environment)
		assert.Equal(t, []string{"svc-api", "net-main"}, nodeIDs(c))
	})
	t.Run("an unsaved workspace canvas is empty, not an error", func(t *testing.T) {
		got, err := callTool(t, newTestService(newFakeRepo(), newFakeBus()), "topology_get", `{"environment":"default"}`)
		require.NoError(t, err)
		assert.Empty(t, got.(canvasResult).Nodes)
	})
	t.Run("a saved environment is readable", func(t *testing.T) {
		s, _ := seededService(t)
		_, err := s.AddNode(t.Context(), "staging", validNetworkNode())
		require.NoError(t, err)
		got, err := callTool(t, s, "topology_get", `{"environment":"staging"}`)
		require.NoError(t, err)
		assert.Equal(t, []string{"net-main"}, nodeIDs(got.(canvasResult)))
	})
	t.Run("a storage failure surfaces", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("boom")
		_, err := callTool(t, newTestService(repo, newFakeBus()), "topology_get", `{"environment":"staging"}`)
		require.Error(t, err)
	})
}

func TestTopologyUpdate_AppliesInOrder(t *testing.T) {
	s, repo := seededService(t)
	_, err := s.AddEdge(t.Context(), DefaultEnvironment, Edge{ID: "old", Type: "relation", Source: "svc-api", Target: "net-main", Data: EdgeData{Kind: KindConnectsTo}})
	require.NoError(t, err)

	got, err := callTool(t, s, "topology_update", `{
		"remove_edge_ids":["old"],
		"remove_node_ids":["net-main"],
		"add_nodes":[{"id":"db","type":"external","name":"Postgres","label":"database","x":10,"y":20},{"id":"net-2","type":"network","name":"back"}],
		"add_edges":[{"id":"api-db","source":"svc-api","target":"db","kind":"depends_on"}]}`)
	require.NoError(t, err)
	c := got.(canvasResult)
	assert.Equal(t, []string{"svc-api", "db", "net-2"}, nodeIDs(c))
	assert.Equal(t, Position{X: 10, Y: 20}, c.Nodes[1].Position)
	assert.Equal(t, LabelDatabase, c.Nodes[1].Data.Label)
	require.Len(t, c.Edges, 1)
	assert.Equal(t, "api-db", c.Edges[0].ID)
	assert.Equal(t, TopicUpdated, repo.outbox[len(repo.outbox)-1].Topic)
}

func TestTopologyUpdate_StopsAtTheFirstFailureAndSaysWhatApplied(t *testing.T) {
	s, _ := seededService(t)
	_, err := callTool(t, s, "topology_update", `{
		"add_nodes":[{"id":"net-2","type":"network","name":"back"}],
		"add_edges":[{"id":"e1","source":"svc-api","target":"ghost","kind":"depends_on"},{"id":"e2","source":"svc-api","target":"net-2","kind":"depends_on"}]}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "already applied: added node net-2")
	c, err := s.Get(t.Context(), DefaultEnvironment)
	require.NoError(t, err)
	assert.Len(t, c.Nodes, 3, "the node added before the failure stays")
	assert.Empty(t, c.Edges, "nothing after the failure runs")
}

package topology

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the topology tools: read a workspace's canvas, and change its hand-drawn nodes and edges.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{topologyGetTool(s), topologyUpdateTool(s)}
}

// canvasResult is the canvas without the saved camera position, which only the web app uses.
type canvasResult struct {
	WorkspaceID string `json:"workspace_id"`
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
}

type topologyGetIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace whose canvas to read, by id from workspace_list."`
}

func topologyGetTool(s *Service) mcptool.Tool {
	return mcptool.New("topology_get", "Get topology",
		"Returns a workspace's infrastructure map: service nodes (one per container of the workspace's stacks, plus "+
			"the gateways routing to them, kept in sync with deploys, with live status and address), network and "+
			"external nodes people drew, and the typed relations between them. Each workspace has its own canvas. "+
			"Use it to understand how services connect, and topology_update to draw on it.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in topologyGetIn) (any, error) {
			c, err := s.Get(ctx, in.WorkspaceID)
			if err != nil {
				return nil, err
			}
			return toCanvasResult(in.WorkspaceID, c), nil
		})
}

type nodeIn struct {
	ID    string  `json:"id" jsonschema:"A new node id, unique on the canvas, for example net-backend."`
	Type  string  `json:"type" jsonschema:"network or external; service nodes come and go with stacks on their own."`
	Name  string  `json:"name" jsonschema:"The node's display name."`
	Label string  `json:"label,omitempty" jsonschema:"Required for an external node: domain, tunnel, proxy, database, api, or other."`
	URL   string  `json:"url,omitempty" jsonschema:"A link for an external node, for example https://status.example.com."`
	X     float64 `json:"x,omitzero" jsonschema:"Horizontal position on the canvas. Defaults to 0."`
	Y     float64 `json:"y,omitzero" jsonschema:"Vertical position on the canvas. Defaults to 0."`
}

type edgeIn struct {
	ID     string `json:"id" jsonschema:"A new edge id, unique on the canvas, for example api-uses-db."`
	Source string `json:"source" jsonschema:"The id of the node the relation starts at."`
	Target string `json:"target" jsonschema:"The id of the node the relation points to."`
	Kind   string `json:"kind" jsonschema:"depends_on, connects_to, or mounts."`
}

type topologyUpdateIn struct {
	WorkspaceID   string   `json:"workspace_id" jsonschema:"The workspace whose canvas to change, by id from workspace_list."`
	AddNodes      []nodeIn `json:"add_nodes,omitempty" jsonschema:"Network or external nodes to add."`
	RemoveNodeIDs []string `json:"remove_node_ids,omitempty" jsonschema:"Ids of network or external nodes to remove, with every edge that touches them."`
	AddEdges      []edgeIn `json:"add_edges,omitempty" jsonschema:"Relations to add between existing or newly added nodes."`
	RemoveEdgeIDs []string `json:"remove_edge_ids,omitempty" jsonschema:"Ids of relations to remove."`
}

// change is one topology_update step; each commits on its own, so a failure reports the ones already applied.
type change struct {
	done  string
	apply func() (*Canvas, error)
}

func topologyUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("topology_update", "Update topology",
		"Draws on the infrastructure map: adds and removes network and external nodes and the typed relations "+
			"between nodes. Service nodes are managed from stacks and cannot be added or removed here. Changes apply "+
			"in order (removed edges, removed nodes, added nodes, added edges), each saved on its own; the first "+
			"failure stops the rest and the error lists what was already applied. Returns the updated canvas.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in topologyUpdateIn) (any, error) {
			changes := topologyChanges(ctx, s, in.WorkspaceID, in)
			if len(changes) == 0 {
				return nil, fmt.Errorf("%w: nothing to change; send add_nodes, remove_node_ids, add_edges, or remove_edge_ids", apperrs.ErrInvalid)
			}
			var c *Canvas
			var err error
			for i, ch := range changes {
				if c, err = ch.apply(); err != nil {
					return nil, appliedBefore(err, changes[:i])
				}
			}
			return toCanvasResult(in.WorkspaceID, c), nil
		})
}

func topologyChanges(ctx context.Context, s *Service, workspaceID string, in topologyUpdateIn) []change {
	var out []change
	for _, id := range in.RemoveEdgeIDs {
		out = append(out, change{"removed edge " + id, func() (*Canvas, error) { return s.RemoveEdge(ctx, workspaceID, id) }})
	}
	for _, id := range in.RemoveNodeIDs {
		out = append(out, change{"removed node " + id, func() (*Canvas, error) { return s.RemoveNode(ctx, workspaceID, id) }})
	}
	for _, n := range in.AddNodes {
		out = append(out, change{"added node " + n.ID, func() (*Canvas, error) { return addNode(ctx, s, workspaceID, n) }})
	}
	for _, e := range in.AddEdges {
		edge := Edge{ID: e.ID, Type: "relation", Source: e.Source, Target: e.Target, Data: EdgeData{Kind: RelationKind(e.Kind)}}
		out = append(out, change{"added edge " + e.ID, func() (*Canvas, error) { return s.AddEdge(ctx, workspaceID, edge) }})
	}
	return out
}

func addNode(ctx context.Context, s *Service, workspaceID string, n nodeIn) (*Canvas, error) {
	return s.AddNode(ctx, workspaceID, Node{
		ID: n.ID, Type: NodeType(n.Type), Position: Position{X: n.X, Y: n.Y},
		Data: NodeData{Name: n.Name, Label: ExternalLabel(n.Label), URL: n.URL},
	})
}

func appliedBefore(err error, done []change) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		err = fmt.Errorf("%w; topology_get shows the canvas's node and edge ids", err)
	}
	if len(done) == 0 {
		return fmt.Errorf("%w (nothing was applied)", err)
	}
	names := make([]string, 0, len(done))
	for _, ch := range done {
		names = append(names, ch.done)
	}
	return fmt.Errorf("%w (already applied: %s)", err, strings.Join(names, ", "))
}

func toCanvasResult(workspaceID string, c *Canvas) canvasResult {
	return canvasResult{WorkspaceID: workspaceID, Nodes: c.Nodes, Edges: c.Edges}
}

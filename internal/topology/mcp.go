package topology

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the topology tool definitions; environment defaults to the canonical one.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "topology_get",
			Description: "Get the topology canvas for an environment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"environment": map[string]any{"type": "string"},
				},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return s.Get(ctx, environmentArg(args))
			},
		},
		{
			Name:        "topology_add_node",
			Description: "Add a network or external node to the topology canvas (service nodes are auto-managed from the deploy domain).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"environment": map[string]any{"type": "string"},
					"type":        map[string]any{"type": "string", "enum": []string{"network", "external"}},
					"id":          map[string]any{"type": "string"},
					"name":        map[string]any{"type": "string"},
					"label":       map[string]any{"type": "string", "enum": []string{"domain", "tunnel", "proxy", "database", "api", "other"}},
					"url":         map[string]any{"type": "string"},
					"x":           map[string]any{"type": "number"},
					"y":           map[string]any{"type": "number"},
				},
				"required": []string{"type", "id", "name"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				kind, err := mcptool.RequiredString(args, "type")
				if err != nil {
					return nil, err
				}
				if kind != string(NodeNetwork) && kind != string(NodeExternal) {
					return nil, fmt.Errorf("%w: type must be network or external (service nodes are auto-managed)", apperrs.ErrInvalid)
				}
				vals, err := mcptool.RequiredStrings(args, "id", "name")
				if err != nil {
					return nil, err
				}
				id, name := vals[0], vals[1]
				node := Node{
					ID:       id,
					Type:     NodeType(kind),
					Position: Position{X: floatArg(args["x"]), Y: floatArg(args["y"])},
					Data: NodeData{
						Name: name,
						URL:  mcptool.OptionalString(args["url"]),
					},
				}
				if kind == string(NodeExternal) {
					label, err := mcptool.RequiredString(args, "label")
					if err != nil {
						return nil, err
					}
					node.Data.Label = ExternalLabel(label)
				}
				return s.AddNode(ctx, environmentArg(args), node)
			},
		},
		{
			Name:        "topology_remove_node",
			Description: "Remove a network or external node and the edges that reference it (service nodes are auto-managed).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"environment": map[string]any{"type": "string"},
					"node_id":     map[string]any{"type": "string"},
				},
				"required": []string{"node_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				nodeID, err := mcptool.RequiredString(args, "node_id")
				if err != nil {
					return nil, err
				}
				return s.RemoveNode(ctx, environmentArg(args), nodeID)
			},
		},
		{
			Name:        "topology_add_edge",
			Description: "Add a typed relation between two nodes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"environment": map[string]any{"type": "string"},
					"id":          map[string]any{"type": "string"},
					"source":      map[string]any{"type": "string"},
					"target":      map[string]any{"type": "string"},
					"kind":        map[string]any{"type": "string", "enum": []string{"depends_on", "connects_to", "mounts"}},
				},
				"required": []string{"id", "source", "target", "kind"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "source", "target", "kind")
				if err != nil {
					return nil, err
				}
				id, source, target, kind := vals[0], vals[1], vals[2], vals[3]
				edge := Edge{
					ID:     id,
					Type:   "relation",
					Source: source,
					Target: target,
					Data:   EdgeData{Kind: RelationKind(kind)},
				}
				return s.AddEdge(ctx, environmentArg(args), edge)
			},
		},
		{
			Name:        "topology_remove_edge",
			Description: "Remove a relation by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"environment": map[string]any{"type": "string"},
					"edge_id":     map[string]any{"type": "string"},
				},
				"required": []string{"edge_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				edgeID, err := mcptool.RequiredString(args, "edge_id")
				if err != nil {
					return nil, err
				}
				return s.RemoveEdge(ctx, environmentArg(args), edgeID)
			},
		},
	}
}

func environmentArg(args map[string]any) string {
	if env := mcptool.OptionalString(args["environment"]); env != "" {
		return env
	}
	return DefaultEnvironment
}

func floatArg(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

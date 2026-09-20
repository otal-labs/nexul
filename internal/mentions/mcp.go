package mentions

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the mention tool definitions; LLMs reference targets the same markdown links humans do.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "mention_search",
			Description: "Search tickets and docs to @-mention in a document. Returns canonical reference ids.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"limit": map[string]any{"type": "integer"},
				},
				"required": []string{"query"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				query, err := mcptool.RequiredString(args, "query")
				if err != nil {
					return nil, err
				}
				return s.Search(ctx, query, intArg(args["limit"]))
			},
		},
		{
			Name:        "mention_resolve",
			Description: "Resolve a batch of mention references to live chips (current title, status label, access-aware clickability) for a document render.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"refs": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type": map[string]any{"type": "string", "description": "ticket or doc"},
								"id":   map[string]any{"type": "string"},
							},
							"required": []string{"type", "id"},
						},
					},
				},
				"required": []string{"refs"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				refs, err := refsArg(args["refs"])
				if err != nil {
					return nil, err
				}
				return s.Resolve(ctx, refs)
			},
		},
	}
}

func intArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}

func refsArg(v any) ([]Ref, error) {
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("%w: refs is required", apperrs.ErrInvalid)
	}
	out := make([]Ref, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		id, _ := m["id"].(string)
		if typ == "" || id == "" {
			continue
		}
		out = append(out, Ref{Type: typ, ID: id})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: refs must contain at least one valid {type, id}", apperrs.ErrInvalid)
	}
	return out, nil
}

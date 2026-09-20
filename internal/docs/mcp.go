package docs

import (
	"context"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the docs tool definitions (named domain_action).
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "doc_create",
			Description: "Create a doc from a markdown body in the given project and return it as markdown.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"title":      map[string]any{"type": "string"},
					"body":       map[string]any{"type": "string", "description": "Markdown body"},
				},
				"required": []string{"project_id", "title"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "project_id", "title")
				if err != nil {
					return nil, err
				}
				projectID, title := vals[0], vals[1]
				d, err := s.Create(ctx, projectID, title, mcptool.OptionalString(args["body"]))
				if err != nil {
					return nil, err
				}
				return docToMarkdown(ctx, s, d)
			},
		},
		{
			Name:        "doc_get",
			Description: "Fetch a single doc by id, rendered as markdown.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				d, err := s.Get(ctx, id)
				if err != nil {
					return nil, err
				}
				return docToMarkdown(ctx, s, d)
			},
		},
		{
			Name:        "doc_search",
			Description: "Full-text search over doc titles and bodies.",
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
			Name:        "doc_update",
			Description: "Update a doc's title and body (markdown), bumping its version.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string"},
					"title": map[string]any{"type": "string"},
					"body":  map[string]any{"type": "string", "description": "Markdown body"},
				},
				"required": []string{"id", "title"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "title")
				if err != nil {
					return nil, err
				}
				id, title := vals[0], vals[1]
				d, err := s.Update(ctx, id, title, mcptool.OptionalString(args["body"]))
				if err != nil {
					return nil, err
				}
				return docToMarkdown(ctx, s, d)
			},
		},
		{
			Name:        "doc_archive",
			Description: "Archive a doc so it is hidden from search.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Archive(ctx, id)
			},
		},
		{
			Name:        "doc_restore",
			Description: "Restore an archived doc.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Restore(ctx, id)
			},
		},
	}
}

// docToMarkdown returns a copy of the doc with its body rendered as markdown for LLM consumption.
func docToMarkdown(ctx context.Context, s *Service, d *Doc) (any, error) {
	md, err := richtext.ToMarkdown(d.Body)
	if err != nil {
		return nil, err
	}
	out := *d
	out.Body = md
	return &out, nil
}

func intArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}

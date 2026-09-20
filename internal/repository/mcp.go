package repository

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the repository tool definitions: the wizard's first two steps, list and scan.
func MCPTools(s Scanner) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "repository_list",
			Description: "List repositories visible through the connected git provider installation.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return ListRepos(ctx, s)
			},
		},
		{
			Name:        "repository_scan",
			Description: "Scan a repository's tree for deployable candidates (compose stacks, standalone Dockerfiles) and env keys.",
			InputSchema: objectSchema(map[string]any{
				"owner": map[string]any{"type": "string"},
				"name":  map[string]any{"type": "string"},
				"ref":   map[string]any{"type": "string"},
			}, "owner", "name"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "owner", "name")
				if err != nil {
					return nil, err
				}
				return Scan(ctx, s, vals[0], vals[1], mcptool.OptionalString(args["ref"]))
			},
		},
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

package access

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// MCPTools' acting user comes from the identity context the MCP server injects.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "access_list_grants",
			Description: "List permission grants on a document, or the user exclusions on a play (resource_type: play).",
			InputSchema: objectSchema(map[string]any{
				"doc_id":        map[string]any{"type": "string"},
				"resource_type": map[string]any{"type": "string", "enum": []string{"doc", "play"}},
				"resource_id":   map[string]any{"type": "string"},
			}),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				resourceType := mcptool.OptionalString(args["resource_type"])
				resourceID := mcptool.OptionalString(args["resource_id"])
				if resourceType == "play" {
					if resourceID == "" {
						return nil, fmt.Errorf("%w: resource_id is required", apperrs.ErrInvalid)
					}
					return s.ListPlayGrants(ctx, actorIDFromCtx(ctx), resourceID)
				}
				docID := mcptool.OptionalString(args["doc_id"])
				if docID == "" {
					docID = resourceID
				}
				if docID == "" {
					return nil, fmt.Errorf("%w: doc_id is required", apperrs.ErrInvalid)
				}
				return s.ListGrants(ctx, actorIDFromCtx(ctx), docID)
			},
		},
		{
			Name: "access_set_grants",
			Description: "Grant or revoke permission actions on documents for users, or deny/undeny plays:run " +
				"on plays (resource_type: play, resource_ids), in one operation.",
			InputSchema: objectSchema(map[string]any{
				"resource_type": map[string]any{"type": "string", "enum": []string{"doc", "play"}},
				"doc_ids":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"resource_ids":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"user_ids":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"actions":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"grant":         map[string]any{"type": "boolean"},
			}, "user_ids", "actions", "grant"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				userIDs, err := requiredStrings(args, "user_ids")
				if err != nil {
					return nil, err
				}
				rawActions, err := requiredStrings(args, "actions")
				if err != nil {
					return nil, err
				}
				actions := make([]permissions.Action, 0, len(rawActions))
				for _, raw := range rawActions {
					a, ok := permissions.ParseAction(raw)
					if !ok {
						return nil, fmt.Errorf("%w: unknown action %s", apperrs.ErrInvalid, raw)
					}
					actions = append(actions, a)
				}
				grant, _ := args["grant"].(bool)
				if mcptool.OptionalString(args["resource_type"]) == "play" {
					resourceIDs, err := requiredStrings(args, "resource_ids")
					if err != nil {
						return nil, err
					}
					if err := s.SetPlayGrants(ctx, actorIDFromCtx(ctx), resourceIDs, userIDs, actions, grant); err != nil {
						return nil, err
					}
					return map[string]string{"status": "ok"}, nil
				}
				docIDs, err := requiredStrings(args, "doc_ids")
				if err != nil {
					return nil, err
				}
				if err := s.SetGrants(ctx, actorIDFromCtx(ctx), docIDs, userIDs, actions, grant); err != nil {
					return nil, err
				}
				return map[string]string{"status": "ok"}, nil
			},
		},
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

func requiredStrings(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("%w: %s is required", apperrs.ErrInvalid, key)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("%w: %s entries must be non-empty strings", apperrs.ErrInvalid, key)
		}
		out = append(out, s)
	}
	return out, nil
}

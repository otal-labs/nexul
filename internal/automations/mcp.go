package automations

import (
	"context"
	"encoding/json"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the automations tool definitions (named domain_action), one per use-case.
func MCPTools(s *Service) []mcptool.Tool {
	return append(crudTools(s), tokenTools(s)...)
}

func crudTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "automation_create",
			Description: "Create a new Custom automation shell and mint its scoped token (returned once).",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"name":   map[string]any{"type": "string"},
				"scopes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, "name", "scopes"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				name, err := mcptool.RequiredString(args, "name")
				if err != nil {
					return nil, err
				}
				scopes, err := stringsArg(args["scopes"])
				if err != nil {
					return nil, err
				}
				a, token, err := s.Create(ctx, actorIDFromCtx(ctx), name, scopes)
				if err != nil {
					return nil, err
				}
				return tokenResponse{Automation: a, Token: token}, nil
			},
		},
		{
			Name:        "automation_list",
			Description: "List all automations, defaults included.",
			InputSchema: mcptool.ObjectSchema(map[string]any{}),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return s.List(ctx, actorIDFromCtx(ctx))
			},
		},
		{
			Name:        "automation_get",
			Description: "Fetch a single automation by id.",
			InputSchema: mcptool.ObjectSchema(map[string]any{"id": map[string]any{"type": "string"}}, "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Get(ctx, actorIDFromCtx(ctx), id)
			},
		},
		{
			Name:        "automation_update_config",
			Description: "Replace an automation's owner-edited config values.",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"id":            map[string]any{"type": "string"},
				"config_values": map[string]any{"type": "object"},
			}, "id", "config_values"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				values, ok := args["config_values"].(map[string]any)
				if !ok {
					values = map[string]any{}
				}
				raw, err := json.Marshal(values)
				if err != nil {
					return nil, err
				}
				return s.UpdateConfigValues(ctx, actorIDFromCtx(ctx), id, raw)
			},
		},
		{
			Name:        "automation_set_enabled",
			Description: "Enable or disable an automation.",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"id":      map[string]any{"type": "string"},
				"enabled": map[string]any{"type": "boolean"},
			}, "id", "enabled"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				enabled, _ := args["enabled"].(bool)
				return s.SetEnabled(ctx, actorIDFromCtx(ctx), id, enabled)
			},
		},
		{
			Name:        "automation_delete",
			Description: "Delete an automation and revoke its token.",
			InputSchema: mcptool.ObjectSchema(map[string]any{"id": map[string]any{"type": "string"}}, "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				if err := s.Delete(ctx, actorIDFromCtx(ctx), id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
	}
}

func tokenTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "automation_mint_token",
			Description: "Rotate an automation's token; the previous token stops authenticating immediately.",
			InputSchema: mcptool.ObjectSchema(map[string]any{"id": map[string]any{"type": "string"}}, "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				a, token, err := s.MintToken(ctx, actorIDFromCtx(ctx), id)
				if err != nil {
					return nil, err
				}
				return tokenResponse{Automation: a, Token: token}, nil
			},
		},
		{
			Name:        "automation_revoke_token",
			Description: "Revoke an automation's token immediately.",
			InputSchema: mcptool.ObjectSchema(map[string]any{"id": map[string]any{"type": "string"}}, "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.RevokeToken(ctx, actorIDFromCtx(ctx), id)
			},
		},
	}
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

func stringsArg(v any) ([]string, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

package auth

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools exposes the caller's own account and account administration through the same use-cases as HTTP.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "account_whoami",
			Description: "Return the account this MCP connection acts as. Call it first to confirm Nexul is reachable.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.Whoami(ctx, actorIDFromContext(ctx))
			},
		},
		{
			Name:        "list_accounts",
			Description: "List registered instance accounts.",
			InputSchema: objectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListAccounts(ctx, actorIDFromContext(ctx))
			},
		},
		accountStatusTool("disable_account", "Disable an instance account.", s.DisableAccount),
		accountStatusTool("reactivate_account", "Reactivate a disabled instance account.", s.ReactivateAccount),
		accountStatusTool("remove_account", "Remove an instance account while preserving authored content.", s.RemoveAccount),
		accountStatusTool("restore_account", "Restore a removed instance account without restoring access.", s.RestoreAccount),
	}
}

func accountStatusTool(name, description string, action func(context.Context, string, string) error) mcptool.Tool {
	return mcptool.Tool{
		Name:        name,
		Description: description,
		InputSchema: objectSchema(map[string]any{"id": map[string]any{"type": "string"}}, "id"),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			id, err := mcptool.RequiredString(args, "id")
			if err != nil {
				return nil, err
			}
			if err := action(ctx, actorIDFromContext(ctx), id); err != nil {
				return nil, err
			}
			return map[string]string{"id": id, "status": "ok"}, nil
		},
	}
}

func actorIDFromContext(ctx context.Context) string {
	if actor, ok := identity.ActorFromCtx(ctx); ok {
		return actor.ID
	}
	return ""
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

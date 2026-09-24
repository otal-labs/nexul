package pairing

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the setup confirmation tools (ADR 0063); MCP is the only surface that writes a confirmation.
func MCPTools(s *Service) []mcptool.Tool {
	computerOnly := computerSchema(nil)
	providerOnly := computerSchema(map[string]any{"provider": providerProperty}, "provider")
	return []mcptool.Tool{
		{
			Name:        "computer_setup_get",
			Description: "Read one of your paired computers' setup confirmation: the overall one and one per provider, each with its confirmed_at (null means unconfirmed) and the skills reported when it was confirmed.",
			InputSchema: computerOnly,
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return withComputer(ctx, args, s.GetSetup)
			},
		},
		{
			Name:        "computer_setup_confirm_provider",
			Description: "Confirm a provider is set up on one of your paired computers: Nexul's MCP server is connected to it and its harness reports the default skills. Pass the skill names the harness reported.",
			InputSchema: computerSchema(map[string]any{
				"provider": providerProperty,
				"skills":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Skill names the harness reported as discovered"},
			}, "provider", "skills"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				skills, err := stringsArg(args, "skills")
				if err != nil {
					return nil, err
				}
				return withProvider(ctx, args, func(ctx context.Context, userID, computerID, provider string) (Setup, error) {
					return s.ConfirmProviderSetup(ctx, userID, computerID, provider, skills)
				})
			},
		},
		{
			Name:        "computer_setup_unconfirm_provider",
			Description: "Withdraw a provider's setup confirmation on one of your paired computers, for example after its skills were removed. The provider cannot run agent work there until confirmed again.",
			InputSchema: providerOnly,
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return withProvider(ctx, args, s.UnconfirmProviderSetup)
			},
		},
		{
			Name:        "computer_setup_confirm",
			Description: "Confirm one of your paired computers is set up overall, the last step of its setup.",
			InputSchema: computerOnly,
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return withComputer(ctx, args, s.ConfirmSetup)
			},
		},
		{
			Name:        "computer_setup_unconfirm",
			Description: "Withdraw the overall setup confirmation of one of your paired computers, for example after the machine was wiped.",
			InputSchema: computerOnly,
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return withComputer(ctx, args, s.UnconfirmSetup)
			},
		},
	}
}

var providerProperty = map[string]any{"type": "string", "description": "The provider's driver kind, such as claude, codex, or opencode"}

func computerSchema(extra map[string]any, required ...string) map[string]any {
	properties := map[string]any{"computer_id": map[string]any{"type": "string"}}
	for k, v := range extra {
		properties[k] = v
	}
	return map[string]any{"type": "object", "properties": properties, "required": append([]string{"computer_id"}, required...)}
}

func withComputer(ctx context.Context, args map[string]any, call func(context.Context, string, string) (Setup, error)) (any, error) {
	computerID, err := mcptool.RequiredString(args, "computer_id")
	if err != nil {
		return nil, err
	}
	return call(ctx, mcpActorID(ctx), computerID)
}

func withProvider(ctx context.Context, args map[string]any, call func(context.Context, string, string, string) (Setup, error)) (any, error) {
	vals, err := mcptool.RequiredStrings(args, "computer_id", "provider")
	if err != nil {
		return nil, err
	}
	return call(ctx, mcpActorID(ctx), vals[0], vals[1])
}

// stringsArg reads a JSON array of strings; any non-string element is invalid rather than silently dropped.
func stringsArg(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key].([]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s must be an array of strings", apperrs.ErrInvalid, key)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s must be an array of strings", apperrs.ErrInvalid, key)
		}
		out = append(out, str)
	}
	return out, nil
}

func mcpActorID(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

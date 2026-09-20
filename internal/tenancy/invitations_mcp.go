package tenancy

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func MCPTools(s *InvitationService) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "create_invitation",
			Description: "Create a single-use invitation link for one or more workspaces.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"grants":          map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
				"expires_in_days": map[string]any{"type": "integer", "enum": []int{1, 7}},
			}, "required": []string{"grants", "expires_in_days"}},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				grants, err := parseInvitationGrants(args["grants"])
				if err != nil {
					return nil, err
				}
				days, ok := args["expires_in_days"].(float64)
				if !ok {
					return nil, fmt.Errorf("expires_in_days is required")
				}
				return s.Create(ctx, actorID(ctx), CreateInvitationInput{Grants: grants, ExpiresInDays: int(days)})
			},
		},
		{
			Name:        "list_invitations",
			Description: "List active invitation bundles the authenticated user may manage.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.List(ctx, actorID(ctx))
			},
		},
		{
			Name:        "revoke_invitation",
			Description: "Revoke an active invitation bundle.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{"invitation_id": map[string]any{"type": "string"}}, "required": []string{"invitation_id"}},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "invitation_id")
				if err != nil {
					return nil, err
				}
				if err := s.Revoke(ctx, actorID(ctx), id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "ok"}, nil
			},
		},
	}
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}

func parseInvitationGrants(raw any) ([]*InvitationGrant, error) {
	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		return nil, fmt.Errorf("grants is required")
	}
	grants := make([]*InvitationGrant, 0, len(entries))
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("grant must be an object")
		}
		workspaceID, _ := entry["workspace_id"].(string)
		roleID, _ := entry["role_id"].(string)
		if workspaceID == "" || roleID == "" {
			return nil, fmt.Errorf("workspace_id and role_id are required")
		}
		allow, err := parseActions(entry["allow"])
		if err != nil {
			return nil, err
		}
		deny, err := parseActions(entry["deny"])
		if err != nil {
			return nil, err
		}
		grants = append(grants, &InvitationGrant{WorkspaceID: workspaceID, RoleID: roleID, Allow: allow, Deny: deny})
	}
	return grants, nil
}

func parseActions(raw any) (permissions.Set, error) {
	if raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("permission set must be an array")
	}
	actions := make([]permissions.Action, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("permission must be a string")
		}
		action, ok := permissions.ParseAction(text)
		if !ok {
			return nil, fmt.Errorf("unknown permission %s", text)
		}
		actions = append(actions, action)
	}
	return permissions.SetOf(actions...), nil
}

package tenancy

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// MCPTools exposes invitations through the same use-cases as HTTP; each call acts as the context's actor.
func MCPTools(s *InvitationService) []mcptool.Tool {
	return []mcptool.Tool{invitationCreateTool(s), invitationListTool(s), invitationDeleteTool(s)}
}

type invitationGrantIn struct {
	WorkspaceID string   `json:"workspace_id" jsonschema:"The workspace the invitation admits the person to."`
	RoleID      string   `json:"role_id" jsonschema:"The role they get in that workspace."`
	Allow       []string `json:"allow,omitempty" jsonschema:"Permissions allowed on top of the role, each <domain>:<action>, for example [\"docs:write\"]."`
	Deny        []string `json:"deny,omitempty" jsonschema:"Permissions denied despite the role, each <domain>:<action>, for example [\"members:write\"]."`
}

type invitationCreateIn struct {
	Grants        []invitationGrantIn `json:"grants" jsonschema:"One entry per workspace the invitation admits the person to; at least one."`
	ExpiresInDays int                 `json:"expires_in_days,omitzero" jsonschema:"How long the link stays valid: 1 or 7 days. Defaults to 7."`
}

type invitationListIn struct {
	mcptool.PageArgs
}

type invitationDeleteIn struct {
	ID string `json:"id" jsonschema:"The invitation's id, from invitation_list."`
}

func invitationCreateTool(s *InvitationService) mcptool.Tool {
	return mcptool.New("invitation_create", "Create invitation",
		"Creates a single-use invitation link that admits one person to the instance with the given role, plus "+
			"optional permission overwrites, in each listed workspace. You need members:write in every one of them. "+
			"The returned url carries the only copy of the link's secret and is never shown again, so hand it straight "+
			"to the person being invited; invitation_list shows the invitation without it, and invitation_delete "+
			"revokes it.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in invitationCreateIn) (any, error) {
			grants, err := invitationGrants(in.Grants)
			if err != nil {
				return nil, err
			}
			return s.Create(ctx, actorID(ctx), CreateInvitationInput{Grants: grants, ExpiresInDays: in.ExpiresInDays})
		})
}

func invitationListTool(s *InvitationService) mcptool.Tool {
	return mcptool.New("invitation_list", "List invitations",
		"Lists the unredeemed, unexpired invitations you may manage, meaning you hold members:write in every "+
			"workspace each one grants. Each shows its workspaces and roles, who invited, and when it expires, never "+
			"its link. Use invitation_create for a new link and invitation_delete to revoke one.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in invitationListIn) (any, error) {
			invitations, err := s.List(ctx, actorID(ctx))
			if err != nil {
				return nil, err
			}
			return mcptool.Paginate(invitations, in.PageArgs), nil
		})
}

func invitationDeleteTool(s *InvitationService) mcptool.Tool {
	return mcptool.New("invitation_delete", "Revoke invitation",
		"Revokes an unredeemed invitation so its link stops working. You need members:write in every workspace it "+
			"grants. An invitation already redeemed, expired, or revoked is not found; invitation_list shows the "+
			"ones still open. Returns {id, deleted: true}.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in invitationDeleteIn) (any, error) {
			if err := s.Revoke(ctx, actorID(ctx), in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

func invitationGrants(in []invitationGrantIn) ([]*InvitationGrant, error) {
	grants := make([]*InvitationGrant, 0, len(in))
	for _, g := range in {
		allow, err := actionSet(g.Allow)
		if err != nil {
			return nil, err
		}
		deny, err := actionSet(g.Deny)
		if err != nil {
			return nil, err
		}
		grants = append(grants, &InvitationGrant{WorkspaceID: g.WorkspaceID, RoleID: g.RoleID, Allow: allow, Deny: deny})
	}
	return grants, nil
}

func actionSet(raw []string) (permissions.Set, error) {
	actions := make([]permissions.Action, 0, len(raw))
	for _, r := range raw {
		a, ok := permissions.ParseAction(r)
		if !ok {
			return nil, fmt.Errorf("%w: unknown permission %q; a permission is <domain>:<action>, for example docs:write", apperrs.ErrInvalid, r)
		}
		actions = append(actions, a)
	}
	return permissions.SetOf(actions...), nil
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}

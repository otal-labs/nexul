package tenancy

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools exposes invitations through the same use-cases as HTTP; each call acts as the context's actor.
func MCPTools(s *InvitationService) []mcptool.Tool {
	return []mcptool.Tool{invitationCreateTool(s), invitationListTool(s), invitationDeleteTool(s)}
}

type invitationGrantIn struct {
	WorkspaceID string   `json:"workspace_id" jsonschema:"The workspace the invitation admits the person to."`
	Roles       []string `json:"roles" jsonschema:"The role ids they get in that workspace, at least one and never the Owner role; workspace_list with the workspace's id lists them."`
}

type invitationCreateIn struct {
	Grants        []invitationGrantIn `json:"grants" jsonschema:"One entry per workspace the invitation admits the person to; at least one."`
	ExpiresInDays int                 `json:"expires_in_days,omitzero" jsonschema:"How long the link stays valid: 1 or 7 days. Defaults to 7."`
}

// invitationResult is an invitation as the tools show it: each workspace with the roles it gives, never its link.
type invitationResult struct {
	ID        string                  `json:"id"`
	InvitedBy string                  `json:"invited_by"`
	CreatedAt time.Time               `json:"created_at"`
	ExpiresAt time.Time               `json:"expires_at"`
	Grants    []invitationGrantResult `json:"grants"`
}

type invitationGrantResult struct {
	WorkspaceID   string      `json:"workspace_id"`
	WorkspaceName string      `json:"workspace_name,omitempty"`
	Roles         []*TeamRole `json:"roles"`
}

type createdInvitationResult struct {
	Invitation invitationResult `json:"invitation"`
	URL        string           `json:"url"`
}

type invitationListIn struct {
	mcptool.PageArgs
}

type invitationDeleteIn struct {
	ID string `json:"id" jsonschema:"The invitation's id, from invitation_list."`
}

func invitationCreateTool(s *InvitationService) mcptool.Tool {
	return mcptool.New("invitation_create", "Create invitation",
		"Creates a single-use invitation link that admits one person to the instance with the given roles in each "+
			"listed workspace, and nothing per person. You need members:write in every one of them, and may give only "+
			"roles whose permissions you hold there; every role is checked again when the link is redeemed. Which "+
			"projects the person is in is set on each project, with permission_overwrite_update. "+
			"The returned url carries the only copy of the link's secret and is never shown again, so hand it straight "+
			"to the person being invited; invitation_list shows the invitation without it, and invitation_delete "+
			"revokes it.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in invitationCreateIn) (any, error) {
			grants, err := invitationGrants(in.Grants)
			if err != nil {
				return nil, err
			}
			created, err := s.Create(ctx, actorID(ctx), CreateInvitationInput{Grants: grants, ExpiresInDays: in.ExpiresInDays})
			if err != nil {
				return nil, err
			}
			return createdInvitationResult{Invitation: toInvitationResult(created.Invitation), URL: created.URL}, nil
		})
}

func invitationListTool(s *InvitationService) mcptool.Tool {
	return mcptool.New("invitation_list", "List invitations",
		"Lists the unredeemed, unexpired invitations you may manage, meaning you hold members:write in every "+
			"workspace each one grants. Each shows its workspaces with the roles it gives there (id, name, whether it "+
			"is the Owner role), who invited, and when it expires, never its link. Use invitation_create for a new link and invitation_delete to revoke one.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in invitationListIn) (any, error) {
			invitations, err := s.List(ctx, actorID(ctx))
			if err != nil {
				return nil, err
			}
			out := make([]invitationResult, len(invitations))
			for i, inv := range invitations {
				out[i] = toInvitationResult(inv)
			}
			return mcptool.Paginate(out, in.PageArgs), nil
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
		if len(g.Roles) == 0 {
			return nil, fmt.Errorf("%w: the grant for workspace %s needs at least one role in roles; workspace_list with its id lists them", apperrs.ErrInvalid, g.WorkspaceID)
		}
		grants = append(grants, &InvitationGrant{WorkspaceID: g.WorkspaceID, RoleID: g.Roles[0], RoleIDs: g.Roles})
	}
	return grants, nil
}

func toInvitationResult(inv *Invitation) invitationResult {
	out := invitationResult{ID: inv.ID, InvitedBy: inv.InvitedBy, CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt, Grants: make([]invitationGrantResult, len(inv.Grants))}
	for i, g := range inv.Grants {
		roles := g.Roles
		if roles == nil {
			for _, id := range g.Held() {
				roles = append(roles, &TeamRole{ID: id})
			}
		}
		out.Grants[i] = invitationGrantResult{WorkspaceID: g.WorkspaceID, WorkspaceName: g.WorkspaceName, Roles: roles}
	}
	return out
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}

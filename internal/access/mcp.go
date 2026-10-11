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
	return []mcptool.Tool{accessGrantListTool(s), accessGrantUpdateTool(s)}
}

type accessGrantListIn struct {
	ResourceType string `json:"resource_type" jsonschema:"doc or play; project too when user_id is given."`
	ResourceID   string `json:"resource_id" jsonschema:"The doc's, play's, or project's id."`
	UserID       string `json:"user_id,omitempty" jsonschema:"An account id, yours or another's, as account_list shows them; returns that person's effective answers instead of the overwrites."`
	mcptool.PageArgs
}

type accessGrantUpdateIn struct {
	ResourceType string   `json:"resource_type" jsonschema:"doc or play."`
	ResourceIDs  []string `json:"resource_ids" jsonschema:"The docs' or plays' ids; every one gets the same change."`
	UserIDs      []string `json:"user_ids" jsonschema:"The account ids the change applies to, as permission_overwrite_list or account_list shows them."`
	Actions      []string `json:"actions" jsonschema:"Permissions to set, each <domain>:<action>, for example [\"docs:read\", \"docs:write\"]. On a play only plays:run."`
	State        string   `json:"state,omitempty" jsonschema:"allow, fallback, or deny: allow and deny hold the actions here whatever the role says, fallback removes this rule's say so the role decides. Takes precedence over grant."`
	Grant        *bool    `json:"grant,omitempty" jsonschema:"Without state: true grants the actions (on a play, lifts a deny); false revokes a grant (on a play, denies running it)."`
}

// state is in.State, else what grant means; neither is invalid.
func (in accessGrantUpdateIn) state() (string, error) {
	if in.State != "" {
		return in.State, nil
	}
	if in.Grant == nil {
		return "", fmt.Errorf("%w: state or grant is required", apperrs.ErrInvalid)
	}
	return GrantState(in.ResourceType, *in.Grant), nil
}

// grantResult is one user's overwrite on the resource that was asked for.
type grantResult struct {
	UserID string          `json:"user_id"`
	Login  string          `json:"login,omitempty"`
	Name   string          `json:"name,omitempty"`
	Allow  permissions.Set `json:"allow"`
	Deny   permissions.Set `json:"deny"`
}

func accessGrantListTool(s *Service) mcptool.Tool {
	return mcptool.New("permission_overwrite_list", "List permission overwrites",
		"Lists the per-user permission overwrites on one doc or play: on a doc, who is allowed or denied which "+
			"docs actions; on a play, whose plays:run is allowed or denied. Needs "+
			"permissions:write on the doc, or plays:write on the play. Returns each user's id, login, and name with "+
			"their allow and deny sets; change them with permission_overwrite_update. With user_id it returns instead "+
			"that person's effective answer for each action the doc's, play's, or project's rules may set, with "+
			"allowed and a source naming the rule or role that decided it; your own needs only opening the entity, "+
			"anyone else's its manage permission (members:write for a project).",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accessGrantListIn) (any, error) {
			if in.UserID != "" {
				return explain(ctx, s, in)
			}
			grants, err := listGrants(ctx, s, in)
			if err != nil {
				return nil, err
			}
			accounts, err := s.accountsByID(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]grantResult, 0, len(grants))
			for _, g := range grants {
				r := grantResult{UserID: g.UserID, Allow: g.Allow, Deny: g.Deny}
				if u, ok := accounts[g.UserID]; ok {
					r.Login, r.Name = u.Login, u.Name
				}
				out = append(out, r)
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func explain(ctx context.Context, s *Service, in accessGrantListIn) (any, error) {
	out, err := s.Explain(ctx, actorIDFromCtx(ctx), in.ResourceType, in.ResourceID, in.UserID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func listGrants(ctx context.Context, s *Service, in accessGrantListIn) ([]*Overwrite, error) {
	if in.ResourceType == resourceTypeDoc {
		return s.ListGrants(ctx, actorIDFromCtx(ctx), in.ResourceID)
	}
	if in.ResourceType == resourceTypePlay {
		return s.ListPlayGrants(ctx, actorIDFromCtx(ctx), in.ResourceID)
	}
	return nil, unknownResourceType(in.ResourceType)
}

func accessGrantUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("permission_overwrite_update", "Update permission overwrites",
		"Sets permission actions to allow, fallback, or deny for every listed user on every listed doc or play, in "+
			"one operation: allow and deny hold here whatever the user's role says, fallback leaves it to the role. "+
			"Only the named actions change. Needs permissions:write on each doc, or plays:write on each play, and "+
			"you can't allow, or lift a deny on, an action you don't hold there; adding a deny is never limited. "+
			"Changes nothing unless every check passes; permission_overwrite_list shows the result.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in accessGrantUpdateIn) (any, error) {
			if err := setGrants(ctx, s, in); err != nil {
				return nil, err
			}
			return in, nil
		})
}

func setGrants(ctx context.Context, s *Service, in accessGrantUpdateIn) error {
	actions, err := parseActions(in.Actions)
	if err != nil {
		return fmt.Errorf("%w; a permission is <domain>:<action>, for example docs:read", err)
	}
	state, err := in.state()
	if err != nil {
		return err
	}
	if in.ResourceType == resourceTypeDoc {
		return s.SetGrants(ctx, actorIDFromCtx(ctx), in.ResourceIDs, in.UserIDs, actions, state)
	}
	if in.ResourceType == resourceTypePlay {
		return s.SetPlayGrants(ctx, actorIDFromCtx(ctx), in.ResourceIDs, in.UserIDs, actions, state)
	}
	return unknownResourceType(in.ResourceType)
}

func unknownResourceType(t string) error {
	return fmt.Errorf("%w: resource_type %q is not doc or play", apperrs.ErrInvalid, t)
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

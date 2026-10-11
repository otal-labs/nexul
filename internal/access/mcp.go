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
	ResourceType string `json:"resource_type" jsonschema:"doc, play, or project."`
	ResourceID   string `json:"resource_id" jsonschema:"The doc's, play's, or project's id."`
	UserID       string `json:"user_id,omitempty" jsonschema:"An account id, yours or another's, as account_list shows them; returns that person's effective answers instead of the rules."`
	mcptool.PageArgs
}

type accessGrantUpdateIn struct {
	ResourceType string   `json:"resource_type" jsonschema:"doc, play, or project."`
	ResourceIDs  []string `json:"resource_ids" jsonschema:"The docs', plays', or projects' ids; every one gets the same change."`
	UserIDs      []string `json:"user_ids,omitempty" jsonschema:"The people the change applies to, by account id as account_list shows them. Send user_ids, role_ids, or both."`
	RoleIDs      []string `json:"role_ids,omitempty" jsonschema:"The roles the change applies to, each a role of the resource's workspace as workspace_list with the workspace's id lists them; the rule reaches everyone holding it."`
	Actions      []string `json:"actions" jsonschema:"Permissions to set, each <domain>:<action>, for example [\"docs:read\", \"docs:write\"]. On a play only plays:run; on a project projects:open (In this project) and the project-area actions."`
	State        string   `json:"state,omitempty" jsonschema:"allow, fallback, or deny: allow and deny hold the actions here whatever the roles say, fallback removes this rule's say so the answer comes from further out. Takes precedence over grant."`
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

// grantResult is one rule on the resource that was asked for, naming a person or a role.
type grantResult struct {
	Target   string          `json:"target"`
	UserID   string          `json:"user_id,omitempty"`
	Login    string          `json:"login,omitempty"`
	Name     string          `json:"name,omitempty"`
	RoleID   string          `json:"role_id,omitempty"`
	RoleName string          `json:"role_name,omitempty"`
	Allow    permissions.Set `json:"allow"`
	Deny     permissions.Set `json:"deny"`
}

func accessGrantListTool(s *Service) mcptool.Tool {
	return mcptool.New("permission_overwrite_list", "List permission overwrites",
		"Lists the rules on one doc, play, or project, each naming a person or a role (target person or role, with "+
			"user_id, login and name, or role_id and role_name) and the actions it allows and denies. A project's rules "+
			"say who is in it (projects:open) and what they hold there beyond their roles; a play's say whose plays:run "+
			"is allowed or denied. Needs permissions:write on the doc, plays:write on the play, or projects:access on "+
			"the project; a project you are not in is not found. Change them with permission_overwrite_update. With "+
			"user_id it returns instead that person's effective answer for each action the rules may set, with "+
			"allowed and a source naming the rule or role that decided it; your own needs only opening the entity, "+
			"anyone else's its manage permission.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accessGrantListIn) (any, error) {
			if in.UserID != "" {
				return explain(ctx, s, in)
			}
			rules, err := s.ListRules(ctx, actorIDFromCtx(ctx), in.ResourceType, in.ResourceID)
			if err != nil {
				return nil, err
			}
			accounts, err := s.accountsByID(ctx)
			if err != nil {
				return nil, err
			}
			names := s.namer(ctx)
			out := make([]grantResult, 0, len(rules))
			for _, g := range rules {
				out = append(out, toGrantResult(g, accounts, names))
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func toGrantResult(g *Overwrite, accounts map[string]*User, names *namer) grantResult {
	if g.Target == TargetRole {
		return grantResult{Target: TargetRole, RoleID: g.RoleID, RoleName: names.name(TargetRole, g.RoleID), Allow: g.Allow, Deny: g.Deny}
	}
	r := grantResult{Target: TargetPerson, UserID: g.UserID, Allow: g.Allow, Deny: g.Deny}
	if u, ok := accounts[g.UserID]; ok {
		r.Login, r.Name = u.Login, u.Name
	}
	return r
}

func explain(ctx context.Context, s *Service, in accessGrantListIn) (any, error) {
	out, err := s.Explain(ctx, actorIDFromCtx(ctx), in.ResourceType, in.ResourceID, in.UserID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func accessGrantUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("permission_overwrite_update", "Update permission overwrites",
		"Sets permission actions to allow, fallback, or deny for every listed person and role on every listed doc, "+
			"play, or project, in one operation: allow and deny hold here whatever the roles say, fallback leaves it "+
			"to further out, and between one person's roles Allow wins while a rule naming the person beats them. "+
			"Listing a role or a person on a project is projects:open allow; projects:open deny keeps a person out "+
			"whatever their roles. Only the named actions change. Needs permissions:write on each doc, plays:write on "+
			"each play, or projects:access on each project, and you can't allow, or lift a deny on, an action you "+
			"don't hold there; adding a deny is never limited. Changes nothing unless every check passes; "+
			"permission_overwrite_list shows the result.",
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
	return s.SetRules(ctx, actorIDFromCtx(ctx), in.ResourceType, in.ResourceIDs, Targets{UserIDs: in.UserIDs, RoleIDs: in.RoleIDs}, actions, state)
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

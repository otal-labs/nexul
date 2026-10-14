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
	ResourceType string `json:"resource_type" jsonschema:"doc, folder, play, project, channel, stack, computer, machine, or instance_area."`
	ResourceID   string `json:"resource_id" jsonschema:"The doc's, folder's, play's, project's, channel's, stack's, computer's, or machine's id, or an instance area's key: instance, sign_in_providers, dns, templates, accounts, runners, topology, automations, integrations, audit, or connector:<id> such as connector:github. A project's folders are project_get's doc_folders, its stacks stack_list's."`
	UserID       string `json:"user_id,omitempty" jsonschema:"An account id, yours or another's, as account_list shows them; returns that person's effective answers instead of the rules."`
	mcptool.PageArgs
}

type accessGrantUpdateIn struct {
	ResourceType string   `json:"resource_type" jsonschema:"doc, folder, play, project, channel, stack, computer, machine, or instance_area."`
	ResourceIDs  []string `json:"resource_ids" jsonschema:"The docs', folders', plays', projects', channels', stacks', computers', or machines' ids, or instance areas' keys as permission_overwrite_list takes them; every one gets the same change."`
	UserIDs      []string `json:"user_ids,omitempty" jsonschema:"The people the change applies to, by account id as account_list shows them. Send user_ids, role_ids, or both."`
	RoleIDs      []string `json:"role_ids,omitempty" jsonschema:"The roles the change applies to, each a role of the resource's workspace (on a machine or an instance area, of any workspace) as workspace_list with the workspace's id lists them; the rule reaches everyone holding it."`
	Actions      []string `json:"actions" jsonschema:"Permissions to set, each <domain>:<action>, for example [\"docs:read\", \"docs:write\"]. On a folder the docs actions and permissions:write; on a stack the stacks and deploys actions; on a play plays:read (See this play), plays:run, plays:write and plays:delete; on a project projects:open (In this project) and the project-area actions; on a channel channels:read and channels:post; on a computer computers:see, computers:run_agents and computers:run_commands, either run allowing see too; on a machine machines:see, machines:deploy, machines:write (Manage) and machines:delete; on an instance area its own actions, such as dns:read, dns:write, dns:delete on dns."`
	State        string   `json:"state,omitempty" jsonschema:"allow, fallback, or deny: allow and deny hold the actions here whatever the roles say, fallback removes this rule's say so the answer comes from further out. Takes precedence over grant."`
	Grant        *bool    `json:"grant,omitempty" jsonschema:"Without state: true grants the actions (on a play, lifts a deny); false revokes a grant (on a play, denies the actions)."`
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
		"Lists the rules on one doc, folder, play, project, channel, stack, computer, machine, or instance area, each naming a person or a role (target person or role, with "+
			"user_id, login and name, or role_id and role_name) and the actions it allows and denies. A project's rules "+
			"say who is in it (projects:open) and what they hold there beyond their roles; a folder's reach every doc in it, "+
			"a doc's own rules beating them; a stack's reach its deploys and branch deployments; a play's say who may see (plays:read), run, edit and delete it, and a play "+
			"someone can't see is gone for them; a channel's whose channels:read is denied, which hides it from them, and who posts there "+
			"(channels:post); a machine's who sees it (machines:see) and may deploy to, manage and delete it beyond their roles, a machine someone can't see being gone for them; a computer's whom "+
			"its owner shares it with, by name; an instance "+
			"area's who holds its actions beyond their roles. Needs permissions:write on the doc or folder, stacks:write on the stack, plays:write on the "+
			"play, projects:access on the project, channels:write on a channel you read, owning the computer, Manage on a machine you see, or being the "+
			"instance Owner for an instance area; a project you are not in, a play or stack you can't see, or a computer you don't own, is not found. Change them with permission_overwrite_update. With "+
			"user_id it returns instead that person's effective answer for each action the rules may set, with "+
			"allowed and a source naming the rule or role that decided it; your own needs only opening the entity, "+
			"anyone else's its manage permission, and on a computer shared with you only your own.",
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
			"folder, play, project, channel, stack, computer, machine, or instance area, in one operation: allow and deny hold here whatever the roles say, fallback leaves it "+
			"to further out, and between one person's roles Allow wins while a rule naming the person beats them. "+
			"Listing a role or a person on a project is projects:open allow; projects:open deny keeps a person out "+
			"whatever their roles. A folder's rules reach every doc in it, between its project's and the doc's own, and "+
			"docs:read deny on a folder hides its docs. A stack's rules reach its deploys and branch deployments, and stacks:read "+
			"deny hides it, deploys included. channels:read deny hides a channel from a role or a person, channels:post deny "+
			"keeps them from posting there while they still read it, and plays:read deny hides a play from its lists, run "+
			"menus, auto plays and play_list. Sharing one of your "+
			"computers is computers:see, computers:run_agents or computers:run_commands allow for people who share a "+
			"workspace with you, never a role; fallback takes a person off it and deny is refused, and only the "+
			"computer's owner writes its rules. A machine's rules give a role of any workspace or a person machines:see, which only a rule gives and without which the machine is not found, and machines:deploy, machines:write and machines:delete beyond their roles. An instance area's rule "+
			"hands out that area alone, such as dns:write on dns, and never denies an Owner. Only the named actions "+
			"change. Needs permissions:write on each doc or folder, stacks:write on each stack, plays:write on each play, projects:access on each project, "+
			"channels:write on each channel, Manage on each machine, or being the instance Owner for an instance area, and you can't allow, or lift a deny on, an action you "+
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

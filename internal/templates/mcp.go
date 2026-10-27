package templates

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type locationIn struct {
	Scope       string `json:"scope" jsonschema:"The layer: instance (every kind; the only layer of agent_prompt and role), workspace (interview, mention_chip, play_instructions), or project (ticket_body)."`
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"The workspace's id, when scope is workspace."`
	ProjectID   string `json:"project_id,omitempty" jsonschema:"The project's id, when scope is project."`
}

func (l locationIn) location() Location {
	return Location{Scope: Scope(l.Scope), WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID}
}

type templateGetIn struct {
	Kind string `json:"kind" jsonschema:"interview, mention_chip, play_instructions, ticket_body, agent_prompt (the Intro and Footer of every full agent turn prompt), or role (a Template role every new workspace copies)."`
	Key  string `json:"key,omitempty" jsonschema:"A built-in play's key (fix-with-ai, to-tickets-via-ai, interview, test-with-ai, interview-draft, clarify, update-doc, audit), a ticket type's name (task, bug, feature at the instance), intro or footer for agent_prompt, or a role template's key (client, junior_developer, senior_developer, devops, administrator, or the generated key of one the instance added). Empty for interview and mention_chip."`
	locationIn
}

type templateUpdateIn struct {
	Kind string `json:"kind" jsonschema:"interview, mention_chip, play_instructions, ticket_body, agent_prompt (the Intro and Footer of every full agent turn prompt), or role (a Template role every new workspace copies)."`
	Key  string `json:"key,omitempty" jsonschema:"A built-in play's key (fix-with-ai, to-tickets-via-ai, interview, test-with-ai, interview-draft, clarify, update-doc, audit), a ticket type's name (task, bug, feature at the instance), intro or footer for agent_prompt, or a role template's key (client, junior_developer, senior_developer, devops, administrator, or the generated key of one the instance added). Empty for interview and mention_chip; empty for role with a body adds a new template role."`
	locationIn
	Body      *string     `json:"body,omitempty" jsonschema:"The new text: markdown, except mention_chip, which uses {ticket.Field} tokens such as {ticket.Ticket} {ticket.Status}, and role, which is JSON with name, permissions, welcome_steps and show_locked_actions. The interview is one ## heading per question, a hint under it, then - single-choice or - [ ] multi-select options, at most 32,000 characters."`
	Reset     bool        `json:"reset,omitempty" jsonschema:"true returns this place to its default: the instance to the code default, a workspace or project to the instance's current text. A role template the instance added has no default; use delete."`
	CloneFrom *locationIn `json:"clone_from,omitempty" jsonschema:"Copy the same template's text from this place over this one."`
	Delete    bool        `json:"delete,omitempty" jsonschema:"true deletes a role template the instance added, at scope instance; the five that ship can only be reset. Workspaces already holding a copy keep it."`
}

// MCPTools returns the template tools; the domains' own tools keep editing the copies they hold.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{templateGetTool(s), templateUpdateTool(s)}
}

func templateGetTool(s *Service) mcptool.Tool {
	return mcptool.New("template_get", "Get template",
		"Returns one template's text at one layer, with default_body (what a reset there gives) and edited (false while it shows that default); the interview also returns its parsed questions. "+
			"Templates resolve code default, then instance, then workspace or project: an unedited workspace's interview and mention_chip follow the instance live, "+
			"while play_instructions and ticket_body are copied into each new workspace or project and never rewritten afterwards. "+
			"agent_prompt and role live only at the instance: an empty agent_prompt body leaves its part out of the prompt, and a role body is JSON each new workspace copies into a role of its own (added is true for one the instance added). "+
			"Use scope instance for the defaults every new workspace and project starts from; change any layer with template_update.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in templateGetIn) (any, error) {
			return s.Get(ctx, in.Kind, in.Key, in.location())
		})
}

func templateUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("template_update", "Update template",
		"Changes one template at one layer in one of four ways: body replaces its text, reset returns it to its default, clone_from copies the same template's text from another layer over it, and delete removes a role template the instance added. "+
			"Writing the instance changes what new workspaces and projects start from, and what every unedited workspace's interview and mention chip show; "+
			"existing play, ticket type and role copies change only where written. "+
			"A role body with no key adds a template role under a generated key. "+
			"A clone matches plays by key and ticket types by name, and fails when the target has no match. "+
			"Returns the template as it now stands, or {kind, key, deleted} after a delete. "+
			"play_update, project_update and role_update still edit a play's, ticket type's or workspace role's own fields.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in templateUpdateIn) (any, error) {
			at := in.location()
			changes := 0
			for _, set := range []bool{in.Body != nil, in.Reset, in.CloneFrom != nil, in.Delete} {
				if set {
					changes++
				}
			}
			switch {
			case changes > 1:
				return nil, fmt.Errorf("%w: pass one of body, reset, clone_from, or delete", apperrs.ErrInvalid)
			case in.Body != nil:
				return s.Update(ctx, in.Kind, in.Key, at, *in.Body)
			case in.Reset:
				return s.Reset(ctx, in.Kind, in.Key, at)
			case in.CloneFrom != nil:
				return s.Clone(ctx, in.Kind, in.Key, in.CloneFrom.location(), at)
			case in.Delete && at.Scope != ScopeInstance:
				return nil, fmt.Errorf("%w: only the instance holds templates to delete; use scope instance", apperrs.ErrInvalid)
			case in.Delete:
				return deleted{Kind: in.Kind, Key: in.Key, Deleted: true}, s.Delete(ctx, in.Kind, in.Key)
			}
			return s.Get(ctx, in.Kind, in.Key, at)
		})
}

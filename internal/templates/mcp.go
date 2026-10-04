package templates

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type locationIn struct {
	Scope       string `json:"scope" jsonschema:"The layer: instance (every kind; the only layer of agent_prompt), workspace (interview, mention_chip, play_instructions), or project (ticket_body)."`
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"The workspace's id, when scope is workspace."`
	ProjectID   string `json:"project_id,omitempty" jsonschema:"The project's id, when scope is project."`
}

func (l locationIn) location() Location {
	return Location{Scope: Scope(l.Scope), WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID}
}

type templateGetIn struct {
	Kind string `json:"kind" jsonschema:"The template kind: interview, mention_chip, play_instructions, ticket_body, or agent_prompt (the Intro and Footer every full agent turn prompt opens and closes with)."`
	Key  string `json:"key,omitempty" jsonschema:"Which template of the kind: a built-in play's key (fix-with-ai, to-tickets-via-ai, interview, test-with-ai, clarify) or a ticket type's name (task, bug, feature at the instance), or intro or footer for agent_prompt. Empty for interview and mention_chip."`
	locationIn
}

type templateUpdateIn struct {
	Kind string `json:"kind" jsonschema:"The template kind: interview, mention_chip, play_instructions, ticket_body, or agent_prompt (the Intro and Footer every full agent turn prompt opens and closes with)."`
	Key  string `json:"key,omitempty" jsonschema:"Which template of the kind: a built-in play's key (fix-with-ai, to-tickets-via-ai, interview, test-with-ai, clarify) or a ticket type's name (task, bug, feature at the instance), or intro or footer for agent_prompt. Empty for interview and mention_chip."`
	locationIn
	Body      *string     `json:"body,omitempty" jsonschema:"The new text, markdown except for mention_chip, which uses {ticket.Field} tokens such as {ticket.Ticket} {ticket.Status}. The interview is one ## heading per question, a hint under it, then - single-choice or - [ ] multi-select options, at most 32,000 characters."`
	Reset     bool        `json:"reset,omitempty" jsonschema:"true returns this place to its default: the instance to the code default, a workspace or project to the instance's current text."`
	CloneFrom *locationIn `json:"clone_from,omitempty" jsonschema:"Copy the same template's text from this place over this one, matching plays by key and ticket types by name."`
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
			"agent_prompt lives only at the instance, and an empty body leaves its part out of the prompt. "+
			"Use scope instance for the defaults every new workspace and project starts from; change any layer with template_update.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in templateGetIn) (any, error) {
			return s.Get(ctx, in.Kind, in.Key, in.location())
		})
}

func templateUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("template_update", "Update template",
		"Changes one template at one layer in one of three ways: body replaces its text, reset returns it to its default, and clone_from copies the same template's text from another layer, workspace, or project over it. "+
			"Writing the instance needs templates:write and changes what new workspaces and projects start from, and what every unedited workspace's interview and mention chip show; "+
			"a workspace or project needs that place's own write permission, and existing play and ticket type copies change only where written. "+
			"A clone fails when the target has no matching play or ticket type. "+
			"Returns the template as it now stands at the target; with no change requested it returns it unchanged. "+
			"play_update and project_update still edit a play's or ticket type's other fields.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in templateUpdateIn) (any, error) {
			at := in.location()
			changes := 0
			for _, set := range []bool{in.Body != nil, in.Reset, in.CloneFrom != nil} {
				if set {
					changes++
				}
			}
			switch {
			case changes > 1:
				return nil, fmt.Errorf("%w: pass one of body, reset, or clone_from", apperrs.ErrInvalid)
			case in.Body != nil:
				return s.Update(ctx, in.Kind, in.Key, at, *in.Body)
			case in.Reset:
				return s.Reset(ctx, in.Kind, in.Key, at)
			case in.CloneFrom != nil:
				return s.Clone(ctx, in.Kind, in.Key, in.CloneFrom.location(), at)
			}
			return s.Get(ctx, in.Kind, in.Key, at)
		})
}

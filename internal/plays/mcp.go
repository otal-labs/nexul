package plays

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// playResult is a play as the model reads it; audit timestamps are left out.
type playResult struct {
	ID                 string   `json:"id"`
	WorkspaceID        string   `json:"workspace_id"`
	Label              string   `json:"label"`
	Type               Type     `json:"type"`
	Description        string   `json:"description"`
	Instructions       string   `json:"instructions"`
	Enabled            bool     `json:"enabled"`
	ShowWhenStage      *Stage   `json:"show_when_stage,omitempty"`
	ExcludedProjectIDs []string `json:"excluded_project_ids"`
}

func toPlayResult(p *Play) playResult {
	return playResult{
		ID: p.ID, WorkspaceID: p.WorkspaceID, Label: p.Label, Type: p.Type, Description: p.Description,
		Instructions: p.Instructions, Enabled: p.Enabled, ShowWhenStage: p.ShowWhenStage, ExcludedProjectIDs: p.ExcludedProjectIDs,
	}
}

type playListIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace's id (a UUID); project_list shows it on every project."`
	Type        Type   `json:"type,omitempty" jsonschema:"ticket, doc, or interview. When set, lists only the plays the caller may run on such a target: enabled, not excluded from project_id, and for ticket plays showing in stage."`
	ProjectID   string `json:"project_id,omitempty" jsonschema:"With type: the target's project id, so plays excluded from that project are left out."`
	Stage       Stage  `json:"stage,omitempty" jsonschema:"With type ticket: the ticket's board stage (backlog, progress, review, testing, or done); only plays showing in that stage are listed."`
	mcptool.PageArgs
}

type playCreateIn struct {
	WorkspaceID        string   `json:"workspace_id" jsonschema:"The workspace's id (a UUID); project_list shows it on every project."`
	Label              string   `json:"label" jsonschema:"The button text people press, for example Fix with AI."`
	Type               Type     `json:"type" jsonschema:"What the play runs on: ticket, doc, or interview. Fixed once created."`
	Description        string   `json:"description,omitempty" jsonschema:"One line saying what the play does, shown under the button."`
	Instructions       string   `json:"instructions,omitempty" jsonschema:"The base instructions the Agent gets on every run, as markdown."`
	Enabled            bool     `json:"enabled,omitempty" jsonschema:"Whether the play shows and can run. Defaults to false."`
	ShowWhenStage      Stage    `json:"show_when_stage,omitempty" jsonschema:"Required for a ticket play and refused for any other: the board stage (backlog, progress, review, testing, or done) whose tickets show the button."`
	ExcludedProjectIDs []string `json:"excluded_project_ids,omitempty" jsonschema:"Ids of projects where the play never shows."`
}

type playUpdateIn struct {
	WorkspaceID        string    `json:"workspace_id" jsonschema:"The workspace's id (a UUID); project_list shows it on every project."`
	ID                 string    `json:"id" jsonschema:"The play's id, from play_list."`
	Label              *string   `json:"label,omitempty" jsonschema:"New button text; omit to keep it."`
	Description        *string   `json:"description,omitempty" jsonschema:"New one-line description; omit to keep it, an empty string clears it."`
	Instructions       *string   `json:"instructions,omitempty" jsonschema:"New base instructions as markdown, replacing the old ones whole; omit to keep them."`
	Enabled            *bool     `json:"enabled,omitempty" jsonschema:"true shows the play and lets it run, false hides it; omit to keep it."`
	ShowWhenStage      *Stage    `json:"show_when_stage,omitempty" jsonschema:"Ticket plays only: the board stage (backlog, progress, review, testing, or done) whose tickets show the button; omit to keep it."`
	ExcludedProjectIDs *[]string `json:"excluded_project_ids,omitempty" jsonschema:"Ids of projects where the play never shows, replacing the whole list; an empty list clears it, omit to keep it."`
}

type playDeleteIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace's id (a UUID); project_list shows it on every project."`
	ID          string `json:"id" jsonschema:"The play's id, from play_list."`
}

// MCPTools returns the play definition tools; running a play and its trails live in RunMCPTools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("play_list", "List plays",
			"Lists a workspace's plays sorted by label, each with its type, instructions, enabled switch, stage, "+
				"and excluded projects. Without type it lists every definition and needs plays:read; with type (and "+
				"project_id, plus stage for ticket plays) it lists only the plays the caller may run on that target, "+
				"which is what to check before play_run. Paged, 50 per page by default.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in playListIn) (any, error) {
				list, err := listPlays(ctx, s, in)
				if err != nil {
					return nil, err
				}
				out := make([]playResult, 0, len(list))
				for _, p := range list {
					out = append(out, toPlayResult(p))
				}
				return mcptool.Paginate(out, in.PageArgs), nil
			}),
		mcptool.New("play_create", "Create play",
			"Creates a play: a button that starts a pre-configured Agent turn on a ticket, a doc, or a project's "+
				"interview. A ticket play needs show_when_stage and no other type takes one; a new play is disabled "+
				"unless enabled is true. Returns the created play; change it later with play_update, and start it "+
				"with play_run. Needs plays:write.",
			mcptool.Hints{Additive: true, Local: true},
			func(ctx context.Context, in playCreateIn) (any, error) {
				p, err := s.Create(ctx, in.WorkspaceID, CreateInput{
					Label: in.Label, Type: in.Type, Description: in.Description, Instructions: in.Instructions,
					Enabled: in.Enabled, ShowWhenStage: optionalStage(in.ShowWhenStage), ExcludedProjectIDs: in.ExcludedProjectIDs,
				})
				if err != nil {
					return nil, err
				}
				return toPlayResult(p), nil
			}),
		mcptool.New("play_update", "Update play",
			"Changes a play's label, description, instructions, enabled switch, stage, or excluded projects. Only "+
				"the fields you pass change; the rest keep their values, and a play's type never changes. Use enabled "+
				"false to hide a play without deleting it, and play_delete to remove it. Returns the updated play. "+
				"Needs plays:read and plays:write.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in playUpdateIn) (any, error) {
				p, err := s.Get(ctx, in.WorkspaceID, in.ID)
				if err != nil {
					return nil, playNotFoundHint(err)
				}
				updated, err := s.Update(ctx, in.WorkspaceID, in.ID, overlayPlay(p, in))
				if err != nil {
					return nil, err
				}
				return toPlayResult(updated), nil
			}),
		mcptool.New("play_delete", "Delete play",
			"Deletes a play from its workspace. Trails of its past runs stay readable through trail_list. To hide "+
				"a play without losing it, use play_update with enabled false instead. Needs plays:delete.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in playDeleteIn) (any, error) {
				if err := s.Delete(ctx, in.WorkspaceID, in.ID); err != nil {
					return nil, playNotFoundHint(err)
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}

func listPlays(ctx context.Context, s *Service, in playListIn) ([]*Play, error) {
	if in.Type == "" {
		return s.List(ctx, in.WorkspaceID)
	}
	return s.ListApplicable(ctx, in.WorkspaceID, actorID(ctx), in.ProjectID, in.Type, optionalStage(in.Stage))
}

// overlayPlay keeps every field the call left out, so a rename never disables the play or wipes its instructions.
func overlayPlay(p *Play, in playUpdateIn) UpdateInput {
	out := UpdateInput{
		Label: p.Label, Description: p.Description, Instructions: p.Instructions, Enabled: p.Enabled,
		ShowWhenStage: p.ShowWhenStage, ExcludedProjectIDs: p.ExcludedProjectIDs,
	}
	if in.Label != nil {
		out.Label = *in.Label
	}
	if in.Description != nil {
		out.Description = *in.Description
	}
	if in.Instructions != nil {
		out.Instructions = *in.Instructions
	}
	if in.Enabled != nil {
		out.Enabled = *in.Enabled
	}
	if in.ShowWhenStage != nil {
		out.ShowWhenStage = in.ShowWhenStage
	}
	if in.ExcludedProjectIDs != nil {
		out.ExcludedProjectIDs = *in.ExcludedProjectIDs
	}
	return out
}

func optionalStage(s Stage) *Stage {
	if s == "" {
		return nil
	}
	return &s
}

func playNotFoundHint(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; play_list shows the workspace's plays", err)
	}
	return err
}

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
	// AutoPlays is left out for a caller without autoplays:read, and on a list filtered by type.
	AutoPlays []autoPlayResult `json:"auto_plays,omitzero"`
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
	Label              string   `json:"label" jsonschema:"The button text people press, for example Fix with AI; unique in the workspace, ignoring case."`
	Type               Type     `json:"type" jsonschema:"What the play runs on: ticket, doc, or interview. Fixed once created."`
	Description        string   `json:"description,omitempty" jsonschema:"One line saying what the play does, shown under the button."`
	Instructions       string   `json:"instructions,omitempty" jsonschema:"The base instructions the Agent gets on every run, as markdown."`
	Enabled            bool     `json:"enabled,omitempty" jsonschema:"Whether the play shows and can run. Defaults to false."`
	ShowWhenStage      Stage    `json:"show_when_stage,omitempty" jsonschema:"Required for a ticket play and refused for any other: the board stage (backlog, progress, review, testing, or done) whose tickets show the button."`
	ExcludedProjectIDs []string `json:"excluded_project_ids,omitempty" jsonschema:"Ids of projects where the play never shows, from project_list."`
}

type playUpdateIn struct {
	WorkspaceID        string             `json:"workspace_id" jsonschema:"The workspace's id (a UUID); project_list shows it on every project."`
	ID                 string             `json:"id" jsonschema:"The play's id, from play_list."`
	Label              *string            `json:"label,omitempty" jsonschema:"New button text, unique in the workspace ignoring case; omit to keep it."`
	Description        *string            `json:"description,omitempty" jsonschema:"New one-line description; omit to keep it, an empty string clears it."`
	Instructions       *string            `json:"instructions,omitempty" jsonschema:"New base instructions as markdown, replacing the old ones whole; omit to keep them."`
	Enabled            *bool              `json:"enabled,omitempty" jsonschema:"true shows the play and lets it run, false hides it; omit to keep it."`
	ShowWhenStage      *Stage             `json:"show_when_stage,omitempty" jsonschema:"Ticket plays only: the board stage (backlog, progress, review, testing, or done) whose tickets show the button; omit to keep it."`
	ExcludedProjectIDs *[]string          `json:"excluded_project_ids,omitempty" jsonschema:"Ids of projects where the play never shows, replacing the whole list; an empty list clears it, omit to keep it."`
	AddAutoPlays       []autoPlayIn       `json:"add_auto_plays,omitempty" jsonschema:"Auto plays to add to the play, each starting it by itself when its moment matches. A new one starts switched off; turn it on with update_auto_plays. Needs autoplays:write."`
	UpdateAutoPlays    []autoPlayChangeIn `json:"update_auto_plays,omitempty" jsonschema:"Changes to the play's auto plays by id, from play_list; only the fields passed change. Needs autoplays:write."`
	RemoveAutoPlays    []string           `json:"remove_auto_plays,omitempty" jsonschema:"Ids of auto plays to delete from the play, from play_list. Needs autoplays:delete."`
}

func (in playUpdateIn) changesPlay() bool {
	return in.Label != nil || in.Description != nil || in.Instructions != nil || in.Enabled != nil || in.ShowWhenStage != nil || in.ExcludedProjectIDs != nil
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
				"and excluded projects. Without type it lists every definition and needs plays:read, and a caller with "+
				"autoplays:read also gets each play's auto_plays: the moment that starts it by itself, its conditions "+
				"(all or any groups of rules, each a field, an op of is, is_not, set, or unset, and values), priority, "+
				"once_within_minutes limit, run_on, and enabled switch. With type (and "+
				"project_id, plus stage for ticket plays) it lists only the plays the caller may run on that target, "+
				"which is what to check before play_run. The seeded Decisions check play is listed like any other: its auto play, "+
				"off by default, runs it when a ticket enters done. Paged, 50 per page by default.",
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
				page := mcptool.Paginate(out, in.PageArgs)
				if in.Type != "" {
					return page, nil
				}
				if err := attachAutoPlays(ctx, s, in.WorkspaceID, page.Items); err != nil {
					return nil, err
				}
				return page, nil
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
			"Changes a play's label, description, instructions, enabled switch, stage, or excluded projects, and adds, "+
				"changes, or removes its auto plays. Only the fields you pass change; the rest keep their values, and a "+
				"play's type never changes. Use enabled false to hide a play without deleting it, and play_delete to remove it. "+
				"An auto play starts the play by itself on a moment: ticket.unblocked, ticket.entered_stage (with "+
				"moment_stage), ticket.created, ticket.developer_set, ticket.tester_set, or ticket.test_failed for a ticket "+
				"play, doc.created or doc.changed for a doc play, none for an interview play. Its conditions read a ticket's "+
				"type, project, stage, status, category, label, developer, tester, source_doc, linked_pr, or blocked, or a "+
				"doc's project or folder, with is and is_not over values or set and unset, at most 20 rules in all. "+
				"Play fields apply first, then add_auto_plays, update_auto_plays, and remove_auto_plays in that order, "+
				"stopping at the first failure with what already took effect. Returns the play, with its auto plays when "+
				"they changed. Needs plays:read, plays:write for play fields, autoplays:write to add or change auto plays, "+
				"and autoplays:delete to remove them. The decisions check is switched on or off through its play's auto play, "+
				"with update_auto_plays and enabled.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in playUpdateIn) (any, error) {
				return updatePlay(ctx, s, in)
			}),
		mcptool.New("play_delete", "Delete play",
			"Deletes a play from its workspace and returns its id with deleted true. Trails of its past runs stay "+
				"readable through trail_list. To hide a play without losing it, use play_update with enabled false "+
				"instead. Needs plays:delete.",
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
	if in.Type == "" && (in.ProjectID != "" || in.Stage != "") {
		return nil, fmt.Errorf("%w: project_id and stage narrow the list only together with type (ticket, doc, or interview)", apperrs.ErrInvalid)
	}
	if in.Type == "" {
		return s.List(ctx, in.WorkspaceID)
	}
	return s.ListApplicable(ctx, in.WorkspaceID, actorID(ctx), in.ProjectID, in.Type, optionalStage(in.Stage))
}

func updatePlay(ctx context.Context, s *Service, in playUpdateIn) (any, error) {
	p, err := s.Get(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, playNotFoundHint(err)
	}
	var applied []string
	if in.changesPlay() {
		if p, err = s.Update(ctx, in.WorkspaceID, in.ID, overlayPlay(p, in)); err != nil {
			return nil, err
		}
		applied = append(applied, "play fields")
	}
	if !in.changesAutoPlays() {
		return toPlayResult(p), nil
	}
	if err := changeAutoPlays(ctx, s, in.WorkspaceID, p.ID, in, applied); err != nil {
		return nil, err
	}
	out := []playResult{toPlayResult(p)}
	if err := attachAutoPlays(ctx, s, in.WorkspaceID, out); err != nil {
		return nil, err
	}
	return out[0], nil
}

// attachAutoPlays reads the page's auto plays in one batch; without autoplays:read the plays go out without them.
func attachAutoPlays(ctx context.Context, s *Service, workspaceID string, page []playResult) error {
	playIDs := make([]string, 0, len(page))
	for _, p := range page {
		playIDs = append(playIDs, p.ID)
	}
	byPlay, err := s.AutoPlaysByPlay(ctx, workspaceID, playIDs)
	if errors.Is(err, apperrs.ErrForbidden) {
		return nil
	}
	if err != nil {
		return err
	}
	for i := range page {
		if list, ok := byPlay[page[i].ID]; ok {
			page[i].AutoPlays = toAutoPlayResults(list)
		}
	}
	return nil
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

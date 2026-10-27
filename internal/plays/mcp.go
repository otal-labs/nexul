package plays

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	// CanRun is set on a listed play only: whether the caller may run it, from its own rules and their roles.
	CanRun *bool `json:"can_run,omitempty"`
	// AutoPlays is left out for a caller without autoplays:read, and on a list filtered by type.
	AutoPlays []autoPlayResult `json:"auto_plays,omitzero"`
	// Warnings carries what an update applied but may have broken, such as automations calling an old label.
	Warnings []string `json:"warnings,omitzero"`
}

func toPlayResult(p *Play) playResult {
	return playResult{
		ID: p.ID, WorkspaceID: p.WorkspaceID, Label: p.Label, Type: p.Type, Description: p.Description,
		Instructions: p.Instructions, Enabled: p.Enabled, ShowWhenStage: p.ShowWhenStage, ExcludedProjectIDs: p.ExcludedProjectIDs,
	}
}

type playListIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace's id, from workspace_list."`
	Type        Type   `json:"type,omitempty" jsonschema:"ticket, doc, interview, or stack. Lists only plays the caller may run on such a target: enabled, not excluded from project_id, and for ticket plays showing in stage."`
	ProjectID   string `json:"project_id,omitempty" jsonschema:"With type: the target's project id."`
	Stage       Stage  `json:"stage,omitempty" jsonschema:"With type ticket: the ticket's board stage (backlog, progress, review, testing, or done)."`
	mcptool.PageArgs
}

type playCreateIn struct {
	WorkspaceID        string   `json:"workspace_id" jsonschema:"The workspace's id, from workspace_list."`
	Label              string   `json:"label" jsonschema:"The button text; unique in the workspace, ignoring case."`
	Type               Type     `json:"type" jsonschema:"What the play runs on: ticket, doc, interview, or stack. Fixed once created."`
	Description        string   `json:"description,omitempty" jsonschema:"One line shown under the button."`
	Instructions       string   `json:"instructions,omitempty" jsonschema:"Base instructions for every run, as markdown."`
	Enabled            bool     `json:"enabled,omitempty" jsonschema:"Shows the play and lets it run. Defaults to false."`
	ShowWhenStage      Stage    `json:"show_when_stage,omitempty" jsonschema:"Required for a ticket play, refused for any other: the board stage (backlog, progress, review, testing, or done) whose tickets show the button."`
	ExcludedProjectIDs []string `json:"excluded_project_ids,omitempty" jsonschema:"Ids of projects where the play never shows, from project_list."`
}

type playUpdateIn struct {
	WorkspaceID        string             `json:"workspace_id" jsonschema:"The workspace's id, from workspace_list."`
	ID                 string             `json:"id" jsonschema:"The play's id, from play_list."`
	Label              *string            `json:"label,omitempty" jsonschema:"New button text, unique in the workspace ignoring case."`
	Description        *string            `json:"description,omitempty" jsonschema:"New one-line description; an empty string clears it."`
	Instructions       *string            `json:"instructions,omitempty" jsonschema:"New base instructions as markdown, replacing the old ones whole."`
	Enabled            *bool              `json:"enabled,omitempty" jsonschema:"true shows the play and lets it run, false hides it."`
	ShowWhenStage      *Stage             `json:"show_when_stage,omitempty" jsonschema:"Ticket plays only: the board stage (backlog, progress, review, testing, or done) whose tickets show the button."`
	ExcludedProjectIDs *[]string          `json:"excluded_project_ids,omitempty" jsonschema:"Ids of projects where the play never shows, replacing the whole list; an empty list clears it."`
	AddAutoPlays       []autoPlayIn       `json:"add_auto_plays,omitempty" jsonschema:"Auto plays to add, each starting the play by itself when its moment matches. A new one starts switched off; turn it on with update_auto_plays."`
	UpdateAutoPlays    []autoPlayChangeIn `json:"update_auto_plays,omitempty" jsonschema:"Changes by auto play id, from play_list; only the fields passed change."`
	RemoveAutoPlays    []string           `json:"remove_auto_plays,omitempty" jsonschema:"Ids of auto plays to delete, from play_list."`
}

func (in playUpdateIn) changesPlay() bool {
	return in.Label != nil || in.Description != nil || in.Instructions != nil || in.Enabled != nil || in.ShowWhenStage != nil || in.ExcludedProjectIDs != nil
}

type playDeleteIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace's id, from workspace_list."`
	ID          string `json:"id" jsonschema:"The play's id, from play_list."`
}

// MCPTools returns the play definition tools; running a play and its trails live in RunMCPTools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("play_list", "List plays",
			"Lists the plays of a workspace the caller can see, each with its type, "+
				"instructions, enabled switch, stage, excluded projects, and whether the caller may run it. Without type it "+
				"lists every definition, and a caller with autoplays:read also gets each play's auto_plays: moment, "+
				"conditions, priority, once_within_minutes, run_on, and enabled switch. With type (and "+
				"project_id, plus stage for ticket plays) it lists only the plays the caller may run on that target, "+
				"which is what to check before play_run.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in playListIn) (any, error) {
				list, err := listPlays(ctx, s, in)
				if err != nil {
					return nil, err
				}
				out := make([]playResult, 0, len(list))
				for _, p := range list {
					res := toPlayResult(&p.Play)
					res.CanRun = &p.CanRun
					out = append(out, res)
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
			"Creates a play: a button that starts a pre-configured Agent turn on a ticket, a doc, a project's "+
				"interview, or a stack. Returns the created play; change it later with play_update, and start it "+
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
			"Changes a play's fields and adds, changes, or removes its auto plays, which start the play by itself on a "+
				"moment (interview plays have none). A play's type never changes. Use enabled false to hide a play "+
				"without deleting it, and play_delete to remove it. Play fields apply first, then add_auto_plays, "+
				"update_auto_plays, and remove_auto_plays, stopping at the first failure with what already took effect. "+
				"Returns the play, with its auto plays when they changed. A new label still applies when automations "+
				"call the play by its old one with runPlay; warnings then names them, since they fail until their code "+
				"changes. Needs plays:write for its fields, autoplays:write to add or change auto plays, and "+
				"autoplays:delete to remove them. The decisions check is switched on or off through its play's auto "+
				"play, with update_auto_plays and enabled.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in playUpdateIn) (any, error) {
				return updatePlay(ctx, s, in)
			}),
		mcptool.New("play_delete", "Delete play",
			"Deletes a play from its workspace. Trails of its past runs stay readable through trail_list. "+
				"To hide a play without losing it, use play_update with enabled false instead. Needs plays:delete on the play.",
			mcptool.Hints{Idempotent: true, Local: true},
			func(ctx context.Context, in playDeleteIn) (any, error) {
				if err := s.Delete(ctx, in.WorkspaceID, in.ID); err != nil {
					return nil, playNotFoundHint(err)
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}

func listPlays(ctx context.Context, s *Service, in playListIn) ([]Listed, error) {
	if in.Type == "" && (in.ProjectID != "" || in.Stage != "") {
		return nil, fmt.Errorf("%w: project_id and stage narrow the list only together with type (ticket, doc, interview, or stack)", apperrs.ErrInvalid)
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
	var applied, warnings []string
	if in.changesPlay() {
		if warnings, err = renameWarnings(ctx, s, p, in.Label); err != nil {
			return nil, err
		}
		if p, err = s.Update(ctx, in.WorkspaceID, in.ID, overlayPlay(p, in)); err != nil {
			return nil, err
		}
		applied = append(applied, "play fields")
	}
	res := toPlayResult(p)
	res.Warnings = warnings
	if !in.changesAutoPlays() {
		return res, nil
	}
	if err := changeAutoPlays(ctx, s, in.WorkspaceID, p.ID, in, applied); err != nil {
		return nil, err
	}
	out := []playResult{res}
	if err := attachAutoPlays(ctx, s, in.WorkspaceID, out); err != nil {
		return nil, err
	}
	return out[0], nil
}

// renameWarnings warns, without refusing, when a new label leaves automations calling the play by its old one.
func renameWarnings(ctx context.Context, s *Service, p *Play, label *string) ([]string, error) {
	if label == nil {
		return nil, nil
	}
	impact, err := s.RenameImpact(ctx, p.WorkspaceID, p.ID, *label)
	if err != nil || impact.Count == 0 {
		return nil, err
	}
	names := ""
	if len(impact.Automations) > 0 {
		names = " (" + strings.Join(impact.Automations, ", ") + ")"
	}
	format := "%d automations%s still call this play by its old label %q with runPlay and fail until their code uses the new label."
	if impact.Count == 1 {
		format = "%d automation%s still calls this play by its old label %q with runPlay and fails until its code uses the new label."
	}
	return []string{fmt.Sprintf(format, impact.Count, names, p.Label)}, nil
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

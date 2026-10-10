package plays

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// autoPlayResult is an auto play as the model reads it; audit fields are left out.
type autoPlayResult struct {
	ID                string     `json:"id"`
	Enabled           bool       `json:"enabled"`
	Moment            Moment     `json:"moment"`
	MomentStage       *Stage     `json:"moment_stage,omitempty"`
	Conditions        Conditions `json:"conditions"`
	Priority          Priority   `json:"priority"`
	OnceWithinMinutes int        `json:"once_within_minutes"`
	RunOn             RunOn      `json:"run_on"`
}

func toAutoPlayResults(list []*AutoPlay) []autoPlayResult {
	out := make([]autoPlayResult, 0, len(list))
	for _, a := range list {
		out = append(out, autoPlayResult{
			ID: a.ID, Enabled: a.Enabled, Moment: a.Moment, MomentStage: a.MomentStage, Conditions: a.Conditions,
			Priority: a.Priority, OnceWithinMinutes: a.OnceWithinMinutes, RunOn: a.RunOn,
		})
	}
	return out
}

type autoPlayIn struct {
	Moment            Moment      `json:"moment" jsonschema:"When it fires. Ticket plays: ticket.unblocked, ticket.entered_stage, ticket.created, ticket.developer_set, ticket.tester_set, or ticket.test_failed. Doc plays: doc.created, or doc.changed once edits have settled."`
	MomentStage       Stage       `json:"moment_stage,omitempty" jsonschema:"Required with ticket.entered_stage and refused otherwise: backlog, progress, review, testing, or done."`
	Conditions        *Conditions `json:"conditions,omitempty" jsonschema:"What must also hold; omit to fire on every match of the moment."`
	Priority          *Priority   `json:"priority,omitempty" jsonschema:"The run's place in its person's queue; omit for normal."`
	OnceWithinMinutes int         `json:"once_within_minutes,omitempty" jsonschema:"At most one run per ticket or doc within this many minutes of the moment, up to 43200; 0 or omitted is no limit."`
	RunOn             RunOn       `json:"run_on,omitempty" jsonschema:"Whose computer runs it: developer (the default for ticket plays), tester, or causer (whoever caused the moment, the only choice for doc plays)."`
}

func (in autoPlayIn) input() AutoPlayInput {
	out := AutoPlayInput{Moment: in.Moment, MomentStage: optionalStage(in.MomentStage), OnceWithinMinutes: in.OnceWithinMinutes, RunOn: in.RunOn}
	if in.Conditions != nil {
		out.Conditions = *in.Conditions
	}
	if in.Priority != nil {
		out.Priority = *in.Priority
	}
	return out
}

type autoPlayChangeIn struct {
	ID                string      `json:"id" jsonschema:"The auto play's id, from play_list."`
	Enabled           *bool       `json:"enabled,omitempty" jsonschema:"true switches it on, false off; omit to keep it."`
	Moment            *Moment     `json:"moment,omitempty" jsonschema:"A new moment, from the same list as add_auto_plays; omit to keep it."`
	MomentStage       *Stage      `json:"moment_stage,omitempty" jsonschema:"A new stage for ticket.entered_stage; omit to keep it."`
	Conditions        *Conditions `json:"conditions,omitempty" jsonschema:"New conditions, replacing the old ones whole; omit to keep them."`
	Priority          *Priority   `json:"priority,omitempty" jsonschema:"A new priority, replacing the old one whole; omit to keep it."`
	OnceWithinMinutes *int        `json:"once_within_minutes,omitempty" jsonschema:"A new limit in minutes, 0 for none; omit to keep it."`
	RunOn             *RunOn      `json:"run_on,omitempty" jsonschema:"developer, tester, or causer; omit to keep it."`
}

// overlay keeps every field the change left out.
func (in autoPlayChangeIn) overlay(a *AutoPlay) (bool, AutoPlayInput) {
	enabled := a.Enabled
	out := AutoPlayInput{
		Moment: a.Moment, MomentStage: a.MomentStage, Conditions: a.Conditions, Priority: a.Priority,
		OnceWithinMinutes: a.OnceWithinMinutes, RunOn: a.RunOn,
	}
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	if in.Moment != nil {
		out.Moment = *in.Moment
	}
	if in.MomentStage != nil {
		out.MomentStage = in.MomentStage
	}
	if in.Conditions != nil {
		out.Conditions = *in.Conditions
	}
	if in.Priority != nil {
		out.Priority = *in.Priority
	}
	if in.OnceWithinMinutes != nil {
		out.OnceWithinMinutes = *in.OnceWithinMinutes
	}
	if in.RunOn != nil {
		out.RunOn = *in.RunOn
	}
	return enabled, out
}

// changeAutoPlays applies adds, then updates, then removes, stopping at the first failure with what already took effect.
func changeAutoPlays(ctx context.Context, s *Service, workspaceID, playID string, in playUpdateIn, applied []string) error {
	for _, add := range in.AddAutoPlays {
		a, err := s.CreateAutoPlay(ctx, workspaceID, playID, add.input())
		if err != nil {
			return stepErr(applied, "add_auto_plays", err)
		}
		applied = append(applied, "added auto play "+a.ID)
	}
	if len(in.UpdateAutoPlays) > 0 {
		current, err := s.ListAutoPlays(ctx, workspaceID, playID)
		if err != nil {
			return stepErr(applied, "update_auto_plays", err)
		}
		for _, change := range in.UpdateAutoPlays {
			a := findAutoPlay(current, change.ID)
			if a == nil {
				return stepErr(applied, "update_auto_plays", fmt.Errorf("%w: auto play %s is not one of this play's; play_list shows them", apperrs.ErrNotFound, change.ID))
			}
			enabled, next := change.overlay(a)
			if _, err := s.UpdateAutoPlay(ctx, workspaceID, playID, change.ID, enabled, next); err != nil {
				return stepErr(applied, "update_auto_plays", err)
			}
			applied = append(applied, "updated auto play "+change.ID)
		}
	}
	for _, id := range in.RemoveAutoPlays {
		if err := s.DeleteAutoPlay(ctx, workspaceID, playID, id); err != nil {
			return stepErr(applied, "remove_auto_plays", err)
		}
		applied = append(applied, "removed auto play "+id)
	}
	return nil
}

func findAutoPlay(list []*AutoPlay, id string) *AutoPlay {
	for _, a := range list {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func (in playUpdateIn) changesAutoPlays() bool {
	return len(in.AddAutoPlays) > 0 || len(in.UpdateAutoPlays) > 0 || len(in.RemoveAutoPlays) > 0
}

// stepErr says which changes already took effect when a later step failed, even when err is hidden.
func stepErr(applied []string, step string, err error) error {
	if len(applied) == 0 {
		return err
	}
	return &mcptool.PartialError{Applied: applied, Err: fmt.Errorf("%s: %w", step, err)}
}

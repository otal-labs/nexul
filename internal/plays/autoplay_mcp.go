package plays

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/labels"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// autoPlayResult is an auto play as the model reads it; audit fields are left out.
type autoPlayResult struct {
	ID                string              `json:"id"`
	Enabled           bool                `json:"enabled"`
	Moment            Moment              `json:"moment"`
	MomentStage       *Stage              `json:"moment_stage,omitempty"`
	Schedule          *Schedule           `json:"schedule,omitempty"`
	Conditions        Conditions          `json:"conditions"`
	Priority          Priority            `json:"priority"`
	OnceWithinMinutes int                 `json:"once_within_minutes"`
	RunOn             RunOn               `json:"run_on"`
	RunAs             string              `json:"run_as,omitempty"`
	ComputerRule      labels.ComputerRule `json:"computer_rule"`
	MemoryIDs         []string            `json:"memory_ids"`
	OneAtATime        bool                `json:"one_at_a_time,omitempty"`
	Resource          string              `json:"resource,omitempty"`
	// LastSweep is a schedule auto play's latest sweep and how its runs ended.
	LastSweep *Sweep `json:"last_sweep,omitempty"`
}

func toAutoPlayResults(list []*AutoPlay) []autoPlayResult {
	out := make([]autoPlayResult, 0, len(list))
	for _, a := range list {
		out = append(out, autoPlayResult{
			ID: a.ID, Enabled: a.Enabled, Moment: a.Moment, MomentStage: a.MomentStage, Schedule: a.Schedule, Conditions: a.Conditions,
			Priority: a.Priority, OnceWithinMinutes: a.OnceWithinMinutes, RunOn: a.RunOn,
			RunAs: a.RunAs, ComputerRule: a.ComputerRule, MemoryIDs: a.MemoryIDs, OneAtATime: a.OneAtATime, Resource: a.Resource,
		})
	}
	return out
}

type autoPlayIn struct {
	Moment            Moment               `json:"moment" jsonschema:"When it fires. Ticket plays: ticket.unblocked, ticket.entered_stage, ticket.created, ticket.developer_set, ticket.tester_set, or ticket.test_failed. Doc plays: doc.created, or doc.changed once edits have settled. Stack plays: stack.service_crashed, stack.service_recovered once the service has run steadily again, or stack.error_rate_crossed when 5xx answers go over the stack's error-rate rule. Ticket plays also: schedule, a cron sweep of every matching ticket."`
	Schedule          *Schedule            `json:"schedule,omitempty" jsonschema:"With schedule only."`
	MomentStage       Stage                `json:"moment_stage,omitempty" jsonschema:"Required with ticket.entered_stage, refused otherwise: backlog, progress, review, testing, or done."`
	Conditions        *Conditions          `json:"conditions,omitempty" jsonschema:"What must also hold; omit to fire on every match of the moment."`
	Priority          *Priority            `json:"priority,omitempty" jsonschema:"The run's place in its person's queue; omit for normal."`
	OnceWithinMinutes int                  `json:"once_within_minutes,omitempty" jsonschema:"At most one run per ticket or doc within this many minutes of the moment, up to 43200; 0 is no limit."`
	RunOn             RunOn                `json:"run_on,omitempty" jsonschema:"Who it runs as, on their computers: developer (default for ticket plays), tester, causer (whoever caused the moment, default for doc plays), or person (you, on your order; the only choice for stack plays)."`
	ComputerRule      *labels.ComputerRule `json:"computer_rule,omitempty" jsonschema:"Narrows the computers its runs land on; ids only with run_on person."`
	MemoryIDs         []string             `json:"memory_ids,omitempty" jsonschema:"Ids of memories each run reads beside the always-included ones, from memory_list, up to 50; a run reads only those in its ticket's, doc's, or stack's own project."`
	OneAtATime        bool                 `json:"one_at_a_time,omitempty" jsonschema:"One run at a time; with resource, across auto plays naming it."`
	Resource          string               `json:"resource,omitempty" jsonschema:"A shared name, such as windows-vm."`
}

func (in autoPlayIn) input() AutoPlayInput {
	out := AutoPlayInput{
		Moment: in.Moment, MomentStage: optionalStage(in.MomentStage), Schedule: in.Schedule, OnceWithinMinutes: in.OnceWithinMinutes, RunOn: in.RunOn,
		MemoryIDs: in.MemoryIDs, OneAtATime: in.OneAtATime, Resource: in.Resource,
	}
	out.ComputerRule = in.ComputerRule
	if in.Conditions != nil {
		out.Conditions = *in.Conditions
	}
	if in.Priority != nil {
		out.Priority = *in.Priority
	}
	return out
}

type autoPlayChangeIn struct {
	ID                string               `json:"id" jsonschema:"The auto play's id, from play_list."`
	Enabled           *bool                `json:"enabled,omitempty" jsonschema:"true switches it on, false off."`
	Moment            *Moment              `json:"moment,omitempty" jsonschema:"A new moment, from the same list as add_auto_plays."`
	MomentStage       *Stage               `json:"moment_stage,omitempty" jsonschema:"A new stage for ticket.entered_stage."`
	Schedule          *Schedule            `json:"schedule,omitempty" jsonschema:"Replaces the schedule whole."`
	Conditions        *Conditions          `json:"conditions,omitempty" jsonschema:"New conditions, replacing the old ones whole."`
	Priority          *Priority            `json:"priority,omitempty" jsonschema:"A new priority, replacing the old one whole."`
	OnceWithinMinutes *int                 `json:"once_within_minutes,omitempty" jsonschema:"A new limit in minutes, 0 for none."`
	RunOn             *RunOn               `json:"run_on,omitempty" jsonschema:"developer, tester, causer, or person."`
	RunAsMe           bool                 `json:"run_as_me,omitempty" jsonschema:"With run_on person: true makes you the person it runs as, on your order."`
	ComputerRule      *labels.ComputerRule `json:"computer_rule,omitempty" jsonschema:"Replaces the computer rule whole."`
	MemoryIDs         *[]string            `json:"memory_ids,omitempty" jsonschema:"New memory ids, replacing the old list whole; [] reads none beyond the always-included ones."`
	OneAtATime        *bool                `json:"one_at_a_time,omitempty" jsonschema:"On or off."`
	Resource          *string              `json:"resource,omitempty" jsonschema:"A new shared name, \"\" for none."`
}

// overlay keeps every field the change left out.
func (in autoPlayChangeIn) overlay(a *AutoPlay) (bool, AutoPlayInput) {
	enabled := a.Enabled
	out := AutoPlayInput{
		Moment: a.Moment, MomentStage: a.MomentStage, Schedule: a.Schedule, Conditions: a.Conditions, Priority: a.Priority,
		OnceWithinMinutes: a.OnceWithinMinutes, RunOn: a.RunOn, MemoryIDs: a.MemoryIDs, OneAtATime: a.OneAtATime, Resource: a.Resource,
		ComputerRule: &a.ComputerRule,
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
	if in.Schedule != nil {
		out.Schedule = in.Schedule
	}
	if in.OneAtATime != nil {
		out.OneAtATime = *in.OneAtATime
	}
	if in.Resource != nil {
		out.Resource = *in.Resource
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
	if in.ComputerRule != nil {
		out.ComputerRule = in.ComputerRule
	}
	out.RunAsMe = in.RunAsMe
	if in.MemoryIDs != nil {
		out.MemoryIDs = *in.MemoryIDs
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

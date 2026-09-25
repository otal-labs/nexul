package plays

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// defaultTrailSteps bounds a trail's transcript in a result; a trail keeps up to MaxTrailActivityLines steps.
const defaultTrailSteps = 50

// trailSummary is a trail without its transcript, what a list and a started or stopped run return.
type trailSummary struct {
	ID             string     `json:"id"`
	PlayID         string     `json:"play_id"`
	PlayLabel      string     `json:"play_label"`
	TargetType     TargetType `json:"target_type"`
	TargetID       string     `json:"target_id"`
	ProjectID      string     `json:"project_id"`
	ConversationID string     `json:"conversation_id"`
	State          TrailState `json:"state"`
	StarterID      string     `json:"starter_id"`
	Via            Via        `json:"via"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	FailureReason  string     `json:"failure_reason,omitempty"`
}

// trailDetail is one trail with its run choices, its question, and the newest steps; a step's raw detail stays out.
type trailDetail struct {
	trailSummary
	SelectedMemoryIDs  []string       `json:"selected_memory_ids"`
	CustomInstructions string         `json:"custom_instructions,omitempty"`
	MoveToStatusID     string         `json:"move_to_status_id,omitempty"`
	ComputerID         string         `json:"computer_id"`
	Provider           string         `json:"provider"`
	Model              string         `json:"model"`
	ReplyMessageID     string         `json:"reply_message_id,omitempty"`
	Question           *TrailQuestion `json:"question,omitempty"`
	Steps              []trailStep    `json:"steps"`
	StepsTotal         int            `json:"steps_total"`
}

type trailStep struct {
	Kind    harness.ActivityKind `json:"kind"`
	Tool    string               `json:"tool,omitempty"`
	Summary string               `json:"summary"`
	At      time.Time            `json:"at"`
}

func toTrailSummary(t *Trail) trailSummary {
	return trailSummary{
		ID: t.ID, PlayID: t.PlayID, PlayLabel: t.PlayLabel, TargetType: t.TargetType, TargetID: t.TargetID,
		ProjectID: t.ProjectID, ConversationID: t.ConversationID, State: t.State, StarterID: t.StarterID, Via: t.Via,
		StartedAt: t.StartedAt, EndedAt: t.EndedAt, LastError: t.LastError, FailureReason: t.FailureReason,
	}
}

func toTrailDetail(t *Trail, steps int) trailDetail {
	if steps < 1 {
		steps = defaultTrailSteps
	}
	tail := t.Activity[max(len(t.Activity)-steps, 0):]
	out := trailDetail{
		trailSummary: toTrailSummary(t), SelectedMemoryIDs: t.SelectedMemoryIDs, CustomInstructions: t.CustomInstructions,
		MoveToStatusID: t.MoveToStatusID, ComputerID: t.ComputerID, Provider: t.Provider, Model: t.Model,
		ReplyMessageID: t.ReplyMessageID, Question: t.Question, Steps: make([]trailStep, 0, len(tail)), StepsTotal: len(t.Activity),
	}
	for _, e := range tail {
		out.Steps = append(out.Steps, trailStep{Kind: e.Kind, Tool: e.Tool, Summary: e.Summary, At: e.At})
	}
	return out
}

type playRunIn struct {
	PlayID             string     `json:"play_id" jsonschema:"The play's id, from play_list."`
	TargetType         TargetType `json:"target_type" jsonschema:"What to run it on, matching the play's type: ticket, doc, or interview."`
	TargetID           string     `json:"target_id" jsonschema:"The ticket's or doc's id (a UUID, not a ticket key such as REF-102), or for an interview the project's id."`
	MemoryIDs          []string   `json:"memory_ids,omitempty" jsonschema:"Ids of the target project's memories to inline in full, from memory_list; the project's always-included memories come along anyway."`
	CustomInstructions string     `json:"custom_instructions,omitempty" jsonschema:"Extra instructions for this run only; they win over the play's where the two conflict."`
	MoveToStatusID     string     `json:"move_to_status_id,omitempty" jsonschema:"Ticket plays only: the status column to move the ticket to when the run ends done, from project_get."`
	ComputerID         string     `json:"computer_id,omitempty" jsonschema:"One of the caller's own paired computers, from computer_list. Omit to use the project link or the caller's pairing defaults."`
	Provider           string     `json:"provider,omitempty" jsonschema:"The harness provider to run on, for example claude. Omit to use the computer's default."`
	Model              string     `json:"model,omitempty" jsonschema:"The model to run, for example sonnet-5. Omit to use the provider's default."`
}

type trailListIn struct {
	ID         string     `json:"id,omitempty" jsonschema:"A trail's id. Returns that one trail with its run choices, open question, and newest steps instead of a list."`
	TargetType TargetType `json:"target_type,omitempty" jsonschema:"Without id: the target kind to list trails for, ticket, doc, or interview."`
	TargetID   string     `json:"target_id,omitempty" jsonschema:"Without id: the ticket's or doc's id, or for an interview the project's id."`
	PlayID     string     `json:"play_id,omitempty" jsonschema:"Without id: only the target's trails of this play."`
	Steps      int        `json:"steps,omitzero" jsonschema:"With id: how many of the newest transcript steps to return, 1 to 300. Defaults to 50."`
	mcptool.PageArgs
}

// answerIn is harness.AnswerValue with schema descriptions; its fields must stay identical for the conversion in steerTrail.
type answerIn struct {
	Selected []string `json:"selected,omitempty" jsonschema:"The chosen options, each by its value or, when it has none, its label; several only for a multi-select question."`
	Text     string   `json:"text,omitempty" jsonschema:"A typed answer, instead of choosing options."`
}

type trailUpdateIn struct {
	ID     string              `json:"id" jsonschema:"The trail's id, from play_run or trail_list."`
	Answer map[string]answerIn `json:"answer,omitempty" jsonschema:"Answers to the question a waiting trail stopped on, keyed by each question's id from trail_list, for example {\"q1\": {\"selected\": [\"Yes\"]}}. The run continues on the same trail."`
	Stop   bool                `json:"stop,omitempty" jsonschema:"true interrupts the run and ends the trail as interrupted, keeping its steps."`
}

type decisionsCheckRunIn struct {
	TicketID string `json:"ticket_id" jsonschema:"The done ticket's id (a UUID)."`
}

// RunMCPTools returns the tools that run plays and read or steer their trails; every run started here carries Via mcp (ADR 0049).
func RunMCPTools(r *Runner) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("play_run", "Run play",
			"Starts a play on a ticket, a doc, or a project's interview as the calling user, on that user's own "+
				"paired computer, and posts the run into the target's thread. Check play_list with type first to see "+
				"which plays the caller may run there; one run at a time per target. Returns the trail in state "+
				"starting; poll trail_list with its id for the outcome, and answer or stop it with trail_update.",
			mcptool.Hints{},
			func(ctx context.Context, in playRunIn) (any, error) {
				t, err := r.Run(ctx, RunInput{
					PlayID: in.PlayID, TargetType: in.TargetType, TargetID: in.TargetID, MemoryIDs: in.MemoryIDs,
					CustomInstructions: in.CustomInstructions, MoveToStatusID: in.MoveToStatusID,
					ComputerID: in.ComputerID, Provider: in.Provider, Model: in.Model, Via: ViaMCP,
				})
				if err != nil {
					return nil, startHint(err)
				}
				return toTrailSummary(t), nil
			}),
		mcptool.New("trail_list", "List trails",
			"Lists the trails of play runs on one target, newest first, filtered to one play with play_id: who "+
				"started each, its state, and how it ended. With id it returns that one trail instead, with its run "+
				"choices, the question a waiting run stopped on, and its newest transcript steps (summaries, not raw "+
				"tool output). Pass either id, or target_type and target_id. Needs plays:read, and the doc's thread "+
				"permission for a doc trail.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in trailListIn) (any, error) {
				if in.ID != "" {
					t, err := r.GetTrail(ctx, in.ID)
					if err != nil {
						return nil, trailNotFoundHint(err)
					}
					return toTrailDetail(t, in.Steps), nil
				}
				if in.TargetType == "" || in.TargetID == "" {
					return nil, fmt.Errorf("%w: pass id for one trail, or target_type (ticket, doc, or interview) and target_id "+
						"for a target's trails, which play_id only narrows; ticket_list, doc_list, and project_list give target ids", apperrs.ErrInvalid)
				}
				list, err := r.ListTrails(ctx, in.TargetType, in.TargetID)
				if err != nil {
					return nil, err
				}
				out := make([]trailSummary, 0, len(list))
				for _, t := range list {
					if in.PlayID != "" && t.PlayID != in.PlayID {
						continue
					}
					out = append(out, toTrailSummary(t))
				}
				return mcptool.Paginate(out, in.PageArgs), nil
			}),
		mcptool.New("trail_update", "Answer or stop play run",
			"Steers a running play: answer gives a waiting trail the answers to its question so the run continues "+
				"on the same trail, and stop true interrupts an active run. Pass exactly one of the two; trail_list "+
				"with the trail's id shows the question and its ids. Returns the trail. Allowed for the run's starter "+
				"or a plays:write holder in its workspace.",
			mcptool.Hints{},
			func(ctx context.Context, in trailUpdateIn) (any, error) {
				t, err := steerTrail(ctx, r, in)
				if err != nil {
					return nil, trailNotFoundHint(err)
				}
				return toTrailSummary(t), nil
			}),
		mcptool.New("decisions_check_run", "Run decisions check",
			"Runs the built-in decisions check on a done ticket again, as the calling user on their own paired "+
				"computer: it reads the ticket, its pull requests, and the project's decisions log, then adds an "+
				"entry, marks a reversed one superseded, or leaves the log alone. Use it when the ticket shows the "+
				"decisions check didn't run; play_run starts any other play. Returns the trail in state starting; "+
				"poll trail_list with its id for the outcome.",
			mcptool.Hints{},
			func(ctx context.Context, in decisionsCheckRunIn) (any, error) {
				t, err := r.RetryDecisionsCheck(ctx, in.TicketID, ViaMCP)
				if err != nil {
					return nil, startHint(err)
				}
				return toTrailSummary(t), nil
			}),
	}
}

func steerTrail(ctx context.Context, r *Runner, in trailUpdateIn) (*Trail, error) {
	if (len(in.Answer) > 0) == in.Stop {
		return nil, fmt.Errorf("%w: pass exactly one of answer or stop: true", apperrs.ErrInvalid)
	}
	if in.Stop {
		return r.Stop(ctx, in.ID)
	}
	answers := make(map[string]harness.AnswerValue, len(in.Answer))
	for id, a := range in.Answer {
		answers[id] = harness.AnswerValue(a)
	}
	return r.Answer(ctx, in.ID, harness.QuestionAnswer{Answers: answers})
}

// startHint points a refused start at the tools that recover from it.
func startHint(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; play_list shows the plays, computer_list the caller's computers, and ticket_list, doc_list, "+
			"or project_list give a target's id (a UUID)", err)
	}
	if errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("%w; trail_list with target_type and target_id shows the active trail, and trail_update with stop ends it", err)
	}
	return err
}

func trailNotFoundHint(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; trail_list with target_type and target_id shows a target's trails", err)
	}
	return err
}

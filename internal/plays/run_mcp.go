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
	SelectedMemoryIDs  []string                `json:"selected_memory_ids"`
	CustomInstructions string                  `json:"custom_instructions,omitempty"`
	ComputerID         string                  `json:"computer_id"`
	Provider           string                  `json:"provider"`
	Model              string                  `json:"model"`
	ModelOptions       []harness.OptionSetting `json:"model_options,omitempty"`
	ReplyMessageID     string                  `json:"reply_message_id,omitempty"`
	Question           *TrailQuestion          `json:"question,omitempty"`
	Steps              []trailStep             `json:"steps"`
	StepsTotal         int                     `json:"steps_total"`
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
		ComputerID: t.ComputerID, Provider: t.Provider, Model: t.Model, ModelOptions: t.ModelOptions,
		ReplyMessageID: t.ReplyMessageID, Question: t.Question, Steps: make([]trailStep, 0, len(tail)), StepsTotal: len(t.Activity),
	}
	for _, e := range tail {
		out.Steps = append(out.Steps, trailStep{Kind: e.Kind, Tool: e.Tool, Summary: e.Summary, At: e.At})
	}
	return out
}

type playRunIn struct {
	PlayID             string                  `json:"play_id,omitempty" jsonschema:"The play's id, from play_list. Required unless decisions_check is true."`
	DecisionsCheck     bool                    `json:"decisions_check,omitempty" jsonschema:"true runs the built-in decisions check again on a done ticket, instead of a play: pass it with target_type ticket and target_id, and no play_id or other run choices."`
	ResumeAutoPlays    bool                    `json:"resume_auto_plays,omitempty" jsonschema:"true resumes auto plays on a ticket or doc the daily cap paused, instead of running a play: pass it with target_type and target_id only. Its count of automatic runs starts again and its queued runs go ahead."`
	TargetType         TargetType              `json:"target_type" jsonschema:"What to run it on, matching the play's type: ticket, doc, or interview."`
	TargetID           string                  `json:"target_id" jsonschema:"The ticket's or doc's id (a UUID, not a ticket key such as REF-102), or for an interview the project's id."`
	MemoryIDs          []string                `json:"memory_ids,omitempty" jsonschema:"Ids of the target project's memories the agent reads before the run, from memory_list; the project's always-included memories come along anyway."`
	CustomInstructions string                  `json:"custom_instructions,omitempty" jsonschema:"Extra instructions for this run only; they win over the play's where the two conflict."`
	ComputerID         string                  `json:"computer_id,omitempty" jsonschema:"One of the caller's own paired computers, from computer_list. Omit to use the caller's own link for this project, else their pairing defaults."`
	Provider           string                  `json:"provider,omitempty" jsonschema:"The harness provider to run on, for example claude. Omit to use the computer's default."`
	Model              string                  `json:"model,omitempty" jsonschema:"The model to run, for example sonnet-5. Omit to use the provider's default."`
	ModelOptions       []harness.OptionSetting `json:"model_options,omitempty" jsonschema:"Options for the picked model, for example [{\"id\": \"effort\", \"value\": \"high\"}, {\"id\": \"fastMode\", \"value\": true}]: a choice id for a select option such as reasoning level or context window, true or false for a switch such as fast mode, as the harness lists them per model. Only applies with model; an option left out keeps the harness default."`
}

type trailListIn struct {
	ID         string     `json:"id,omitempty" jsonschema:"A trail's id. Returns that one trail with its run choices, open question, and newest steps instead of a list."`
	TargetType TargetType `json:"target_type,omitempty" jsonschema:"Without id: the target kind to list trails for, ticket, doc, or interview."`
	TargetID   string     `json:"target_id,omitempty" jsonschema:"Without id: the ticket's or doc's id, or for an interview the project's id."`
	PlayID     string     `json:"play_id,omitempty" jsonschema:"Without id: only the target's trails of this play."`
	Steps      int        `json:"steps,omitzero" jsonschema:"With id: how many of the newest transcript steps to return, 1 to 300. Defaults to 50."`
	Queue      bool       `json:"queue,omitempty" jsonschema:"Without id: true lists the target's auto runs instead of its trails, newest first: queued, started, skipped, didn't run, or cancelled, each with why, plus whether the daily cap paused auto plays there."`
	mcptool.PageArgs
}

// answerIn is harness.AnswerValue with schema descriptions; its fields must stay identical for the conversion in steerTrail.
type answerIn struct {
	Selected []string `json:"selected,omitempty" jsonschema:"The chosen options, each by its value or, when it has none, its label; several only for a multi-select question."`
	Text     string   `json:"text,omitempty" jsonschema:"A typed answer, instead of choosing options."`
}

type trailUpdateIn struct {
	ID       string              `json:"id" jsonschema:"The trail's id, from play_run or trail_list."`
	Answer   map[string]answerIn `json:"answer,omitempty" jsonschema:"Answers to the question a waiting trail stopped on, keyed by each question's id from trail_list, for example {\"q1\": {\"selected\": [\"Yes\"]}}. The run continues on the same trail."`
	Stop     bool                `json:"stop,omitempty" jsonschema:"true interrupts the run and ends the trail as interrupted, keeping its steps."`
	Continue string              `json:"continue,omitempty" jsonschema:"A message for an ended trail's agent, sent to the same harness thread as the trail's next turn, for example \"Carry on with the second option.\" The play's instructions are not sent again."`
	Cancel   bool                `json:"cancel,omitempty" jsonschema:"true cancels a queued auto run before it starts; id is then the queued run's id from trail_list with queue true, not a trail's."`
}

// queuePage is a target's auto runs as trail_list with queue returns them: one page of items and the paused state.
type queuePage struct {
	mcptool.Page[*QueueItem]
	Paused   bool `json:"paused"`
	AutoRuns int  `json:"auto_runs"`
	DailyCap int  `json:"daily_cap"`
}

// RunMCPTools returns the tools that run plays and read or steer their trails; every run started here carries Via mcp (ADR 0049).
func RunMCPTools(r *Runner) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("play_run", "Run play",
			"Starts a play on a ticket, a doc, or a project's interview as the calling user, on that user's own "+
				"paired computer, and posts the run into the target's thread. Check play_list with type first to see "+
				"which plays the caller may run there; one run at a time per target. A doc play locks its doc as the run "+
				"starts and leaves it locked, so the doc's title and body refuse edits until someone unlocks it; "+
				"Clarify via AI is the exception, its run opens the doc's next clarification round and unlocks the doc again "+
				"when it ends, unless the doc was locked before. "+
				"With decisions_check true instead "+
				"of a play_id it reruns the built-in decisions check on a done ticket, which reads the ticket, its pull "+
				"requests, and the project's decisions log, then adds an entry, marks a reversed one superseded, or "+
				"leaves the log alone; use that when the ticket shows the decisions check didn't run. With "+
				"resume_auto_plays true it resumes auto plays on a ticket or doc the daily cap paused, so its queued runs go "+
				"ahead, and returns its queue as trail_list with queue does. Otherwise returns the trail "+
				"in state starting; poll trail_list with its id for the outcome, and answer or stop it with trail_update.",
			mcptool.Hints{},
			func(ctx context.Context, in playRunIn) (any, error) {
				if in.ResumeAutoPlays {
					return resumeAutoPlays(ctx, r, in)
				}
				t, err := runPlay(ctx, r, in)
				if err != nil {
					return nil, startHint(err)
				}
				return toTrailSummary(t), nil
			}),
		mcptool.New("trail_list", "List trails",
			"Lists the trails of play runs on one target, newest first, filtered to one play with play_id: who "+
				"started each, its state, and how it ended. With id it returns that one trail instead, with its run "+
				"choices, the question a waiting run stopped on, and its newest transcript steps (summaries, not raw "+
				"tool output). Pass either id, or target_type and target_id. With queue true it lists the target's auto "+
				"runs instead: each queued, started, skipped, didn't-run, or cancelled run of an auto play with why, and "+
				"whether the daily cap paused auto plays there. Needs plays:read, and the doc's thread "+
				"permission for a doc trail.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in trailListIn) (any, error) {
				if in.Queue && in.ID == "" {
					return listQueue(ctx, r, in)
				}
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
				list, err := r.ListTrails(ctx, in.TargetType, in.TargetID, in.PlayID)
				if err != nil {
					return nil, err
				}
				out := make([]trailSummary, 0, len(list))
				for _, t := range list {
					out = append(out, toTrailSummary(t))
				}
				return mcptool.Paginate(out, in.PageArgs), nil
			}),
		mcptool.New("trail_update", "Answer, stop, or continue play run",
			"Steers a play run: answer gives a waiting trail the answers to its question so the run continues "+
				"on the same trail, stop true interrupts an active run, and continue sends a message to an ended run's "+
				"own harness thread so it goes on as the same trail, without the play's instructions again. When that "+
				"thread was deleted, continue starts the play again as a new run carrying the message and returns that "+
				"trail. Pass exactly one of the three; trail_list with the trail's id shows the question and its ids. "+
				"Returns the trail. Allowed for the run's starter or a plays:write holder in its workspace. "+
				"With cancel true instead, id names a queued auto run from trail_list with queue true, which is dropped "+
				"before it starts and returned; allowed for the person it would run on or an autoplays:write holder.",
			mcptool.Hints{},
			func(ctx context.Context, in trailUpdateIn) (any, error) {
				if in.Cancel {
					return cancelQueued(ctx, r, in)
				}
				t, err := steerTrail(ctx, r, in)
				if err != nil {
					return nil, trailNotFoundHint(err)
				}
				return toTrailSummary(t), nil
			}),
	}
}

func listQueue(ctx context.Context, r *Runner, in trailListIn) (any, error) {
	q, err := r.GetQueue(ctx, in.TargetType, in.TargetID)
	if err != nil {
		return nil, err
	}
	return toQueuePage(q, in.PageArgs), nil
}

func cancelQueued(ctx context.Context, r *Runner, in trailUpdateIn) (any, error) {
	if len(in.Answer) > 0 || in.Stop || in.Continue != "" {
		return nil, fmt.Errorf("%w: cancel takes only id, a queued run's id", apperrs.ErrInvalid)
	}
	return r.CancelQueued(ctx, in.ID)
}

func toQueuePage(q *Queue, page mcptool.PageArgs) queuePage {
	return queuePage{Page: mcptool.Paginate(q.Items, page), Paused: q.Paused, AutoRuns: q.AutoRuns, DailyCap: q.DailyCap}
}

// resumeAutoPlays is play_run's resume_auto_plays: it starts no run itself, but lets the paused queue go on.
func resumeAutoPlays(ctx context.Context, r *Runner, in playRunIn) (any, error) {
	choices := in.PlayID != "" || in.DecisionsCheck || len(in.MemoryIDs) > 0 || in.CustomInstructions != "" ||
		in.ComputerID != "" || in.Provider != "" || in.Model != "" || len(in.ModelOptions) > 0
	if choices {
		return nil, fmt.Errorf("%w: resume_auto_plays takes only target_type and target_id", apperrs.ErrInvalid)
	}
	q, err := r.ResumeAutoPlays(ctx, in.TargetType, in.TargetID)
	if err != nil {
		return nil, err
	}
	return toQueuePage(q, mcptool.PageArgs{}), nil
}

// runPlay starts the named play, or with decisions_check the built-in decisions check (ADR 0066: it is a play).
func runPlay(ctx context.Context, r *Runner, in playRunIn) (*Trail, error) {
	if !in.DecisionsCheck {
		if in.PlayID == "" {
			return nil, fmt.Errorf("%w: pass play_id from play_list, or decisions_check true to rerun the decisions check", apperrs.ErrInvalid)
		}
		return r.Run(ctx, RunInput{
			PlayID: in.PlayID, TargetType: in.TargetType, TargetID: in.TargetID, MemoryIDs: in.MemoryIDs,
			CustomInstructions: in.CustomInstructions,
			ComputerID:         in.ComputerID, Provider: in.Provider, Model: in.Model, ModelOptions: in.ModelOptions, Via: ViaMCP,
		})
	}
	choices := in.PlayID != "" || len(in.MemoryIDs) > 0 || in.CustomInstructions != "" ||
		in.ComputerID != "" || in.Provider != "" || in.Model != "" || len(in.ModelOptions) > 0
	if choices || in.TargetType != TargetTicket {
		return nil, fmt.Errorf("%w: decisions_check takes only target_type ticket and target_id; it runs with its own "+
			"instructions on the caller's default computer", apperrs.ErrInvalid)
	}
	return r.RetryDecisionsCheck(ctx, in.TargetID, ViaMCP)
}

func steerTrail(ctx context.Context, r *Runner, in trailUpdateIn) (*Trail, error) {
	picked := 0
	for _, set := range []bool{len(in.Answer) > 0, in.Stop, in.Continue != ""} {
		if set {
			picked++
		}
	}
	if picked != 1 {
		return nil, fmt.Errorf("%w: pass exactly one of answer, stop: true, or continue", apperrs.ErrInvalid)
	}
	if in.Continue != "" {
		return r.Continue(ctx, in.ID, in.Continue, ViaMCP)
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

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

// playRunSummary is a run on a play's list of runs, with the ticket's key and title or the doc's, stack's or
// project's name it ran on.
type playRunSummary struct {
	trailSummary
	TargetKey   string `json:"target_key,omitempty"`
	TargetTitle string `json:"target_title"`
	AutoPlayID  string `json:"auto_play_id,omitempty"`
}

// trailDetail is one trail with its run choices, its question, and the newest steps; a step's raw detail stays out.
type trailDetail struct {
	trailSummary
	SelectedMemoryIDs  []string                `json:"selected_memory_ids"`
	CustomInstructions string                  `json:"custom_instructions,omitempty"`
	ComputerID         string                  `json:"computer_id"`
	PassedOver         []PassedOver            `json:"passed_over,omitempty" jsonschema:"To the run's starter alone: the computers it could not land on, and why."`
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
		ComputerID: t.ComputerID, PassedOver: t.PassedOver, Provider: t.Provider, Model: t.Model, ModelOptions: t.ModelOptions,
		ReplyMessageID: t.ReplyMessageID, Question: t.Question, Steps: make([]trailStep, 0, len(tail)), StepsTotal: len(t.Activity),
	}
	for _, e := range tail {
		out.Steps = append(out.Steps, trailStep{Kind: e.Kind, Tool: e.Tool, Summary: e.Summary, At: e.At})
	}
	return out
}

type playRunIn struct {
	PlayID             string                  `json:"play_id,omitempty" jsonschema:"The play's id, from play_list. Required unless decisions_check is true."`
	DecisionsCheck     bool                    `json:"decisions_check,omitempty" jsonschema:"true runs the Decisions check play on a done ticket instead of a play_id; pass only target_type ticket and target_id."`
	ResumeAutoPlays    bool                    `json:"resume_auto_plays,omitempty" jsonschema:"true resumes auto plays the daily cap paused on a ticket or doc, instead of running a play; pass only target_type and target_id. Its automatic-run count restarts and queued runs go ahead."`
	TargetType         TargetType              `json:"target_type" jsonschema:"What to run it on, matching the play's type: ticket, doc, interview, or stack."`
	TargetID           string                  `json:"target_id" jsonschema:"The ticket's or doc's id (a UUID, not a ticket key such as REF-102), or for an interview the project's id."`
	MemoryIDs          []string                `json:"memory_ids,omitempty" jsonschema:"Ids of the target project's memories to read before the run, from memory_list; its always-included ones come anyway."`
	CustomInstructions string                  `json:"custom_instructions,omitempty" jsonschema:"Extra instructions for this run only; they win over the play's where the two conflict."`
	ComputerID         string                  `json:"computer_id,omitempty" jsonschema:"One of your paired computers, from computer_list. Omit to use the first that can take the run."`
	T3ProjectID        string                  `json:"t3_project_id,omitempty" jsonschema:"With computer_id: the T3 project to run in, from computer_list with that computer's id, remembered as the target project's checkout there. Required when none of your computers has a checkout of the project."`
	Provider           string                  `json:"provider,omitempty" jsonschema:"Harness provider, for example claude. Omit for the computer's default."`
	Model              string                  `json:"model,omitempty" jsonschema:"Model, for example sonnet-5. Omit for the provider's default."`
	ModelOptions       []harness.OptionSetting `json:"model_options,omitempty" jsonschema:"Options for the picked model, for example [{\"id\": \"effort\", \"value\": \"high\"}, {\"id\": \"fastMode\", \"value\": true}]: a choice id for a select option, true or false for a switch, as the harness lists them per model. Only applies with model."`
}

type trailListIn struct {
	ID         string     `json:"id,omitempty" jsonschema:"A trail's id. Returns that one trail with its run choices, open question, and newest steps instead of a list."`
	TargetType TargetType `json:"target_type,omitempty" jsonschema:"Without id: the target kind, ticket, doc, interview, or stack."`
	TargetID   string     `json:"target_id,omitempty" jsonschema:"Without id: the ticket's or doc's id, or for an interview the project's id."`
	PlayID     string     `json:"play_id,omitempty" jsonschema:"Without id: only the trails of this play."`
	Steps      int        `json:"steps,omitzero" jsonschema:"With id: newest transcript steps to return, 1 to 300. Defaults to 50."`
	Queue      bool       `json:"queue,omitempty" jsonschema:"Without id: true lists the target's auto runs instead of its trails, newest first: queued, started, skipped, didn't run, or cancelled, each with why, plus whether the daily cap paused auto plays there."`
	mcptool.PageArgs
}

// answerIn is harness.AnswerValue with schema descriptions; its fields must stay identical for the conversion in steerTrail.
type answerIn struct {
	Selected []string `json:"selected,omitempty" jsonschema:"The chosen options, each by its value or, when it has none, its label; several only for multi-select."`
	Text     string   `json:"text,omitempty" jsonschema:"A typed answer, instead of choosing options."`
}

type trailUpdateIn struct {
	ID       string              `json:"id" jsonschema:"The trail's id, from play_run or trail_list."`
	Answer   map[string]answerIn `json:"answer,omitempty" jsonschema:"Answers to the question a waiting trail stopped on, keyed by each question's id from trail_list, for example {\"q1\": {\"selected\": [\"Yes\"]}}."`
	Stop     bool                `json:"stop,omitempty" jsonschema:"true interrupts the run and ends the trail as interrupted, keeping its steps."`
	Continue string              `json:"continue,omitempty" jsonschema:"A message for an ended trail's agent, sent to the same harness thread as its next turn, without the play's instructions again."`
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
			"Starts a play on a ticket, a doc, or a project's interview as the calling user, on their own "+
				"paired computer, and posts the run into the target's thread; one run at a time per target. Check "+
				"play_list with type first for the plays the caller may run there. A doc play locks its doc as the run "+
				"starts and leaves it locked, so its title and body refuse edits until someone unlocks it; "+
				"Clarify via AI instead opens the doc's next clarification round and unlocks the doc when it ends, "+
				"unless it was locked before. With decisions_check true instead of a play_id it runs the Decisions "+
				"check play on a done ticket, which reads it and its pull requests and updates the project's decisions "+
				"log; use it when the ticket shows the check didn't run. With resume_auto_plays true it lets queued runs go ahead and "+
				"returns the queue as trail_list with queue does. Otherwise returns the trail in state starting; poll "+
				"trail_list with its id, and answer or stop it with trail_update. The first run in a project the "+
				"caller has not linked needs computer_id and t3_project_id, saved as their link there.",
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
			"Lists the trails of play runs on one target, newest first, optionally of one play, or with play_id alone "+
				"the runs of that play on every target. With id it returns that one trail instead, with its run "+
				"choices, the question a waiting run stopped on, and its newest transcript steps. "+
				"With queue true it lists the auto runs of the target instead. A trail is visible to its starter and to "+
				"whoever can see its play (a doc trail also needs the doc thread permission); others are left out or not found.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in trailListIn) (any, error) {
				return listTrails(ctx, r, in)
			}),
		mcptool.New("trail_update", "Answer, stop, or continue play run",
			"Steers a play run: answer gives a waiting trail its question's answers so the run continues "+
				"on the same trail, stop true interrupts an active run, and continue sends a message to an ended run's "+
				"harness thread, or starts the play again as a new run carrying it when that thread was deleted. "+
				"trail_list with the trail's id shows the question and its ids. "+
				"Returns the trail. Allowed for the run's starter or someone holding plays:write on its play. "+
				"With cancel true instead, id names a queued auto run, which is dropped "+
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
		in.ComputerID != "" || in.T3ProjectID != "" || in.Provider != "" || in.Model != "" || len(in.ModelOptions) > 0
	if choices {
		return nil, fmt.Errorf("%w: resume_auto_plays takes only target_type and target_id", apperrs.ErrInvalid)
	}
	q, err := r.ResumeAutoPlays(ctx, in.TargetType, in.TargetID)
	if err != nil {
		return nil, err
	}
	return toQueuePage(q, mcptool.PageArgs{}), nil
}

// runPlay starts the named play, or with decisions_check the workspace's seeded decisions check.
func runPlay(ctx context.Context, r *Runner, in playRunIn) (*Trail, error) {
	if !in.DecisionsCheck {
		if in.PlayID == "" {
			return nil, fmt.Errorf("%w: pass play_id from play_list, or decisions_check true to rerun the decisions check", apperrs.ErrInvalid)
		}
		return r.Run(ctx, RunInput{
			PlayID: in.PlayID, TargetType: in.TargetType, TargetID: in.TargetID, MemoryIDs: in.MemoryIDs,
			CustomInstructions: in.CustomInstructions,
			ComputerID:         in.ComputerID, HarnessProjectID: in.T3ProjectID, Provider: in.Provider, Model: in.Model,
			ModelOptions: in.ModelOptions, Via: ViaMCP,
		})
	}
	choices := in.PlayID != "" || len(in.MemoryIDs) > 0 || in.CustomInstructions != "" ||
		in.ComputerID != "" || in.T3ProjectID != "" || in.Provider != "" || in.Model != "" || len(in.ModelOptions) > 0
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
		t, err := r.Continue(ctx, in.ID, in.Continue, ViaMCP)
		return t, locationHint(err)
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
	return locationHint(err)
}

// locationHint points a run that needs a location, pressed or started again by a continue, at play_run (ADR 0145).
func locationHint(err error) error {
	var refusal *HarnessRefusal
	if errors.As(err, &refusal) && refusal.Reason == RefusalNeedsLocation {
		return fmt.Errorf("%w Ask the user which computer and T3 project to run in, then call play_run again with computer_id "+
			"and t3_project_id; computer_list lists their computers, and with a computer's id that computer's T3 projects", err)
	}
	return err
}

func trailNotFoundHint(err error) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; trail_list with target_type and target_id shows a target's trails", err)
	}
	return err
}

// listPlayRuns is trail_list's page of one play's runs across every target.
func listPlayRuns(ctx context.Context, r *Runner, in trailListIn) (any, error) {
	w := in.Window()
	runs, total, err := r.ListPlayRuns(ctx, in.PlayID, false, w)
	if err != nil {
		return nil, err
	}
	out := make([]playRunSummary, 0, len(runs))
	for i := range runs {
		out = append(out, playRunSummary{trailSummary: toTrailSummary(&runs[i].Trail), TargetKey: runs[i].TargetKey, TargetTitle: runs[i].TargetTitle, AutoPlayID: runs[i].AutoPlayID})
	}
	return mcptool.PageOf(out, total, w), nil
}

// listTrails is trail_list: one trail by id, a target's auto runs or trails, or one play's runs on every target.
func listTrails(ctx context.Context, r *Runner, in trailListIn) (any, error) {
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
	if in.TargetType == "" && in.TargetID == "" && in.PlayID != "" {
		return listPlayRuns(ctx, r, in)
	}
	if in.TargetType == "" || in.TargetID == "" {
		return nil, fmt.Errorf("%w: pass id for one trail, or target_type (ticket, doc, interview, or stack) and target_id "+
			"for a target's trails, which play_id only narrows, or play_id alone for that play's runs; ticket_list, doc_list, and project_list give target ids", apperrs.ErrInvalid)
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
}

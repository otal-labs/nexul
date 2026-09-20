package plays

import (
	"context"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// RunMCPTools returns the run tools; every run started here carries Via mcp (ADR 0049 provenance).
func RunMCPTools(r *Runner) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name: "play_run",
			Description: "Run a play on a ticket or a doc as the calling user, on that user's own paired harness. " +
				"Returns the trail in state starting; poll play_run_get for its outcome. Memories are inlined in " +
				"full (the project's always-included ones first); custom instructions win over the play's where they conflict. " +
				"computer_id, provider, and model are optional; left unset, the run resolves the caller's own project link or pairing defaults.",
			InputSchema: objectSchema(map[string]any{
				"play_id":             map[string]any{"type": "string"},
				"target_type":         map[string]any{"type": "string", "enum": []string{"ticket", "doc"}},
				"target_id":           map[string]any{"type": "string"},
				"memory_ids":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"custom_instructions": map[string]any{"type": "string"},
				"move_to_status_id":   map[string]any{"type": "string", "description": "Ticket plays only: the column to move to on success."},
				"computer_id":         map[string]any{"type": "string", "description": "One of the caller's own paired computers; must belong to them."},
				"provider":            map[string]any{"type": "string"},
				"model":               map[string]any{"type": "string"},
			}, "play_id", "target_type", "target_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "play_id", "target_type", "target_id")
				if err != nil {
					return nil, err
				}
				return r.Run(ctx, RunInput{
					PlayID: vals[0], TargetType: TargetType(vals[1]), TargetID: vals[2],
					MemoryIDs:          stringsArg(args["memory_ids"]),
					CustomInstructions: mcptool.OptionalString(args["custom_instructions"]),
					MoveToStatusID:     mcptool.OptionalString(args["move_to_status_id"]),
					ComputerID:         mcptool.OptionalString(args["computer_id"]),
					Provider:           mcptool.OptionalString(args["provider"]),
					Model:              mcptool.OptionalString(args["model"]),
					Via:                ViaMCP,
				})
			},
		},
		{
			Name:        "play_run_get",
			Description: "Get one play run's trail: state, harness session, activity lines, outcome, and reply message id.",
			InputSchema: objectSchema(map[string]any{
				"id": map[string]any{"type": "string"},
			}, "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return r.GetTrail(ctx, id)
			},
		},
		{
			Name: "play_run_stop",
			Description: "Stop an active play run: interrupts the harness turn and ends the trail as interrupted, keeping " +
				"its activity lines. Allowed for the run's starter or a plays:write holder in its workspace.",
			InputSchema: objectSchema(map[string]any{
				"id": map[string]any{"type": "string"},
			}, "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return r.Stop(ctx, id)
			},
		},
		{
			Name: "play_run_answer",
			Description: "Answer the question a waiting play run stopped on. answers maps each question id to the chosen " +
				"option label (an array for a multi-select) or free text; the run continues on the same trail and session. " +
				"Allowed for the run's starter or a plays:write holder in its workspace.",
			InputSchema: objectSchema(map[string]any{
				"id":      map[string]any{"type": "string"},
				"answers": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": []string{"string", "array"}}},
			}, "id", "answers"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return r.Answer(ctx, id, harness.QuestionAnswer{Answers: answersArg(args["answers"])})
			},
		},
		{
			Name:        "play_list_runs",
			Description: "List a ticket's or a doc's play runs, newest first.",
			InputSchema: objectSchema(map[string]any{
				"target_type": map[string]any{"type": "string", "enum": []string{"ticket", "doc"}},
				"target_id":   map[string]any{"type": "string"},
			}, "target_type", "target_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "target_type", "target_id")
				if err != nil {
					return nil, err
				}
				return r.ListTrails(ctx, TargetType(vals[0]), vals[1])
			},
		},
	}
}

// answersArg reads the tool's answers object: a string is free text or one label, an array is a multi-select.
func answersArg(v any) map[string]harness.AnswerValue {
	raw, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]harness.AnswerValue, len(raw))
	for id, value := range raw {
		if text, ok := value.(string); ok {
			out[id] = harness.AnswerValue{Text: text}
			continue
		}
		out[id] = harness.AnswerValue{Selected: stringsArg(value)}
	}
	return out
}

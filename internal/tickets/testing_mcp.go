package tickets

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func ticketTestingMCPTools(s *Service) []mcptool.Tool {
	idOnly := map[string]any{
		"type":       "object",
		"properties": map[string]any{"id": map[string]any{"type": "string"}},
		"required":   []string{"id"},
	}
	return []mcptool.Tool{
		{
			Name:        "ticket_get_test_target",
			Description: "Where to test a ticket: its branch's preview URL (kind preview), else a shared test environment (kind shared, which may include other changes). Production is never returned; an empty url means no safe environment exists, so ask for a deploy branch on its own network rather than testing anywhere else.",
			InputSchema: idOnly,
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.TestTarget(ctx, id)
			},
		},
		{
			Name:        "ticket_test_pass",
			Description: "Record that a ticket passed testing against its acceptance criteria: it moves to the project's first done-stage column, the caller becomes its tester only when none is assigned, and \"Passed by Nexul · for <login>\" with the test URL is posted to its thread.",
			InputSchema: idOnly,
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.TestPass(ctx, id, true)
			},
		},
		{
			Name:        "ticket_test_fail",
			Description: "Record that a ticket failed testing: posts the bug report, headed \"Test failed by Nexul · for <login>\" (steps to reproduce, expected result, actual result, screenshots as ticket attachment ids) to the ticket's thread and moves it back to the project's first progress-stage column. No bug ticket is created. A done ticket is refused; file a bug found in it with ticket_create instead.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":          map[string]any{"type": "string"},
					"steps":       map[string]any{"type": "string", "description": "Steps to reproduce."},
					"expected":    map[string]any{"type": "string", "description": "What should have happened."},
					"actual":      map[string]any{"type": "string", "description": "What happened instead."},
					"screenshots": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids of images attached to the ticket."},
				},
				"required": []string{"id", "actual"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				report := TestReport{
					Steps:    mcptool.OptionalString(args["steps"]),
					Expected: mcptool.OptionalString(args["expected"]),
					Actual:   mcptool.OptionalString(args["actual"]),
				}
				if raw, ok := args["screenshots"].([]any); ok && len(raw) > 0 {
					if report.Screenshots, err = stringSliceArg(args, "screenshots"); err != nil {
						return nil, err
					}
				}
				return s.TestFail(ctx, id, report, true)
			},
		},
	}
}

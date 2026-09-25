package tickets

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools are the ticket tools that need only this domain; the ones showing project names live in internal/mcp/composite.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{ticketDeleteTool(s), ticketTestReportTool(s)}
}

// resolve names the tool that finds tickets when the one asked for is missing.
func resolve(ctx context.Context, s *Service, idOrKey string) (*Ticket, error) {
	t, err := s.Resolve(ctx, idOrKey)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w; ticket_list finds tickets by text or project", err)
	}
	return t, err
}

type ticketDeleteIn struct {
	ID string `json:"id" jsonschema:"The ticket's id or its key, for example REF-102."`
}

func ticketDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("ticket_delete", "Delete ticket",
		"Deletes a ticket for good, with its labels and links; there is no restore. Use it for a ticket filed by "+
			"mistake or a duplicate; to finish or abandon work, move the ticket to a done-stage column with "+
			"ticket_update instead, which keeps its history. Returns the deleted ticket's id.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in ticketDeleteIn) (any, error) {
			t, err := resolve(ctx, s, in.ID)
			if err != nil {
				return nil, err
			}
			if err := s.Delete(ctx, t.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(t.ID), nil
		})
}

type ticketTestReportIn struct {
	ID          string   `json:"id" jsonschema:"The ticket's id or its key, for example REF-102."`
	Outcome     string   `json:"outcome" jsonschema:"The test result: pass or fail."`
	Actual      string   `json:"actual,omitempty" jsonschema:"What happened instead, in markdown. Required when outcome is fail."`
	Steps       string   `json:"steps,omitempty" jsonschema:"Steps to reproduce the failure, in markdown."`
	Expected    string   `json:"expected,omitempty" jsonschema:"What should have happened, in markdown."`
	Screenshots []string `json:"screenshots,omitempty" jsonschema:"Ids of images already attached to the ticket, for example att-9."`
}

type testReportResult struct {
	ID       string `json:"id"`
	Outcome  string `json:"outcome"`
	StatusID string `json:"status_id"`
	Tester   string `json:"tester"`
}

func ticketTestReportTool(s *Service) mcptool.Tool {
	return mcptool.New("ticket_test_report", "Report test result",
		"Records the result of testing a ticket against its acceptance criteria, signed as Nexul for you. On pass "+
			"the ticket moves to the project's first done-stage column, you become its tester when none is assigned, "+
			"and a passed note with the test target's URL is posted to its thread. On fail the report (actual, and "+
			"steps, expected, and screenshots when given) is posted to its thread and the ticket moves back to the "+
			"first progress-stage column; no bug ticket is created, and a done ticket is refused, so file a bug found "+
			"in it with ticket_create instead. ticket_get shows where to test (test_target). Returns the ticket's new "+
			"status_id.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in ticketTestReportIn) (any, error) {
			t, err := resolve(ctx, s, in.ID)
			if err != nil {
				return nil, err
			}
			moved, err := reportTest(ctx, s, t.ID, in)
			if err != nil {
				return nil, err
			}
			return testReportResult{ID: moved.ID, Outcome: in.Outcome, StatusID: string(moved.Status), Tester: moved.Tester}, nil
		})
}

func reportTest(ctx context.Context, s *Service, id string, in ticketTestReportIn) (*Ticket, error) {
	if in.Outcome == "pass" {
		return s.TestPass(ctx, id, true)
	}
	if in.Outcome == "fail" {
		return s.TestFail(ctx, id, TestReport{Steps: in.Steps, Expected: in.Expected, Actual: in.Actual, Screenshots: in.Screenshots}, true)
	}
	return nil, fmt.Errorf("%w: outcome must be pass or fail, got %q", apperrs.ErrInvalid, in.Outcome)
}

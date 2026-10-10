package plays

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// DecisionsCheckKey is the built-in key of the decisions check, the ticket play whose seeded auto play runs it as a
// ticket enters done (ADR 0132).
const DecisionsCheckKey = "decisions-check"

const decisionsCheckLabel = "Decisions check"

const decisionsCheckDescription = "When a ticket reaches done, an agent decides whether it changed how the project works and logs it."

const decisionsCheckInstructions = "This ticket just reached done. Decide whether it changed how the project works: a new pattern, " +
	"a library added or dropped, a rule or an earlier decision reversed. Routine work that follows the existing patterns " +
	"changes nothing. Read the ticket and its linked pull requests with `ticket_get`, each pull request with " +
	"`pull_request_get`, and " +
	"the project's decisions log, the memory of kind `decisions_log` that `memory_list` finds, with `memory_get`. If nothing " +
	"changed, reply \"No decision recorded\" with one line on why, and write nothing. Otherwise add one entry of at most " +
	"three lines as its own paragraph at the end of the log: `<YYYY-MM-DD> — <the decision in one line>`, then " +
	"`Why: <one line>`, then `Ticket: [<ticket key>](/tickets/<ticket id>)`. When the decision reverses an earlier entry, " +
	"append ` (superseded by <ticket key>)` to that entry's first line instead of deleting it, and leave every other " +
	"entry as it is. Save with `memory_update` on the log; if the project has no log yet, create it with `memory_create` " +
	"passing `kind` `decisions_log` and the project id. Reply with the entry you wrote."

// decisionsCheckAutoPlay is the decisions check's seeded auto play: whoever moves a ticket into done runs it, off until
// a workspace switches it on.
func decisionsCheckAutoPlay() *AutoPlayInput {
	done := StageDone
	return &AutoPlayInput{
		Moment: MomentTicketEnteredStage, MomentStage: &done, RunOn: RunOnCauser,
		Conditions: Conditions{Match: MatchAll, Groups: []Group{}},
		Priority:   Priority{Rules: []PriorityRule{}, Otherwise: LevelNormal},
	}
}

// RetryDecisionsCheck runs the workspace's decisions check on a done ticket on the caller's own harness, as pressing
// the play does; it is how a ticket's "Decisions check didn't run" is run again.
func (r *Runner) RetryDecisionsCheck(ctx context.Context, ticketID string, via Via) (*Trail, error) {
	if actorID(ctx) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	tgt, err := r.readTarget(ctx, TargetTicket, ticketID)
	if err != nil {
		return nil, err
	}
	workspaceID, err := r.projects.WorkspaceForProject(ctx, tgt.projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for project %s: %w", tgt.projectID, err)
	}
	list, err := r.plays.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list plays for workspace %s: %w", workspaceID, err)
	}
	play := findBuiltin(list, DecisionsCheckKey)
	if play == nil {
		return nil, fmt.Errorf("%w: this workspace's decisions check play was deleted", apperrs.ErrNotFound)
	}
	return r.startRun(ctx, RunInput{PlayID: play.ID, TargetType: TargetTicket, TargetID: ticketID, Via: via}, launchPress)
}

// DefaultInstructions returns every built-in instruction text a run hands the agent, the seeded plays' and the origin
// line; they name MCP tools.
func DefaultInstructions() []string {
	return []string{fixWithAIInstructions, toTicketsInstructions, interviewInstructions, testWithAIInstructions, decisionsCheckInstructions, originLine}
}

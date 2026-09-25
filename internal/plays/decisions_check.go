package plays

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DecisionsCheckPlayID is the built-in decisions check's play id on its trails; no plays row holds it.
const DecisionsCheckPlayID = "decisions-check"

const decisionsCheckLabel = "Decisions check"

const decisionsCheckInstructions = "This ticket just reached done. Decide whether it changed how the project works: a new pattern, " +
	"a library added or dropped, a rule or an earlier decision reversed. Routine work that follows the existing patterns " +
	"changes nothing. Read the ticket and its linked pull requests with `ticket_get`, each pull request with " +
	"`pull_request_get`, and " +
	"the project's decisions log, the memory of kind `decisions_log` in your memory index, with `memory_get`. If nothing " +
	"changed, reply \"No decision recorded\" with one line on why, and write nothing. Otherwise add one entry of at most " +
	"three lines as its own paragraph at the end of the log: `<YYYY-MM-DD> — <the decision in one line>`, then " +
	"`Why: <one line>`, then `Ticket: [<ticket key>](/tickets/<ticket id>)`. When the decision reverses an earlier entry, " +
	"append ` (superseded by <ticket key>)` to that entry's first line instead of deleting it, and leave every other " +
	"entry as it is. Save with `memory_update` on the log; if the project has no log yet, create it with `memory_create` " +
	"passing `kind` `decisions_log` and the project id. Reply with the entry you wrote."

// decisionsCheckPlay is the built-in play the check runs as; no workspace lists, edits, disables, or excludes it.
func decisionsCheckPlay(workspaceID string) *Play {
	stage := StageDone
	return &Play{
		ID: DecisionsCheckPlayID, WorkspaceID: workspaceID, Label: decisionsCheckLabel, Type: TypeTicket,
		Instructions: decisionsCheckInstructions, Enabled: true, ShowWhenStage: &stage,
	}
}

// RetryDecisionsCheck reruns the check on a done ticket on the caller's own harness, keeping refusals the way a pressed play does.
func (r *Runner) RetryDecisionsCheck(ctx context.Context, ticketID string, via Via) (*Trail, error) {
	starter := actorID(ctx)
	if starter == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	return r.startDecisionsCheck(ctx, ticketID, starter, via, false)
}

// startDecisionsCheck starts the check as starter; record keeps every refusal as a failed trail on the ticket.
func (r *Runner) startDecisionsCheck(ctx context.Context, ticketID, starter string, via Via, record bool) (*Trail, error) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	tgt, err := r.readTarget(ctx, TargetTicket, ticketID)
	if err != nil {
		return nil, err
	}
	if tgt.stage != StageDone {
		return nil, fmt.Errorf("%w: the decisions check runs on done tickets; this ticket is in %s", apperrs.ErrInvalid, tgt.stage)
	}
	workspaceID, err := r.projects.WorkspaceForProject(ctx, tgt.projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for project %s: %w", tgt.projectID, err)
	}
	play := decisionsCheckPlay(workspaceID)
	trail := &Trail{
		ID: ids.New(), WorkspaceID: workspaceID, PlayID: play.ID, PlayLabel: play.Label, TargetType: TargetTicket,
		TargetID: ticketID, ProjectID: tgt.projectID, StarterID: starter, Via: via, SelectedMemoryIDs: []string{},
		State: TrailStarting, StartedAt: r.now().UTC(), Activity: []ActivityEntry{},
	}
	if err := r.checkDecisionsStarter(ctx, starter, workspaceID); err != nil {
		if record {
			r.createFailed(ctx, trail, tgt.title, err.Error())
		}
		return nil, err
	}
	return r.launch(ctx, play, trail, tgt, HarnessChoice{}, record)
}

func (r *Runner) checkDecisionsStarter(ctx context.Context, starter, workspaceID string) error {
	if starter == "" {
		return fmt.Errorf("%w: nobody to run it for: an automation moved the card and the ticket has no developer", apperrs.ErrInvalid)
	}
	if !r.perm.HasPermission(ctx, starter, workspaceID, permissions.PlaysRun, "", "") {
		return fmt.Errorf("%w: %s is required to run the decisions check", apperrs.ErrForbidden, permissions.PlaysRun)
	}
	return nil
}

// ticketMove is the slice of ticket.status_changed the decisions check reads, declared here so plays never imports tickets.
type ticketMove struct {
	Ticket struct {
		ID        string `json:"id"`
		Developer string `json:"developer"`
	} `json:"ticket"`
	From  string `json:"from"`
	To    string `json:"to"`
	Actor struct {
		Kind   string `json:"kind"`
		UserID string `json:"user_id"`
	} `json:"actor"`
}

// onTicketMoved fires the decisions check once per ticket, the first time it enters a done-stage column.
func (r *Runner) onTicketMoved(ctx context.Context, m ticketMove) error {
	entered, err := r.enteredDone(ctx, m.From, m.To)
	if err != nil || !entered {
		return err
	}
	checked, err := r.hasDecisionsCheck(ctx, m.Ticket.ID)
	if err != nil || checked {
		return err
	}
	starter, err := r.decisionsStarter(ctx, m)
	if err != nil {
		return err
	}
	via := ViaWeb
	if strings.HasSuffix(m.Actor.Kind, ":mcp") {
		via = ViaMCP
	}
	runCtx := identity.WithActor(ctx, identity.Actor{ID: starter})
	if _, err := r.startDecisionsCheck(runCtx, m.Ticket.ID, starter, via, true); err != nil {
		r.log.Warn("plays: decisions check did not start", "ticket", m.Ticket.ID, "starter", starter, "error", err)
	}
	return nil
}

// enteredDone reads the stages, never the column names; a move between two done columns is not an entry.
func (r *Runner) enteredDone(ctx context.Context, from, to string) (bool, error) {
	toStatus, err := r.targets.GetStatus(ctx, to)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get status %s: %w", to, err)
	}
	if toStatus.Stage != StageDone {
		return false, nil
	}
	if from == "" {
		return true, nil
	}
	fromStatus, err := r.targets.GetStatus(ctx, from)
	if errors.Is(err, apperrs.ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("get status %s: %w", from, err)
	}
	return fromStatus.Stage != StageDone, nil
}

// hasDecisionsCheck makes a redelivered or repeated entry into done a no-op: done tickets are never reopened (ADR 0064).
func (r *Runner) hasDecisionsCheck(ctx context.Context, ticketID string) (bool, error) {
	trails, err := r.trails.ListTrailsByTarget(ctx, TargetTicket, ticketID)
	if err != nil {
		return false, fmt.Errorf("list trails for ticket %s: %w", ticketID, err)
	}
	for _, t := range trails {
		if t.PlayID == DecisionsCheckPlayID {
			return true, nil
		}
	}
	return false, nil
}

// decisionsStarter is the person who moved the card, or the ticket's developer when an automation (a merged PR) did.
func (r *Runner) decisionsStarter(ctx context.Context, m ticketMove) (string, error) {
	if m.Actor.UserID != "" {
		return m.Actor.UserID, nil
	}
	if m.Ticket.Developer == "" || r.users == nil {
		return "", nil
	}
	id, err := r.users.UserID(ctx, m.Ticket.Developer)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve developer %s: %w", m.Ticket.Developer, err)
	}
	return id, nil
}

// DefaultInstructions returns every built-in instruction text, the seeded plays' and the decisions check's; they name MCP tools.
func DefaultInstructions() []string {
	return []string{fixWithAIInstructions, toTicketsInstructions, interviewInstructions, testWithAIInstructions, decisionsCheckInstructions}
}

package plays

import (
	"context"
	"fmt"
	"strings"
)

// originCharCap bounds the origin's body and doc each, so one hop of context never crowds out the turn's own request.
const originCharCap = 20_000

// LinkedTicket is a ticket at the other end of a found-in or blocked-by link, as the prompt names it.
type LinkedTicket struct {
	Key   string
	Title string
	Done  bool
}

// PullRequest is one PR linked to a bug's origin ticket.
type PullRequest struct {
	Owner  string
	Repo   string
	Number int
	Title  string
	State  string
}

// OriginContext is the one hop a bug's play receives (ADR 0064): the ticket it was found in, that ticket's doc, and its PRs.
type OriginContext struct {
	LinkedTicket
	Body     string
	DocTitle string
	DocBody  string
	PRs      []PullRequest
}

// TicketLinks is what a ticket play's prompt is told about the ticket's found-in and blocked-by links.
type TicketLinks struct {
	Origin        *OriginContext
	OriginUnknown bool
	Blockers      []LinkedTicket
}

// LinkReader is the runner's seam onto tickets' links (ADR 0017); the origin's own origin is never read.
type LinkReader interface {
	TicketLinks(ctx context.Context, ticketID string) (TicketLinks, error)
}

// linkBlocks reads a ticket target's links into prompt blocks; a doc target, or no reader wired, carries none.
func (r *Runner) linkBlocks(ctx context.Context, targetType TargetType, targetID string) ([]string, error) {
	if targetType != TargetTicket || r.links == nil {
		return nil, nil
	}
	links, err := r.links.TicketLinks(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("read links for ticket %s: %w", targetID, err)
	}
	return renderLinkBlocks(links), nil
}

func renderLinkBlocks(l TicketLinks) []string {
	var blocks []string
	if l.Origin != nil {
		blocks = append(blocks, renderOrigin(*l.Origin))
	}
	if l.OriginUnknown {
		blocks = append(blocks, "This bug's origin is unknown: its reporter could not say which ticket it was found in. Do not guess one.")
	}
	if len(l.Blockers) > 0 {
		blocks = append(blocks, renderBlockers(l.Blockers))
	}
	return blocks
}

func renderOrigin(o OriginContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This bug was found in %s %q, a ticket whose work is recorded; fix the bug without rewriting that record. Its context, one hop only:", o.Key, o.Title)
	fmt.Fprintf(&b, "\n\nOrigin ticket body:\n%s", orNone(capChars(o.Body)))
	if o.DocTitle != "" {
		fmt.Fprintf(&b, "\n\nOrigin doc %q:\n%s", o.DocTitle, orNone(capChars(o.DocBody)))
	}
	b.WriteString("\n\nOrigin pull requests:")
	if len(o.PRs) == 0 {
		b.WriteString(" (none)")
	}
	for _, pr := range o.PRs {
		fmt.Fprintf(&b, "\n- %s/%s#%d %q (%s)", pr.Owner, pr.Repo, pr.Number, pr.Title, pr.State)
	}
	return b.String()
}

func renderBlockers(blockers []LinkedTicket) string {
	header := "This ticket was blocked by these tickets, all now done:"
	for _, t := range blockers {
		if !t.Done {
			header = "This ticket is blocked by these tickets; the user chose to run the play before they are all done:"
		}
	}
	var b strings.Builder
	b.WriteString(header)
	for _, t := range blockers {
		state := "not done"
		if t.Done {
			state = "done"
		}
		fmt.Fprintf(&b, "\n- %s %q: %s", t.Key, t.Title, state)
	}
	return b.String()
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(empty)"
	}
	return s
}

func capChars(s string) string {
	if len(s) <= originCharCap {
		return s
	}
	return strings.ToValidUTF8(s[:originCharCap], "") + "\n(trimmed to fit the turn size limit)"
}

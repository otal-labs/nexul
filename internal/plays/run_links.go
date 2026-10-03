package plays

import (
	"context"
	"fmt"
	"strings"
)

// originLine names a bug's origin for the agent to read itself, one hop only (ADR 0064, ADR 0111).
const originLine = "This bug was found in %s %q, a ticket whose work is recorded; fix the bug without rewriting that record. " +
	"Read it with ticket_get for its body, doc, and pull requests, one hop only."

// LinkedTicket is a ticket at the other end of a found-in or blocked-by link, as the prompt names it.
type LinkedTicket struct {
	Key   string
	Title string
	Done  bool
}

// TicketLinks is what a ticket play's prompt is told about the ticket's found-in and blocked-by links.
type TicketLinks struct {
	// Origin is the ticket a bug was found in, named rather than carried.
	Origin        *LinkedTicket
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
		blocks = append(blocks, fmt.Sprintf(originLine, l.Origin.Key, l.Origin.Title))
	}
	if l.OriginUnknown {
		blocks = append(blocks, "This bug's origin is unknown: its reporter could not say which ticket it was found in. Do not guess one.")
	}
	if len(l.Blockers) > 0 {
		blocks = append(blocks, renderBlockers(l.Blockers))
	}
	return blocks
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

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// changeContextReader adapts tickets, docs, workspace, and memories to gitprovider's change-context seam (ADR 0017).
type changeContextReader struct {
	tickets   *tickets.Service
	docs      *docs.Service
	workspace *workspace.Service
	memories  *memories.Service
}

func (a changeContextReader) TicketsForPR(ctx context.Context, owner, repo string, number int) ([]gitprovider.ChangeTicket, error) {
	ts, err := a.tickets.ListByPR(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	out := make([]gitprovider.ChangeTicket, 0, len(ts))
	for _, t := range ts {
		ct, err := a.changeTicket(ctx, t)
		if err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, nil
}

func (a changeContextReader) changeTicket(ctx context.Context, t *tickets.Ticket) (gitprovider.ChangeTicket, error) {
	ct := gitprovider.ChangeTicket{ID: t.ID, Key: t.ID, Title: t.Title, ProjectID: t.ProjectID, BugsFound: []gitprovider.ChangeBug{}}
	if p, err := a.workspace.Get(ctx, t.ProjectID); err == nil {
		ct.Key = fmt.Sprintf("%s-%d", p.Prefix, t.Number)
	}
	if st, err := a.workspace.GetStatus(ctx, string(t.Status)); err == nil {
		ct.Done = st.Kind == workspace.StatusKindDone
	}
	links, err := a.tickets.Links(ctx, t.ID)
	if err != nil {
		return gitprovider.ChangeTicket{}, err
	}
	for _, b := range links.BugsFound {
		ct.BugsFound = append(ct.BugsFound, gitprovider.ChangeBug{ID: b.ID, Key: playsLinkedTicket(b).Key, Title: b.Title, Done: b.Done})
	}
	if t.DocID == "" {
		return ct, nil
	}
	d, err := a.docs.Get(ctx, t.DocID)
	// A doc the caller cannot read, or one since deleted, is left out rather than failing the walk.
	if errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrNotFound) {
		return ct, nil
	}
	if err != nil {
		return gitprovider.ChangeTicket{}, err
	}
	ct.Doc = &gitprovider.ChangeDoc{ID: d.ID, Title: d.Title}
	return ct, nil
}

func (a changeContextReader) DecisionEntries(ctx context.Context, projectID string, refs []string) ([]string, error) {
	entries, err := a.memories.DecisionEntriesCiting(ctx, projectID, refs)
	// A caller without memories:read still gets the PR, tickets, and bugs, just no decisions.
	if errors.Is(err, apperrs.ErrForbidden) {
		return []string{}, nil
	}
	return entries, err
}

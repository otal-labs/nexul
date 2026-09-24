package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// playsTargetReader adapts tickets, docs, and workspace statuses to the runner's TargetReader seam (ADR 0017):
// a ticket's stage is its current column's kind.
type playsTargetReader struct {
	tickets   *tickets.Service
	docs      *docs.Service
	workspace *workspace.Service
}

func (a playsTargetReader) GetTicket(ctx context.Context, id string) (plays.TicketTarget, error) {
	t, err := a.tickets.Get(ctx, id)
	if err != nil {
		return plays.TicketTarget{}, err
	}
	st, err := a.workspace.GetStatus(ctx, string(t.Status))
	if err != nil {
		return plays.TicketTarget{}, err
	}
	key := t.ID
	if p, err := a.workspace.Get(ctx, t.ProjectID); err == nil {
		key = fmt.Sprintf("%s-%d", p.Prefix, t.Number)
	}
	return plays.TicketTarget{ProjectID: t.ProjectID, Key: key, Stage: plays.Stage(st.Kind)}, nil
}

func (a playsTargetReader) GetDoc(ctx context.Context, id string) (plays.DocTarget, error) {
	d, err := a.docs.Get(ctx, id)
	if err != nil {
		return plays.DocTarget{}, err
	}
	return plays.DocTarget{ProjectID: d.ProjectID, Title: d.Title}, nil
}

func (a playsTargetReader) GetStatus(ctx context.Context, id string) (plays.StatusTarget, error) {
	st, err := a.workspace.GetStatus(ctx, id)
	if err != nil {
		return plays.StatusTarget{}, err
	}
	return plays.StatusTarget{Name: st.Name, Stage: plays.Stage(st.Kind)}, nil
}

// playsProjectLookup adds the project read an interview run needs to the workspace lookup plays shares with memories.
type playsProjectLookup struct {
	memoriesProjectLookup
}

func (a playsProjectLookup) GetProject(ctx context.Context, projectID string) (plays.ProjectTarget, error) {
	p, err := a.svc.Get(ctx, projectID)
	if err != nil {
		return plays.ProjectTarget{}, err
	}
	return plays.ProjectTarget{Name: p.Name, TestsLocation: string(p.TestsLocation)}, nil
}

// playsStatusMover adapts tickets' status setter to the runner's move-to seam; an MCP-started run carries the :mcp suffix (ADR 0049).
type playsStatusMover struct {
	svc *tickets.Service
}

func (a playsStatusMover) MoveTicket(ctx context.Context, ticketID, statusID string, actor plays.PlayActor) error {
	kind := tickets.ActorKindPlay
	if actor.Via == plays.ViaMCP {
		kind += ":mcp"
	}
	_, err := a.svc.SetStatusAs(ctx, ticketID, tickets.Status(statusID), tickets.Actor{Kind: kind, PlayLabel: actor.PlayLabel, TrailID: actor.TrailID, UserID: actor.StarterID}, "")
	return err
}

// playsHarnessResolver adapts pairing's ResolveTarget/ResolveTargetOverride to the runner's seam: an empty
// choice resolves the caller's own project link or pairing defaults, same as a chat mention.
type playsHarnessResolver struct {
	svc *pairing.Service
}

func (a playsHarnessResolver) ResolveTarget(ctx context.Context, userID, projectID string, choice plays.HarnessChoice) (plays.HarnessChoice, error) {
	target, err := a.svc.ResolveTargetOverride(ctx, userID, projectID, choice.ComputerID, choice.Provider, choice.Model)
	var nc *pairing.NotConfiguredError
	if errors.As(err, &nc) {
		return plays.HarnessChoice{}, &plays.HarnessRefusal{Reason: string(nc.Reason), ComputerID: nc.ComputerID, Provider: nc.ProviderID, Err: err}
	}
	if err != nil {
		return plays.HarnessChoice{}, err
	}
	return plays.HarnessChoice{ComputerID: target.Computer.ID, Provider: target.Provider, Model: target.Model}, nil
}

// playsMemoryReader adapts memories to the runner's MemoryReader seam, exporting each body to markdown (ADR 0026).
type playsMemoryReader struct {
	svc *memories.Service
}

func (a playsMemoryReader) ListForProject(ctx context.Context, projectID string) ([]plays.Memory, error) {
	ms, err := a.svc.ListForProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]plays.Memory, 0, len(ms))
	for _, m := range ms {
		md, err := richtext.ToMarkdown(m.Body)
		if err != nil {
			return nil, err
		}
		out = append(out, plays.Memory{ID: m.ID, Title: m.Title, Markdown: md, AlwaysIncluded: m.AlwaysIncluded})
	}
	return out, nil
}

// playsThreads adapts chat to the runner's Threads seam (ADR 0017: plays never imports chat).
type playsThreads struct {
	svc *chat.Service
}

func (a playsThreads) GetOrCreateTicketThread(ctx context.Context, workspaceID, ticketID, userID string) (string, error) {
	c, err := a.svc.GetOrCreateTicketThread(ctx, workspaceID, ticketID, userID)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (a playsThreads) GetOrCreateDocThread(ctx context.Context, workspaceID, docID, userID string) (string, error) {
	c, err := a.svc.GetOrCreateDocThread(ctx, workspaceID, docID, userID)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (a playsThreads) GetOrCreateInterviewThread(ctx context.Context, workspaceID, projectID, userID string) (string, error) {
	c, err := a.svc.GetOrCreateInterviewThread(ctx, workspaceID, projectID, userID)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (a playsThreads) PostMessage(ctx context.Context, conversationID, authorID, body string) (string, error) {
	m, err := a.svc.PostMessage(ctx, conversationID, authorID, body)
	if err != nil {
		return "", err
	}
	return m.ID, nil
}

func (a playsThreads) PostSystemNote(ctx context.Context, conversationID, viaUserID, body string) error {
	_, err := a.svc.PostSystemMessage(ctx, conversationID, viaUserID, body)
	return err
}

// playsLinkReader adapts tickets and docs to the runner's LinkReader seam: one hop, so the origin's own found-in is never read.
type playsLinkReader struct {
	tickets *tickets.Service
	docs    agentDocReader
}

func (a playsLinkReader) TicketLinks(ctx context.Context, id string) (plays.TicketLinks, error) {
	set, err := a.tickets.Links(ctx, id)
	if err != nil {
		return plays.TicketLinks{}, err
	}
	out := plays.TicketLinks{OriginUnknown: set.OriginUnknown}
	for _, b := range set.BlockedBy {
		out.Blockers = append(out.Blockers, playsLinkedTicket(b))
	}
	if set.FoundIn == nil {
		return out, nil
	}
	origin, err := a.origin(ctx, *set.FoundIn)
	if err != nil {
		return plays.TicketLinks{}, err
	}
	out.Origin = origin
	return out, nil
}

func (a playsLinkReader) origin(ctx context.Context, linked tickets.LinkedTicket) (*plays.OriginContext, error) {
	t, err := a.tickets.Get(ctx, linked.ID)
	if err != nil {
		return nil, err
	}
	prs, _, err := a.tickets.ListLinks(ctx, linked.ID)
	if err != nil {
		return nil, err
	}
	o := &plays.OriginContext{LinkedTicket: playsLinkedTicket(linked), Body: t.Body}
	for _, pr := range prs {
		o.PRs = append(o.PRs, plays.PullRequest{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number, Title: pr.Title, State: string(pr.State)})
	}
	if t.DocID == "" {
		return o, nil
	}
	d, err := a.docs.Get(ctx, t.DocID)
	// A doc the starter cannot read, or one since deleted, is left out rather than failing the run.
	if errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrNotFound) {
		return o, nil
	}
	if err != nil {
		return nil, err
	}
	o.DocTitle, o.DocBody = d.Title, d.BodyMarkdown
	return o, nil
}

func playsLinkedTicket(t tickets.LinkedTicket) plays.LinkedTicket {
	key := t.ID
	if t.Prefix != "" {
		key = fmt.Sprintf("%s-%d", t.Prefix, t.Number)
	}
	return plays.LinkedTicket{Key: key, Title: t.Title, Done: t.Done}
}

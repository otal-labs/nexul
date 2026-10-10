package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
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
	prefix := ""
	if p, err := a.workspace.Get(ctx, t.ProjectID); err == nil {
		prefix = p.Prefix
	}
	return plays.TicketTarget{ProjectID: t.ProjectID, Key: ticketKey(prefix, t.Number, t.ID), Stage: plays.Stage(st.Kind)}, nil
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

// projectTargets lists a project's tickets or docs through their own use-cases, which check the read.
type projectTargets struct {
	tickets *tickets.Service
	docs    *docs.Service
}

func (p projectTargets) TicketIDs(ctx context.Context, projectID string) ([]string, error) {
	ts, err := p.tickets.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(ts))
	for i, t := range ts {
		ids[i] = t.ID
	}
	return ids, nil
}

func (p projectTargets) TargetIDs(ctx context.Context, targetType plays.TargetType, projectID string) ([]string, error) {
	if targetType == plays.TargetTicket {
		return p.TicketIDs(ctx, projectID)
	}
	if targetType != plays.TargetDoc {
		return nil, fmt.Errorf("%w: project_id takes target_type ticket or doc", apperrs.ErrInvalid)
	}
	ds, err := p.docs.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(ds))
	for i, d := range ds {
		ids[i] = d.ID
	}
	return ids, nil
}

// playsProjectLookup adds the project read an interview run needs to the workspace lookup plays shares with memories.
type playsProjectLookup struct {
	memoriesProjectLookup
}

func (a playsProjectLookup) GetProject(ctx context.Context, projectID string) (plays.ProjectTarget, error) {
	p, err := a.projects.Get(ctx, projectID)
	if err != nil {
		return plays.ProjectTarget{}, err
	}
	return plays.ProjectTarget{Name: p.Name, TestsLocation: string(p.TestsLocation)}, nil
}

// playsHarnessResolver preserves fallback for unattended runs and requires location for a person's run (ADR 0145).
type playsHarnessResolver struct {
	svc *pairing.Service
}

func (a playsHarnessResolver) ResolveTarget(ctx context.Context, userID, projectID string, choice plays.HarnessChoice) (plays.HarnessChoice, error) {
	return toHarnessChoice(a.svc.ResolveTargetOverride(ctx, userID, projectID, choice.ComputerID, choice.Provider, choice.Model, choice.ModelOptions))
}

func (a playsHarnessResolver) ResolvePersonTarget(ctx context.Context, userID, projectID string, choice plays.HarnessChoice) (plays.HarnessChoice, error) {
	return toHarnessChoice(a.svc.ResolvePersonRun(ctx, userID, projectID, choice.ComputerID, choice.HarnessProjectID, choice.Provider, choice.Model, choice.ModelOptions))
}

func toHarnessChoice(target *pairing.ResolvedTarget, err error) (plays.HarnessChoice, error) {
	var nc *pairing.NotConfiguredError
	if errors.As(err, &nc) {
		return plays.HarnessChoice{}, &plays.HarnessRefusal{Reason: string(nc.Reason), ComputerID: nc.ComputerID, Provider: nc.ProviderID, Err: err}
	}
	if err != nil {
		return plays.HarnessChoice{}, err
	}
	return plays.HarnessChoice{ComputerID: target.Computer.ID, Provider: target.Provider, Model: target.Model, ModelOptions: target.ModelOptions}, nil
}

// playsMemoryReader adapts memories to the runner's MemoryReader seam.
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
		out = append(out, plays.Memory{ID: m.ID, Title: m.Title, AlwaysIncluded: m.AlwaysIncluded, Interview: m.Kind == memories.KindInterview, Footer: m.Footer})
	}
	return out, nil
}

// playsInterviewAnswers adapts the memories domain's stored answers to the runner's InterviewAnswers seam.
type playsInterviewAnswers struct {
	svc *memories.Service
}

func (a playsInterviewAnswers) RecordRound(ctx context.Context, projectID, answeredBy string, followUps []plays.FollowUp) error {
	rows := make([]memories.InterviewAnswer, 0, len(followUps))
	for _, f := range followUps {
		options := make([]memories.AnswerOption, 0, len(f.Options))
		for _, o := range f.Options {
			options = append(options, memories.AnswerOption{Label: o.Label, Description: o.Description})
		}
		rows = append(rows, memories.InterviewAnswer{
			Question: f.Question, Why: f.Why, Options: options, MultiSelect: f.MultiSelect, Selected: f.Selected, Text: f.Text,
		})
	}
	_, err := a.svc.RecordRound(ctx, projectID, answeredBy, rows)
	return err
}

func (a playsInterviewAnswers) ListSources(ctx context.Context, projectID string) ([]plays.InterviewSource, error) {
	srcs, err := a.svc.ListSources(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]plays.InterviewSource, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, plays.InterviewSource{ID: s.ID, Kind: s.Kind, Ref: s.Ref, Label: s.Label, Stance: s.Stance, Unreadable: s.NotVisible || s.Gone})
	}
	return out, nil
}

func (a playsInterviewAnswers) ClearSuggestions(ctx context.Context, projectID string) error {
	return a.svc.ClearSuggestions(ctx, projectID)
}

// playsCheckouts adapts pairing's project links and computer project lists to the runner's Checkouts seam.
type playsCheckouts struct {
	svc *pairing.Service
}

func (a playsCheckouts) LinkedProject(ctx context.Context, userID, projectID string) (string, string, error) {
	link, err := a.svc.GetProjectLink(ctx, userID, projectID)
	if err != nil {
		return "", "", err
	}
	return link.ComputerID, link.HarnessProjectID, nil
}

func (a playsCheckouts) ListProjects(ctx context.Context, userID, computerID string) ([]harness.Project, error) {
	return a.svc.ListProjects(ctx, userID, computerID)
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

func (a playsThreads) PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) error {
	_, err := a.svc.PostSystemMessage(ctx, conversationID, viaUserID, body)
	return err
}

// playsLinkReader adapts tickets to the runner's LinkReader seam: one hop, so the origin's own found-in is never read.
type playsLinkReader struct {
	tickets *tickets.Service
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
	if set.FoundIn != nil {
		origin := playsLinkedTicket(*set.FoundIn)
		out.Origin = &origin
	}
	return out, nil
}

func playsLinkedTicket(t tickets.LinkedTicket) plays.LinkedTicket {
	return plays.LinkedTicket{Key: ticketKey(t.Prefix, t.Number, t.ID), Title: t.Title, Done: t.Done}
}

// playsFacts reads a ticket's or doc's fields for auto play conditions straight from storage: matching runs in the
// background as nobody, and the queue checks the person a run lands on itself.
type playsFacts struct {
	store *storage.Store
}

func (a playsFacts) Facts(ctx context.Context, targetType plays.TargetType, id string) (plays.Facts, error) {
	if targetType == plays.TargetDoc {
		d, err := a.store.Docs.GetByID(ctx, id)
		if err != nil {
			return plays.Facts{}, err
		}
		return plays.Facts{ProjectID: d.ProjectID, FolderID: d.FolderID, Archived: d.Archived}, nil
	}
	t, err := a.store.Tickets.GetByID(ctx, id)
	if err != nil {
		return plays.Facts{}, err
	}
	f := plays.Facts{ProjectID: t.ProjectID, Status: string(t.Status), Labels: t.Labels, SourceDoc: t.DocID != ""}
	if f.Type, err = optional(a.store.TicketTypes.TypeName(ctx, t.TypeID)); err != nil {
		return plays.Facts{}, err
	}
	if st, err := a.store.Statuses.Get(ctx, string(t.Status)); err == nil {
		f.Stage = plays.Stage(st.Kind)
	}
	if c, err := a.store.Categories.Get(ctx, t.CategoryID); t.CategoryID != "" && err == nil {
		f.Category = c.Name
	}
	if f.Developer, err = a.userID(ctx, t.Developer); err != nil {
		return plays.Facts{}, err
	}
	if f.Tester, err = a.userID(ctx, t.Tester); err != nil {
		return plays.Facts{}, err
	}
	prs, err := a.store.Tickets.ListPRLinks(ctx, id)
	if err != nil {
		return plays.Facts{}, err
	}
	blockers, err := a.store.Tickets.UnclearedBlockersOf(ctx, []string{id})
	if err != nil {
		return plays.Facts{}, err
	}
	f.LinkedPR, f.Blocked = len(prs) > 0, len(blockers[id]) > 0
	return f, nil
}

// userID resolves a member login to its user id; a login no account holds any more is nobody.
func (a playsFacts) userID(ctx context.Context, login string) (string, error) {
	if login == "" {
		return "", nil
	}
	u, err := a.store.Users.GetUserByLogin(ctx, login)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

// optional reads a lookup whose row may be gone as empty.
func optional(v string, err error) (string, error) {
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	return v, err
}

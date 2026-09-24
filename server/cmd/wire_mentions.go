package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

// searchMentionHits runs an FTS query and maps rows onto SearchHit, the only flow shared across sources.
func searchMentionHits[T any](ctx context.Context, search func(context.Context, string, int) ([]T, error), query string, limit int, project func(T) mentions.SearchHit) ([]mentions.SearchHit, error) {
	hits, err := search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]mentions.SearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, project(h))
	}
	return out, nil
}

// mentionTicketSource adapts tickets to mentions.TicketSource so mentions never imports tickets (ADR 0017).
type mentionTicketSource struct {
	repo *storage.TicketsRepo
}

func (a mentionTicketSource) GetByID(ctx context.Context, id string) (*mentions.Ticket, error) {
	t, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ticketToMentionTicket(t), nil
}

func (a mentionTicketSource) Search(ctx context.Context, query string, limit int) ([]mentions.SearchHit, error) {
	return searchMentionHits(ctx, a.repo.Search, query, limit, func(h tickets.SearchResult) mentions.SearchHit {
		return mentions.SearchHit{ID: h.ID, Title: h.Title}
	})
}

// GetByKey resolves the @PREFIX-NUMBER mention path via the ticket's display id (ADR 0004).
func (a mentionTicketSource) GetByKey(ctx context.Context, prefix string, number int) (*mentions.Ticket, error) {
	t, err := a.repo.GetByPrefixAndNumber(ctx, prefix, number)
	if err != nil {
		return nil, err
	}
	return ticketToMentionTicket(t), nil
}

// ticketToMentionTicket projects a Ticket onto mentions.Ticket for the mention-chip layout template.
func ticketToMentionTicket(t *tickets.Ticket) *mentions.Ticket {
	return &mentions.Ticket{
		ID:        t.ID,
		Title:     t.Title,
		Status:    string(t.Status),
		Number:    t.Number,
		ProjectID: t.ProjectID,
		TypeID:    t.TypeID,
		Developer: t.Developer,
	}
}

// mentionDocSource adapts docs to mentions.DocSource; reads are repo-level so chips show a title the actor can't open.
type mentionDocSource struct {
	repo *storage.DocsRepo
}

func (a mentionDocSource) GetByID(ctx context.Context, id string) (*mentions.Doc, error) {
	d, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &mentions.Doc{ID: d.ID, Title: d.Title}, nil
}

func (a mentionDocSource) Search(ctx context.Context, query string, limit int) ([]mentions.SearchHit, error) {
	return searchMentionHits(ctx, a.repo.Search, query, limit, func(h docs.SearchResult) mentions.SearchHit {
		return mentions.SearchHit{ID: h.ID, Title: h.Title}
	})
}

// mentionStatusSource adapts statuses to mentions.StatusSource so chips show the configurable column name.
type mentionStatusSource struct {
	repo *storage.StatusesRepo
}

func (a mentionStatusSource) Get(ctx context.Context, id string) (*mentions.Status, error) {
	st, err := a.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &mentions.Status{ID: st.ID, Name: st.Name}, nil
}

// mentionProjectSource adapts projects to mentions.ProjectSource so chips render {ticket.Project} as PREFIX-NUMBER.
type mentionProjectSource struct {
	repo *storage.ProjectsRepo
}

func (a mentionProjectSource) Get(ctx context.Context, id string) (*mentions.Project, error) {
	p, err := a.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &mentions.Project{ID: p.ID, Prefix: p.Prefix}, nil
}

// mentionTicketTypeSource adapts ticket-types to mentions.TicketTypeSource so chips can render {ticket.Type}.
type mentionTicketTypeSource struct {
	repo *storage.TicketTypesRepo
}

func (a mentionTicketTypeSource) Get(ctx context.Context, id string) (*mentions.TicketType, error) {
	tt, err := a.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &mentions.TicketType{ID: tt.ID, Name: tt.Name}, nil
}

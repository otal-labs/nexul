package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// ListLinkEnds returns the links a ticket holds and the links other tickets hold against it.
func (r *TicketsRepo) ListLinkEnds(ctx context.Context, id string) ([]tickets.LinkEnd, []tickets.LinkEnd, error) {
	fromRows, err := r.q.ListTicketLinksFrom(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list links from ticket %s: %w", id, err)
	}
	toRows, err := r.q.ListTicketLinksTo(ctx, nullString(id))
	if err != nil {
		return nil, nil, fmt.Errorf("list links to ticket %s: %w", id, err)
	}
	from := make([]tickets.LinkEnd, 0, len(fromRows))
	for _, row := range fromRows {
		end := tickets.LinkEnd{Kind: tickets.LinkKind(row.Kind)}
		if row.TargetID.Valid {
			end.Ticket = &tickets.LinkedTicket{
				ID: row.TargetID.String, ProjectID: row.ProjectID.String, Prefix: row.Prefix.String,
				Number: int(row.Number.Int64), Title: row.Title.String, Status: tickets.Status(row.Status.String), Done: isDoneStage(row.Stage),
			}
		}
		from = append(from, end)
	}
	to := make([]tickets.LinkEnd, 0, len(toRows))
	for _, row := range toRows {
		to = append(to, tickets.LinkEnd{Kind: tickets.LinkKind(row.Kind), Ticket: &tickets.LinkedTicket{
			ID: row.TicketID, ProjectID: row.ProjectID.String, Prefix: row.Prefix.String,
			Number: int(row.Number), Title: row.Title, Status: tickets.Status(row.Status), Done: isDoneStage(row.Stage),
		}})
	}
	return from, to, nil
}

// BlockerIDs returns the ids a ticket is directly blocked by.
func (r *TicketsRepo) BlockerIDs(ctx context.Context, id string) ([]string, error) {
	rows, err := r.q.ListTicketBlockerIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list blockers of ticket %s: %w", id, err)
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.String)
	}
	return out, nil
}

// PutLink inserts a link; a found_in link first clears the ticket's earlier found-in in the same transaction.
func (r *TicketsRepo) PutLink(ctx context.Context, link tickets.TicketLink, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if link.Kind == tickets.LinkFoundIn {
			if _, err := q.DeleteTicketFoundIn(ctx, link.TicketID); err != nil {
				return fmt.Errorf("clear found-in on ticket %s: %w", link.TicketID, err)
			}
		}
		err := q.InsertTicketLink(ctx, sqlcgen.InsertTicketLinkParams{
			TicketID: link.TicketID, Kind: string(link.Kind), TargetID: nullStringOrNil(link.TargetID), CreatedAt: link.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert %s link on ticket %s: %w", link.Kind, link.TicketID, classifyWriteErr(err))
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
}

// DeleteLink enqueues evts only when a row went away, so removing a missing link publishes nothing.
func (r *TicketsRepo) DeleteLink(ctx context.Context, link tickets.TicketLink, evts ...eventbus.OutboxEvent) (bool, error) {
	var removed bool
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		var n int64
		var err error
		if link.Kind == tickets.LinkFoundIn {
			n, err = q.DeleteTicketFoundIn(ctx, link.TicketID)
		}
		if link.Kind == tickets.LinkBlockedBy {
			n, err = q.DeleteTicketBlocker(ctx, sqlcgen.DeleteTicketBlockerParams{TicketID: link.TicketID, TargetID: nullString(link.TargetID)})
		}
		if err != nil {
			return fmt.Errorf("delete %s link on ticket %s: %w", link.Kind, link.TicketID, err)
		}
		removed = n > 0
		if !removed {
			return nil
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
	return removed, err
}

// UnclearedBlockers maps each blocked ticket id to its blockers whose status is not in the done stage.
func (r *TicketsRepo) UnclearedBlockers(ctx context.Context) (map[string][]tickets.LinkedTicket, error) {
	rows, err := r.q.ListUnclearedTicketBlockers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list uncleared blockers: %w", err)
	}
	out := make(map[string][]tickets.LinkedTicket)
	for _, row := range rows {
		out[row.BlockedID] = append(out[row.BlockedID], tickets.LinkedTicket{
			ID: row.ID, ProjectID: row.ProjectID.String, Prefix: row.Prefix.String,
			Number: int(row.Number), Title: row.Title, Status: tickets.Status(row.Status),
		})
	}
	return out, nil
}

func isDoneStage(stage sql.NullString) bool {
	return stage.String == string(workspace.StatusKindDone)
}

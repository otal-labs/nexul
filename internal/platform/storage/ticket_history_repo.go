package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tickets"
)

// writeTicketChanges stores a save's changes and prunes the ticket's list to its newest KeptChanges; none writes nothing.
func writeTicketChanges(ctx context.Context, q *sqlcgen.Queries, ticketID string, changes []tickets.Change) error {
	if len(changes) == 0 {
		return nil
	}
	for _, c := range changes {
		actor, err := json.Marshal(c.Actor)
		if err != nil {
			return fmt.Errorf("encode actor of ticket %s change: %w", ticketID, err)
		}
		err = q.InsertTicketChange(ctx, sqlcgen.InsertTicketChangeParams{
			ID: c.ID, TicketID: ticketID, Field: c.Field, FromValue: c.From, ToValue: c.To, Actor: string(actor), CreatedAt: c.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert %s change of ticket %s: %w", c.Field, ticketID, classifyWriteErr(err))
		}
	}
	if err := q.PruneTicketChanges(ctx, sqlcgen.PruneTicketChangesParams{ForTicket: ticketID, Keep: tickets.KeptChanges}); err != nil {
		return fmt.Errorf("prune changes of ticket %s: %w", ticketID, err)
	}
	return nil
}

// writeDescriptionVersion appends the body as the ticket's next version unless the newest version already holds it, then prunes.
func writeDescriptionVersion(ctx context.Context, q *sqlcgen.Queries, ticketID, body string, editors []string, via string, at time.Time) error {
	same, err := q.TicketDescriptionMatchesLatest(ctx, sqlcgen.TicketDescriptionMatchesLatestParams{ForTicket: ticketID, WantBody: body})
	if err != nil {
		return fmt.Errorf("compare ticket %s with its newest description version: %w", ticketID, err)
	}
	if same {
		return nil
	}
	err = q.InsertTicketDescriptionVersion(ctx, sqlcgen.InsertTicketDescriptionVersionParams{
		ForTicket: ticketID, Body: body, EditorIds: idsJSON(editors), Via: via, CreatedAt: at.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert description version of ticket %s: %w", ticketID, classifyWriteErr(err))
	}
	if err := q.PruneTicketDescriptionVersions(ctx, sqlcgen.PruneTicketDescriptionVersionsParams{ForTicket: ticketID, Keep: tickets.KeptDescriptionVersions}); err != nil {
		return fmt.Errorf("prune description versions of ticket %s: %w", ticketID, err)
	}
	return nil
}

// recordCreated writes a new ticket's created change and its description as version 1, under the actor ctx names.
func recordCreated(ctx context.Context, q *sqlcgen.Queries, t *tickets.Ticket) error {
	actor := tickets.ActorOf(ctx)
	change := tickets.Change{ID: ids.New(), Field: tickets.FieldCreated, Actor: actor, CreatedAt: t.CreatedAt}
	if err := writeTicketChanges(ctx, q, t.ID, []tickets.Change{change}); err != nil {
		return err
	}
	return writeDescriptionVersion(ctx, q, t.ID, t.Body, soleEditor(actor.UserID), tickets.ViaOf(ctx), t.CreatedAt)
}

func (r *TicketsRepo) ListDescriptionVersions(ctx context.Context, id string) ([]tickets.VersionSummary, error) {
	rows, err := r.q.ListTicketDescriptionVersionSummaries(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list description versions of ticket %s: %w", id, err)
	}
	out := make([]tickets.VersionSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, tickets.VersionSummary{
			Version: int(row.Version), CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), EditorIDs: decodeEditorIDs(row.EditorIds), Via: row.Via,
		})
	}
	return out, nil
}

func (r *TicketsRepo) GetDescriptionVersion(ctx context.Context, id string, version int) (*tickets.DescriptionVersion, error) {
	row, err := r.q.GetTicketDescriptionVersion(ctx, sqlcgen.GetTicketDescriptionVersionParams{TicketID: id, Version: int64(version)})
	if err != nil {
		return nil, fmt.Errorf("get description version %d of ticket %s: %w", version, id, notFoundIfNoRows(err))
	}
	return &tickets.DescriptionVersion{
		TicketID: row.TicketID, Version: int(row.Version), Body: row.Body, EditorIDs: decodeEditorIDs(row.EditorIds), Via: row.Via,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}, nil
}

func (r *TicketsRepo) ListChanges(ctx context.Context, id string) ([]tickets.Change, error) {
	rows, err := r.q.ListTicketChanges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list changes of ticket %s: %w", id, err)
	}
	out := make([]tickets.Change, 0, len(rows))
	for _, row := range rows {
		var actor tickets.Actor
		if err := json.Unmarshal([]byte(row.Actor), &actor); err != nil {
			return nil, fmt.Errorf("decode actor of ticket %s change %s: %w", id, row.ID, err)
		}
		out = append(out, tickets.Change{
			ID: row.ID, Field: row.Field, From: row.FromValue, To: row.ToValue, Actor: actor, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

// AppendSittingVersion writes the ticket's current body as a version crediting editorIDs, unless the newest version
// already holds it; a ticket's body has no version counter of its own, so versions number from the newest kept.
func (r *TicketsRepo) AppendSittingVersion(ctx context.Context, id string, editorIDs []string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetTicket(ctx, id)
		if err != nil {
			return fmt.Errorf("sitting version of ticket %s: %w", id, notFoundIfNoRows(err))
		}
		return writeDescriptionVersion(ctx, q, id, row.Ticket.Body, editorIDs, "", at)
	})
}

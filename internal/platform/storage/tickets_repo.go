package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tickets"
)

var _ tickets.Repo = (*TicketsRepo)(nil)

type TicketsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Create appends the ticket to its (status, category) order and assigns the next number in one transaction.
func (r *TicketsRepo) Create(ctx context.Context, t *tickets.Ticket, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		pos, err := nextTicketPosition(ctx, tx, string(t.Status), t.CategoryID)
		if err != nil {
			return err
		}
		num, err := nextTicketNumber(ctx, tx, t.ProjectID)
		if err != nil {
			return err
		}
		err = q.CreateTicket(ctx, sqlcgen.CreateTicketParams{
			ID:                     t.ID,
			Title:                  t.Title,
			Body:                   t.Body,
			Status:                 string(t.Status),
			Position:               int64(pos),
			Number:                 int64(num),
			DocID:                  sql.NullString{String: t.DocID, Valid: t.DocID != ""},
			ProjectID:              sql.NullString{String: t.ProjectID, Valid: t.ProjectID != ""},
			CategoryID:             sql.NullString{String: t.CategoryID, Valid: t.CategoryID != ""},
			TypeID:                 sql.NullString{String: defaultType(t.TypeID), Valid: true},
			Developer:              t.Developer,
			Tester:                 t.Tester,
			ReporterKind:           t.Reporter.Kind,
			ReporterLogin:          t.Reporter.Login,
			ReporterAutomationID:   t.Reporter.AutomationID,
			ReporterAutomationName: t.Reporter.AutomationName,
			CreatedAt:              t.CreatedAt.Unix(),
			UpdatedAt:              t.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert ticket %s: %w", t.ID, classifyWriteErr(err))
		}
		if err := insertTicketLabels(ctx, tx, t.ID, t.Labels); err != nil {
			return err
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
}

func (r *TicketsRepo) GetByID(ctx context.Context, id string) (*tickets.Ticket, error) {
	row, err := r.q.GetTicket(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ticket %s: %w", id, notFoundIfNoRows(err))
	}
	t := toTicket(row)
	labels, err := r.ListLabels(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ticket %s: %w", id, err)
	}
	t.Labels = labels
	return t, nil
}

// GetByPrefixAndNumber resolves a ticket via its PREFIX-NUMBER display id, used by mentions' key resolution.
func (r *TicketsRepo) GetByPrefixAndNumber(ctx context.Context, prefix string, number int) (*tickets.Ticket, error) {
	row, err := r.q.GetTicketByPrefixAndNumber(ctx, sqlcgen.GetTicketByPrefixAndNumberParams{Prefix: prefix, Number: int64(number)})
	if err != nil {
		return nil, fmt.Errorf("get ticket by key %s-%d: %w", prefix, number, notFoundIfNoRows(err))
	}
	t := toTicket(row)
	labels, err := r.ListLabels(ctx, t.ID)
	if err != nil {
		return nil, fmt.Errorf("get ticket by key %s-%d: %w", prefix, number, err)
	}
	t.Labels = labels
	return t, nil
}

func (r *TicketsRepo) List(ctx context.Context) ([]*tickets.Ticket, error) {
	rows, err := r.q.ListTickets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	out := toTickets(rows)
	if err := r.attachLabels(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TicketsRepo) ListByDoc(ctx context.Context, docID string) ([]*tickets.Ticket, error) {
	rows, err := r.q.ListTicketsByDoc(ctx, nullString(docID))
	if err != nil {
		return nil, fmt.Errorf("list tickets for doc %s: %w", docID, err)
	}
	out := toTickets(rows)
	if err := r.attachLabels(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TicketsRepo) ListByProject(ctx context.Context, projectID string) ([]*tickets.Ticket, error) {
	rows, err := r.q.ListTicketsByProject(ctx, nullString(projectID))
	if err != nil {
		return nil, fmt.Errorf("list tickets for project %s: %w", projectID, err)
	}
	out := toTickets(rows)
	if err := r.attachLabels(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateStatus moves a ticket to a new status, appending it rather than carrying the old position over.
func (r *TicketsRepo) UpdateStatus(ctx context.Context, id string, status tickets.Status, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetTicketStatusAndCategory(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("update ticket %s status: %w", id, apperrs.ErrNotFound)
			}
			return fmt.Errorf("update ticket %s status: %w", id, err)
		}
		pos, err := nextTicketPosition(ctx, tx, string(status), row.CategoryID.String)
		if err != nil {
			return fmt.Errorf("update ticket %s status: %w", id, err)
		}
		if _, err := q.UpdateTicketStatus(ctx, sqlcgen.UpdateTicketStatusParams{
			Status: string(status), Position: int64(pos), UpdatedAt: time.Now().Unix(), ID: id,
		}); err != nil {
			return fmt.Errorf("update ticket %s status: %w", id, err)
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
}

// SetPosition updates a ticket's manual ordering within its current (status, category) pair; the ticket must exist.
func (r *TicketsRepo) SetPosition(ctx context.Context, id string, position int) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetTicketPosition(ctx, sqlcgen.SetTicketPositionParams{
			Position: int64(position), UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set position on ticket %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set position on ticket %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// nextTicketPosition must run in the same transaction as the row it positions, so writers can't race on the max.
func nextTicketPosition(ctx context.Context, tx *sql.Tx, status, categoryID string) (int, error) {
	max, err := sqlcgen.New(tx).MaxTicketPositionInPair(ctx, sqlcgen.MaxTicketPositionInPairParams{
		Status:     status,
		CategoryID: sql.NullString{String: categoryID, Valid: categoryID != ""},
	})
	if err != nil {
		return 0, fmt.Errorf("compute next position: %w", err)
	}
	n, ok, err := optionalInt(max)
	if err != nil {
		return 0, fmt.Errorf("compute next position: %w", err)
	}
	if !ok {
		return 0, nil
	}
	return int(n) + 1, nil
}

// nextTicketNumber must run in the same transaction as the row it numbers, so writers can't race on the max.
func nextTicketNumber(ctx context.Context, tx *sql.Tx, projectID string) (int, error) {
	max, err := sqlcgen.New(tx).MaxTicketNumberInProject(ctx, sql.NullString{String: projectID, Valid: projectID != ""})
	if err != nil {
		return 0, fmt.Errorf("compute next ticket number: %w", err)
	}
	n, ok, err := optionalInt(max)
	if err != nil {
		return 0, fmt.Errorf("compute next ticket number: %w", err)
	}
	if !ok {
		return 1, nil
	}
	return int(n) + 1, nil
}

func (r *TicketsRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteTicket(ctx, id)
		if err != nil {
			return fmt.Errorf("delete ticket %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete ticket %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
}

func (r *TicketsRepo) LinkPR(ctx context.Context, id string, ref tickets.PRRef, state tickets.PRState) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).LinkTicketPR(ctx, sqlcgen.LinkTicketPRParams{
			TicketID: id, PrOwner: ref.Owner, PrRepo: ref.Repo, PrNumber: int64(ref.Number),
			PrTitle: ref.Title, PrSha: ref.SHA, PrState: string(state), LinkedAt: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("link pr to ticket %s: %w", id, err)
		}
		return nil
	})
}

func (r *TicketsRepo) ListPRLinks(ctx context.Context, id string) ([]tickets.PRLink, error) {
	rows, err := r.q.ListTicketPRLinks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list pr links for ticket %s: %w", id, err)
	}
	out := make([]tickets.PRLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, tickets.PRLink{
			PRRef: tickets.PRRef{Owner: row.PrOwner, Repo: row.PrRepo, Number: int(row.PrNumber), Title: row.PrTitle, SHA: row.PrSha},
			State: tickets.PRState(row.PrState),
		})
	}
	return out, nil
}

func (r *TicketsRepo) ListPRLinksBatch(ctx context.Context, ids []string) (map[string][]tickets.PRLink, error) {
	out := make(map[string][]tickets.PRLink, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.q.ListTicketPRLinksBatch(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list pr links for tickets: %w", err)
	}
	for _, row := range rows {
		out[row.TicketID] = append(out[row.TicketID], tickets.PRLink{
			PRRef: tickets.PRRef{Owner: row.PrOwner, Repo: row.PrRepo, Number: int(row.PrNumber), Title: row.PrTitle, SHA: row.PrSha},
			State: tickets.PRState(row.PrState),
		})
	}
	return out, nil
}

func (r *TicketsRepo) MarkPRState(ctx context.Context, owner, repo string, number int, state tickets.PRState) ([]string, error) {
	var affected []string
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		ids, err := q.ListTicketIDsByPR(ctx, sqlcgen.ListTicketIDsByPRParams{PrOwner: owner, PrRepo: repo, PrNumber: int64(number)})
		if err != nil {
			return err
		}
		affected = ids
		return q.UpdateTicketPRState(ctx, sqlcgen.UpdateTicketPRStateParams{
			PrState: string(state), PrOwner: owner, PrRepo: repo, PrNumber: int64(number),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("mark pr %s/%s#%d as %s: %w", owner, repo, number, state, err)
	}
	return affected, nil
}

func (r *TicketsRepo) SetFinishedAt(ctx context.Context, id string, at time.Time, evts ...eventbus.OutboxEvent) (bool, error) {
	var set bool
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.SetTicketFinishedAt(ctx, sqlcgen.SetTicketFinishedAtParams{
			FinishedAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, ID: id,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		set = true
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
	if err != nil {
		return false, fmt.Errorf("finish ticket %s: %w", id, err)
	}
	return set, nil
}

func (r *TicketsRepo) LinkBranch(ctx context.Context, id string, link tickets.BranchLink) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).LinkTicketBranch(ctx, sqlcgen.LinkTicketBranchParams{
			TicketID: id, BranchOwner: link.Owner, BranchRepo: link.Repo, BranchName: link.Branch, LinkedAt: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("link branch to ticket %s: %w", id, err)
		}
		return nil
	})
}

func (r *TicketsRepo) ListBranchLinks(ctx context.Context, id string) ([]tickets.BranchLink, error) {
	rows, err := r.q.ListTicketBranchLinks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list branch links for ticket %s: %w", id, err)
	}
	out := make([]tickets.BranchLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, tickets.BranchLink{Owner: row.BranchOwner, Repo: row.BranchRepo, Branch: row.BranchName})
	}
	return out, nil
}

// UpdateType changes a ticket's type id in place; the type FK guarantees only a real type id is referenced.
func (r *TicketsRepo) UpdateType(ctx context.Context, id, typeID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateTicketType(ctx, sqlcgen.UpdateTicketTypeParams{
			TypeID: sql.NullString{String: typeID, Valid: true}, UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("update ticket %s type: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update ticket %s type: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// UpdatePerson sets a ticket's developer or tester, enqueueing evts in the same transaction; empty clears it.
func (r *TicketsRepo) UpdatePerson(ctx context.Context, id string, role tickets.Role, login string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		now := time.Now().Unix()
		var n int64
		var err error
		switch role {
		case tickets.RoleTester:
			n, err = q.UpdateTicketTester(ctx, sqlcgen.UpdateTicketTesterParams{Tester: login, UpdatedAt: now, ID: id})
		default:
			n, err = q.UpdateTicketDeveloper(ctx, sqlcgen.UpdateTicketDeveloperParams{Developer: login, UpdatedAt: now, ID: id})
		}
		if err != nil {
			return fmt.Errorf("update ticket %s %s: %w", id, role, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update ticket %s %s: %w", id, role, apperrs.ErrNotFound)
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
}

// UpdateTicket edits a ticket's title and body in place, enqueueing the
// given outbox events in the same transaction.
func (r *TicketsRepo) UpdateTicket(ctx context.Context, id, title, body string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateTicketTitleBody(ctx, sqlcgen.UpdateTicketTitleBodyParams{
			Title: title, Body: body, UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("update ticket %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update ticket %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueTicketsOutbox(ctx, tx, evts)
	})
}

// moveTicketCategory runs in an existing transaction; used by SetTicketCategory to write the shared row.
func moveTicketCategory(ctx context.Context, tx *sql.Tx, ticketID, categoryID string) error {
	q := sqlcgen.New(tx)
	row, err := q.GetTicketStatusAndCategory(ctx, ticketID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrs.ErrNotFound
		}
		return err
	}
	// A move to the current category is a no-op, mirroring tickets.Service.transition's same-status guard.
	if row.CategoryID.String == categoryID {
		return nil
	}
	pos, err := nextTicketPosition(ctx, tx, row.Status, categoryID)
	if err != nil {
		return err
	}
	if _, err := q.MoveTicketCategory(ctx, sqlcgen.MoveTicketCategoryParams{
		CategoryID: sql.NullString{String: categoryID, Valid: categoryID != ""}, Position: int64(pos), UpdatedAt: time.Now().Unix(), ID: ticketID,
	}); err != nil {
		return classifyWriteErr(err)
	}
	return nil
}

func (r *TicketsRepo) AddLabel(ctx context.Context, id, label string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).AddTicketLabel(ctx, sqlcgen.AddTicketLabelParams{TicketID: id, Label: label}); err != nil {
			return fmt.Errorf("add label %q to ticket %s: %w", label, id, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *TicketsRepo) RemoveLabel(ctx context.Context, id, label string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).RemoveTicketLabel(ctx, sqlcgen.RemoveTicketLabelParams{TicketID: id, Label: label}); err != nil {
			return fmt.Errorf("remove label %q from ticket %s: %w", label, id, err)
		}
		return nil
	})
}

func (r *TicketsRepo) ListLabels(ctx context.Context, id string) ([]string, error) {
	out, err := r.q.ListTicketLabels(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list labels for ticket %s: %w", id, err)
	}
	return out, nil
}

func (r *TicketsRepo) ListAllLabels(ctx context.Context) ([]string, error) {
	out, err := r.q.ListAllTicketLabels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all labels: %w", err)
	}
	return out, nil
}

// SetLabelColor upserts a label's palette color within a project, so an unused label still works.
func (r *TicketsRepo) SetLabelColor(ctx context.Context, projectID, label string, color colors.Color) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		if err := r.q.WithTx(tx).SetLabelColor(ctx, sqlcgen.SetLabelColorParams{
			ProjectID: projectID, Label: label, Color: string(color), CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return fmt.Errorf("set color for label %q: %w", label, classifyWriteErr(err))
		}
		return nil
	})
}

// LabelColors returns a project's colors for a set of labels in one query, so the board avoids a round trip per label.
func (r *TicketsRepo) LabelColors(ctx context.Context, projectID string, labels []string) (map[string]colors.Color, error) {
	out := make(map[string]colors.Color, len(labels))
	if len(labels) == 0 {
		return out, nil
	}
	rows, err := r.q.ListLabelColors(ctx, sqlcgen.ListLabelColorsParams{ProjectID: projectID, Labels: labels})
	if err != nil {
		return nil, fmt.Errorf("list label colors: %w", err)
	}
	for _, row := range rows {
		out[row.Label] = colors.Color(row.Color)
	}
	return out, nil
}

// attachLabels loads labels for every ticket in one query, so responses carry the filter bar's label dimension.
func (r *TicketsRepo) attachLabels(ctx context.Context, ts []*tickets.Ticket) error {
	if len(ts) == 0 {
		return nil
	}
	byID := make(map[string]*tickets.Ticket, len(ts))
	ids := make([]string, 0, len(ts))
	for _, t := range ts {
		byID[t.ID] = t
		ids = append(ids, t.ID)
	}
	rows, err := r.q.ListTicketLabelsForTickets(ctx, ids)
	if err != nil {
		return fmt.Errorf("load labels: %w", err)
	}
	for _, row := range rows {
		if t, ok := byID[row.TicketID]; ok {
			t.Labels = append(t.Labels, row.Label)
		}
	}
	return nil
}

func insertTicketLabels(ctx context.Context, tx *sql.Tx, ticketID string, labels []string) error {
	q := sqlcgen.New(tx)
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if err := q.AddTicketLabel(ctx, sqlcgen.AddTicketLabelParams{TicketID: ticketID, Label: label}); err != nil {
			return fmt.Errorf("insert label %q for ticket %s: %w", label, ticketID, err)
		}
	}
	return nil
}

func defaultType(typeID string) string {
	if strings.TrimSpace(typeID) == "" {
		return "ticket-type-task"
	}
	return typeID
}

func toTicket(row sqlcgen.Ticket) *tickets.Ticket {
	t := &tickets.Ticket{
		ID:         row.ID,
		Title:      row.Title,
		Body:       row.Body,
		Status:     tickets.Status(row.Status),
		Position:   int(row.Position),
		Number:     int(row.Number),
		DocID:      row.DocID.String,
		ProjectID:  row.ProjectID.String,
		CategoryID: row.CategoryID.String,
		TypeID:     row.TypeID.String,
		Developer:  row.Developer,
		Tester:     row.Tester,
		Reporter: tickets.Reporter{
			Kind:           row.ReporterKind,
			Login:          row.ReporterLogin,
			AutomationID:   row.ReporterAutomationID,
			AutomationName: row.ReporterAutomationName,
		},
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if row.FinishedAt.Valid {
		t.FinishedAt = atPtr(time.Unix(row.FinishedAt.Int64, 0).UTC())
	}
	return t
}

func toTickets(rows []sqlcgen.Ticket) []*tickets.Ticket {
	var out []*tickets.Ticket
	for _, row := range rows {
		out = append(out, toTicket(row))
	}
	return out
}

func atPtr(t time.Time) *time.Time {
	return &t
}

func enqueueTicketsOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}

// enqueueWorkspaceOutbox writes the workspace domain's outbox events inside the same transaction as the mutation.
func enqueueWorkspaceOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}

package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.CrashTicketRepo = (*CrashTicketsRepo)(nil)

// CrashTicketsRepo backs deploy.CrashTicketRepo: the stack_crash_tickets table.
type CrashTicketsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *CrashTicketsRepo) Get(ctx context.Context, stackID, subject string) (string, error) {
	id, err := r.q.GetStackCrashTicket(ctx, sqlcgen.GetStackCrashTicketParams{StackID: stackID, Subject: subject})
	if err != nil {
		return "", fmt.Errorf("get crash ticket of %s on stack %s: %w", subject, stackID, notFoundIfNoRows(err))
	}
	return id, nil
}

func (r *CrashTicketsRepo) Set(ctx context.Context, stackID, subject, ticketID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetStackCrashTicket(ctx, sqlcgen.SetStackCrashTicketParams{StackID: stackID, Subject: subject, TicketID: ticketID})
		if err != nil {
			return fmt.Errorf("link crash ticket %s to %s on stack %s: %w", ticketID, subject, stackID, classifyWriteErr(err))
		}
		return nil
	})
}

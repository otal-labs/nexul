package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ access.Computers = (*AccessRepo)(nil)

// ComputerAccess reads computerID's owner and the live rule naming userID on it in one statement, past every memo.
func (r *AccessRepo) ComputerAccess(ctx context.Context, computerID, userID string) (string, permissions.Set, error) {
	row, err := r.q.GetComputerAccess(ctx, sqlcgen.GetComputerAccessParams{UserID: userID, ComputerID: computerID})
	if err != nil {
		return "", nil, fmt.Errorf("computer %s for %s: %w", computerID, userID, notFoundIfNoRows(err))
	}
	if row.RuleAllow == "" {
		return row.OwnerID, nil, nil
	}
	allow, err := parseSet(row.RuleAllow)
	if err != nil {
		return "", nil, fmt.Errorf("decode the rule on computer %s for %s: %w", computerID, userID, err)
	}
	return row.OwnerID, allow, nil
}

// SharesWithOwner reports whether personID is an active account sharing a workspace with ownerID.
func (r *AccessRepo) SharesWithOwner(ctx context.Context, ownerID, personID string) (bool, error) {
	shares, err := r.q.ComputerRuleTargetShares(ctx, sqlcgen.ComputerRuleTargetSharesParams{OwnerID: ownerID, PersonID: personID})
	if err != nil {
		return false, fmt.Errorf("does %s share a workspace with %s: %w", personID, ownerID, err)
	}
	return shares, nil
}

// EndComputerRules deletes every computer rule no longer live, with one event each, in one transaction.
func (r *AccessRepo) EndComputerRules(ctx context.Context, event func(computerID, ownerID, userID string) eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		ended, err := q.ListEndedComputerRules(ctx)
		if err != nil {
			return fmt.Errorf("list ended computer rules: %w", err)
		}
		events := make([]eventbus.OutboxEvent, 0, len(ended))
		for _, e := range ended {
			if _, err := q.DeleteOverwrite(ctx, sqlcgen.DeleteOverwriteParams{ResourceType: "computer", ResourceID: e.ComputerID, UserID: nullString(e.UserID)}); err != nil {
				return fmt.Errorf("end the rule on computer %s for %s: %w", e.ComputerID, e.UserID, err)
			}
			events = append(events, event(e.ComputerID, e.OwnerID, e.UserID))
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

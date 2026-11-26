package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ sso.Moves = (*SSORepo)(nil)

// moveName is what a person is called on the page: the name they set, else their sign-in account's.
func moveName(name string, display sql.NullString) string {
	if display.Valid && strings.TrimSpace(display.String) != "" {
		return display.String
	}
	return name
}

// OldSignIns lists the active accounts with a single sign-on of the other protocol than current and none of current's.
func (r *SSORepo) OldSignIns(ctx context.Context, current string) ([]sso.MoveAccount, error) {
	rows, err := r.q.ListSSOOldSignIns(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("list old sign-ins: %w", err)
	}
	out := make([]sso.MoveAccount, 0, len(rows))
	for _, row := range rows {
		out = append(out, sso.MoveAccount{ID: row.ID, Login: row.Login, Name: moveName(row.Name, row.DisplayName), Email: row.Email, CreatedAt: time.Unix(row.CreatedAt, 0).UTC()})
	}
	return out, nil
}

// OnlyOn lists the active accounts whose one sign-in is current's single sign-on.
func (r *SSORepo) OnlyOn(ctx context.Context, current string) ([]sso.MoveAccount, error) {
	rows, err := r.q.ListSSOOnlyAccounts(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("list accounts only %s signs in: %w", current, err)
	}
	out := make([]sso.MoveAccount, 0, len(rows))
	for _, row := range rows {
		out = append(out, sso.MoveAccount{ID: row.ID, Login: row.Login, Name: moveName(row.Name, row.DisplayName), Email: row.Email, CreatedAt: time.Unix(row.CreatedAt, 0).UTC()})
	}
	return out, nil
}

// ExpectedLinks lists every expected sign-in, by email.
func (r *SSORepo) ExpectedLinks(ctx context.Context) ([]sso.ExpectedLink, error) {
	rows, err := r.q.ListSSOExpectedLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expected sign-ins: %w", err)
	}
	out := make([]sso.ExpectedLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, sso.ExpectedLink{
			Account:   sso.MoveAccount{ID: row.UserID, Login: row.Login, Name: moveName(row.Name, row.DisplayName), Email: row.Email},
			CreatedBy: row.CreatedBy, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

// Expect saves the account's expected sign-in in one transaction with its event, refusing a removed account, one that
// signs in through current already, and an email another account expects.
func (r *SSORepo) Expect(ctx context.Context, link sso.ExpectedLink, current string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		user, err := q.GetUserByID(ctx, link.Account.ID)
		if err != nil {
			return fmt.Errorf("get account %s: %w", link.Account.ID, notFoundIfNoRows(err))
		}
		if user.AccountStatus == string(auth.AccountRemoved) {
			return fmt.Errorf("%w: that account was removed; restore it first", apperrs.ErrInvalid)
		}
		if err := hasNoSignInOf(ctx, q, user.ID, current); err != nil {
			return err
		}
		err = q.UpsertSSOExpectedLink(ctx, sqlcgen.UpsertSSOExpectedLinkParams{
			UserID: user.ID, Email: link.Account.Email, CreatedBy: link.CreatedBy, CreatedAt: link.CreatedAt.Unix(),
		})
		if errors.Is(classifyWriteErr(err), apperrs.ErrConflict) {
			return fmt.Errorf("%w: another account already expects a sign-in from %s", apperrs.ErrConflict, link.Account.Email)
		}
		if err != nil {
			return fmt.Errorf("save expected sign-in: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// Unexpect deletes the account's expected sign-in with its event.
func (r *SSORepo) Unexpect(ctx context.Context, userID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteSSOExpectedLink(ctx, userID)
		if err != nil {
			return fmt.Errorf("delete expected sign-in: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("%w: account %s expects no sign-in", apperrs.ErrNotFound, userID)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// ByExpected finds the account an expected sign-in names for email, ignoring case.
func (r *SSORepo) ByExpected(ctx context.Context, email string) (sso.Account, error) {
	row, err := r.q.GetSSOExpectedLinkByEmail(ctx, email)
	if err != nil {
		return sso.Account{}, fmt.Errorf("get expected sign-in: %w", notFoundIfNoRows(err))
	}
	return sso.Account{UserID: row.ID, Status: row.AccountStatus}, nil
}

// LinkExpected attaches the subject to the account and deletes its expected sign-in, with identity.linked and evts, in
// one transaction.
func (r *SSORepo) LinkExpected(ctx context.Context, userID string, id sso.Identity, evts ...eventbus.OutboxEvent) error {
	identity := ssoIdentity(userID, id)
	linked := eventbus.OutboxEvent{ID: ids.New(), Topic: auth.TopicIdentityLinked, Payload: auth.IdentityChangedEvent{UserID: userID, Provider: identity.Provider, Login: identity.Login}}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := insertIdentity(ctx, q, identity, time.Now()); err != nil {
			return fmt.Errorf("link expected sign-in: %w", err)
		}
		if _, err := q.DeleteSSOExpectedLink(ctx, userID); err != nil {
			return fmt.Errorf("use expected sign-in: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, append([]eventbus.OutboxEvent{linked}, evts...))
	})
}

// dropExpected deletes the expected sign-in of an account a sign-in was just moved onto, which no longer waits for one,
// and returns its event.
func dropExpected(ctx context.Context, q *sqlcgen.Queries, userID, actorID string) ([]eventbus.OutboxEvent, error) {
	email, err := q.GetSSOExpectedLinkEmail(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get expected sign-in of %s: %w", userID, err)
	}
	if _, err := q.DeleteSSOExpectedLink(ctx, userID); err != nil {
		return nil, fmt.Errorf("drop expected sign-in of %s: %w", userID, err)
	}
	return []eventbus.OutboxEvent{{ID: ids.New(), Topic: sso.TopicExpectedChanged, Payload: sso.ExpectedChangedEvent{
		AccountID: userID, Email: email, Removed: true, ChangedBy: actorID, ChangedAt: time.Now().UTC(),
	}}}, nil
}

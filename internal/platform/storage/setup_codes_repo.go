package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auth.SetupCodeStore = (*SetupCodesRepo)(nil)

type SetupCodesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// ReplaceSetupCode drops every earlier code and stores hash, in one transaction.
func (r *SetupCodesRepo) ReplaceSetupCode(ctx context.Context, hash string, createdAt, expiresAt time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeleteSetupCodes(ctx); err != nil {
			return fmt.Errorf("delete setup codes: %w", err)
		}
		err := q.CreateSetupCode(ctx, sqlcgen.CreateSetupCodeParams{CodeHash: hash, CreatedAt: createdAt.Unix(), ExpiresAt: expiresAt.Unix()})
		if err != nil {
			return fmt.Errorf("insert setup code: %w", classifyWriteErr(err))
		}
		return nil
	})
}

// SetupCodeValid reports whether hash names a stored code that has not expired at now.
func (r *SetupCodesRepo) SetupCodeValid(ctx context.Context, hash string, now time.Time) (bool, error) {
	n, err := r.q.CountLiveSetupCodes(ctx, sqlcgen.CountLiveSetupCodesParams{CodeHash: hash, ExpiresAt: now.Unix()})
	if err != nil {
		return false, fmt.Errorf("check setup code: %w", err)
	}
	return n > 0, nil
}

// ClearSetupCodes deletes every stored code.
func (r *SetupCodesRepo) ClearSetupCodes(ctx context.Context) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteSetupCodes(ctx); err != nil {
			return fmt.Errorf("delete setup codes: %w", err)
		}
		return nil
	})
}

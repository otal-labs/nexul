package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// LicenseRepo stores the instance's one enterprise license key as the signed text it was pasted as.
type LicenseRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Key returns the stored key text, empty when the instance holds none.
func (r *LicenseRepo) Key(ctx context.Context) (string, error) {
	key, err := r.q.GetLicenseKey(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get license key: %w", err)
	}
	return key, nil
}

// SetKey replaces the stored key and records who pasted it, with its events in the same transaction.
func (r *LicenseRepo) SetKey(ctx context.Context, key, setBy string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetLicenseKey(ctx, sqlcgen.SetLicenseKeyParams{Key: key, SetBy: setBy, SetAt: at.Unix()})
		if err != nil {
			return fmt.Errorf("set license key: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// DeleteKey removes the stored key, with its events in the same transaction.
func (r *LicenseRepo) DeleteKey(ctx context.Context, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteLicenseKey(ctx); err != nil {
			return fmt.Errorf("delete license key: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

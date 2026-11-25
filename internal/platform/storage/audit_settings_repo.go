package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/ee/auditlog"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auditlog.Repo = (*AuditSettingsRepo)(nil)

// AuditSettingsRepo stores the audit log's one setting, its stream secret sealed.
type AuditSettingsRepo struct {
	db     *sql.DB
	w      *Serializer
	q      *sqlcgen.Queries
	encKey []byte
}

// Get returns the zero Settings when the instance has none.
func (r *AuditSettingsRepo) Get(ctx context.Context) (auditlog.Settings, error) {
	row, err := r.q.GetAuditSettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return auditlog.Settings{}, nil
	}
	if err != nil {
		return auditlog.Settings{}, fmt.Errorf("get audit settings: %w", err)
	}
	st := auditlog.Settings{
		RetentionDays: int(row.RetentionDays), StreamURL: row.StreamUrl, StreamEnabled: row.StreamEnabled != 0,
		UpdatedBy: row.UpdatedBy, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if row.StreamSecret == "" {
		return st, nil
	}
	secret, err := crypto.Decrypt(r.encKey, row.StreamSecret)
	if err != nil {
		return auditlog.Settings{}, fmt.Errorf("decrypt audit stream secret: %w", err)
	}
	st.StreamSecret = string(secret)
	return st, nil
}

// Set replaces the settings, with its events in the same transaction.
func (r *AuditSettingsRepo) Set(ctx context.Context, s auditlog.Settings, evts ...eventbus.OutboxEvent) error {
	sealed := ""
	if s.StreamSecret != "" {
		var err error
		if sealed, err = crypto.Encrypt(r.encKey, []byte(s.StreamSecret)); err != nil {
			return fmt.Errorf("encrypt audit stream secret: %w", err)
		}
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetAuditSettings(ctx, sqlcgen.SetAuditSettingsParams{
			RetentionDays: int64(s.RetentionDays), StreamUrl: s.StreamURL, StreamSecret: sealed, StreamEnabled: boolToInt(s.StreamEnabled),
			UpdatedBy: s.UpdatedBy, UpdatedAt: s.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("set audit settings: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// Delete removes the settings, with its events in the same transaction.
func (r *AuditSettingsRepo) Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteAuditSettings(ctx); err != nil {
			return fmt.Errorf("delete audit settings: %w", err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

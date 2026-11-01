package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auth.PasskeyStore = (*PasskeysRepo)(nil)

type PasskeysRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *PasskeysRepo) InsertPasskey(ctx context.Context, p *auth.Passkey, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).InsertPasskey(ctx, sqlcgen.InsertPasskeyParams{
			ID: p.ID, UserID: p.UserID, Name: p.Name, RpID: p.RPID, Credential: string(p.Credential), CreatedAt: p.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert passkey: %w", classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *PasskeysRepo) GetPasskey(ctx context.Context, id string) (*auth.Passkey, error) {
	row, err := r.q.GetPasskey(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get passkey: %w", notFoundIfNoRows(err))
	}
	p := toPasskey(row)
	return &p, nil
}

func (r *PasskeysRepo) ListPasskeys(ctx context.Context, userID string) ([]auth.Passkey, error) {
	rows, err := r.q.ListPasskeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	out := make([]auth.Passkey, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPasskey(row))
	}
	return out, nil
}

func (r *PasskeysRepo) UpdatePasskeyUse(ctx context.Context, id string, credential []byte, usedAt time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdatePasskeyUse(ctx, sqlcgen.UpdatePasskeyUseParams{
			Credential: string(credential), LastUsedAt: sql.NullInt64{Int64: usedAt.Unix(), Valid: true}, ID: id,
		})
		return affectedOne(n, err, "update passkey "+id)
	})
}

func (r *PasskeysRepo) RenamePasskey(ctx context.Context, userID, id, name string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RenamePasskey(ctx, sqlcgen.RenamePasskeyParams{Name: name, ID: id, UserID: userID})
		return affectedOne(n, err, "rename passkey "+id)
	})
}

func (r *PasskeysRepo) DeletePasskey(ctx context.Context, userID, id string, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeletePasskey(ctx, sqlcgen.DeletePasskeyParams{ID: id, UserID: userID})
		if err := affectedOne(n, err, "delete passkey "+id); err != nil {
			return err
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func affectedOne(n int64, err error, what string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if n == 0 {
		return fmt.Errorf("%s: %w", what, apperrs.ErrNotFound)
	}
	return nil
}

func toPasskey(row sqlcgen.UserPasskey) auth.Passkey {
	p := auth.Passkey{
		ID: row.ID, UserID: row.UserID, Name: row.Name, RPID: row.RpID, Credential: []byte(row.Credential),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
	if row.LastUsedAt.Valid {
		p.LastUsedAt = optionalTime(row.LastUsedAt.Int64)
	}
	return p
}

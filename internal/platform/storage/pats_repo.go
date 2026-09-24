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

var _ auth.PATStore = (*PATsRepo)(nil)

// PATsRepo persists only the SHA-256 hash of a token, never the raw value, so a leaked DB dump cannot be replayed.
type PATsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *PATsRepo) Create(ctx context.Context, p *auth.PersonalAccessToken, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreatePAT(ctx, sqlcgen.CreatePATParams{
			ID:         p.ID,
			UserID:     p.UserID,
			Name:       p.Name,
			TokenHash:  p.TokenHash,
			Prefix:     p.Prefix,
			CreatedAt:  p.CreatedAt.Unix(),
			ComputerID: p.ComputerID,
		})
		if err != nil {
			return fmt.Errorf("insert pat %s: %w", p.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *PATsRepo) GetActiveForComputer(ctx context.Context, userID, computerID string) (*auth.PersonalAccessToken, error) {
	row, err := r.q.GetActiveComputerPAT(ctx, sqlcgen.GetActiveComputerPATParams{UserID: userID, ComputerID: computerID})
	if err != nil {
		return nil, fmt.Errorf("get active pat for computer %s: %w", computerID, notFoundIfNoRows(err))
	}
	return toPAT(row), nil
}

func (r *PATsRepo) GetByHash(ctx context.Context, hash string) (*auth.PersonalAccessToken, error) {
	row, err := r.q.GetPATByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get pat: %w", notFoundIfNoRows(err))
	}
	return toPAT(row), nil
}

func (r *PATsRepo) ListByUser(ctx context.Context, userID string) ([]auth.PersonalAccessToken, error) {
	rows, err := r.q.ListPATsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list pats: %w", err)
	}
	out := make([]auth.PersonalAccessToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toPAT(row))
	}
	return out, nil
}

func (r *PATsRepo) Revoke(ctx context.Context, id, userID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RevokePAT(ctx, sqlcgen.RevokePATParams{
			RevokedAt: nullUnixNow(), ID: id, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("revoke pat %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("revoke pat %s: %w", id, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *PATsRepo) TouchLastUsed(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).TouchPATLastUsed(ctx, sqlcgen.TouchPATLastUsedParams{
			LastUsedAt: nullUnixNow(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("touch pat %s: %w", id, err)
		}
		return nil
	})
}

func toPAT(row sqlcgen.PersonalAccessToken) *auth.PersonalAccessToken {
	p := &auth.PersonalAccessToken{
		ID:         row.ID,
		UserID:     row.UserID,
		Name:       row.Name,
		TokenHash:  row.TokenHash,
		Prefix:     row.Prefix,
		CreatedAt:  time.Unix(row.CreatedAt, 0).UTC(),
		ComputerID: row.ComputerID,
	}
	if row.LastUsedAt.Valid {
		t := time.Unix(row.LastUsedAt.Int64, 0).UTC()
		p.LastUsedAt = &t
	}
	if row.RevokedAt.Valid {
		t := time.Unix(row.RevokedAt.Int64, 0).UTC()
		p.RevokedAt = &t
	}
	return p
}

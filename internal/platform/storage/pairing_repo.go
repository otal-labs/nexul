package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ pairing.Repo = (*PairingRepo)(nil)

// PairingRepo persists pairing state: computers and per-user
// defaults. Bearer tokens arrive already encrypted from the use-case layer.
type PairingRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *PairingRepo) SaveComputer(ctx context.Context, c pairing.Computer) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingComputer(ctx, sqlcgen.SavePairingComputerParams{
			ID: c.ID, UserID: c.UserID, Kind: string(c.Kind), Name: c.Name, ServerUrl: c.ServerURL, BearerToken: c.BearerToken,
			TokenExpiresAt: c.TokenExpiresAt.Unix(), HarnessVersion: c.HarnessVersion,
			CreatedAt: c.CreatedAt.Unix(), UpdatedAt: c.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save computer %s: %w", c.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *PairingRepo) GetComputer(ctx context.Context, userID, id string) (*pairing.Computer, error) {
	row, err := r.q.GetPairingComputer(ctx, sqlcgen.GetPairingComputerParams{ID: id, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", id, notFoundIfNoRows(err))
	}
	c := toPairingComputer(row)
	return &c, nil
}

func (r *PairingRepo) ListComputers(ctx context.Context, userID string) ([]pairing.Computer, error) {
	rows, err := r.q.ListPairingComputers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list computers: %w", err)
	}
	out := make([]pairing.Computer, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPairingComputer(row))
	}
	return out, nil
}

func (r *PairingRepo) DeleteComputer(ctx context.Context, userID, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeletePairingComputer(ctx, sqlcgen.DeletePairingComputerParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("delete computer %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete computer %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *PairingRepo) GetComputerByID(ctx context.Context, id string) (*pairing.Computer, error) {
	row, err := r.q.GetPairingComputerByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get computer %s: %w", id, notFoundIfNoRows(err))
	}
	c := toPairingComputer(row)
	return &c, nil
}

func (r *PairingRepo) GetProjectLink(ctx context.Context, projectID string) (pairing.ProjectLink, error) {
	row, err := r.q.GetPairingProjectLink(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pairing.ProjectLink{}, nil
		}
		return pairing.ProjectLink{}, fmt.Errorf("get project link %s: %w", projectID, err)
	}
	return pairing.ProjectLink{
		ProjectID: row.ProjectID, ComputerID: row.ComputerID, HarnessProjectID: row.HarnessProjectID,
		Provider: row.Provider, Model: row.Model, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func (r *PairingRepo) SaveProjectLink(ctx context.Context, l pairing.ProjectLink) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingProjectLink(ctx, sqlcgen.SavePairingProjectLinkParams{
			ProjectID: l.ProjectID, ComputerID: l.ComputerID, HarnessProjectID: l.HarnessProjectID,
			Provider: l.Provider, Model: l.Model, UpdatedAt: l.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save project link %s: %w", l.ProjectID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *PairingRepo) DeleteProjectLink(ctx context.Context, projectID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeletePairingProjectLink(ctx, projectID); err != nil {
			return fmt.Errorf("clear project link %s: %w", projectID, err)
		}
		return nil
	})
}

func (r *PairingRepo) GetDefaults(ctx context.Context, userID string) (pairing.Defaults, error) {
	row, err := r.q.GetPairingDefaults(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pairing.Defaults{}, nil
		}
		return pairing.Defaults{}, fmt.Errorf("get defaults: %w", err)
	}
	return pairing.Defaults{
		DefaultComputerID: row.DefaultComputerID.String,
		FallbackProjectID: row.FallbackProjectID,
		Provider:          row.Provider,
		Model:             row.Model,
	}, nil
}

func (r *PairingRepo) SaveDefaults(ctx context.Context, d pairing.Defaults) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePairingDefaults(ctx, sqlcgen.SavePairingDefaultsParams{
			UserID:            d.UserID,
			DefaultComputerID: sql.NullString{String: d.DefaultComputerID, Valid: d.DefaultComputerID != ""},
			FallbackProjectID: d.FallbackProjectID,
			Provider:          d.Provider,
			Model:             d.Model,
		})
		if err != nil {
			return fmt.Errorf("save defaults for %s: %w", d.UserID, classifyWriteErr(err))
		}
		return nil
	})
}

func toPairingComputer(row sqlcgen.PairingComputer) pairing.Computer {
	return pairing.Computer{
		ID: row.ID, UserID: row.UserID, Kind: harness.Kind(row.Kind), Name: row.Name, ServerURL: row.ServerUrl, BearerToken: row.BearerToken,
		TokenExpiresAt: time.Unix(row.TokenExpiresAt, 0).UTC(), HarnessVersion: row.HarnessVersion,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

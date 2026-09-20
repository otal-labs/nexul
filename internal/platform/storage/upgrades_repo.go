package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/runner"
)

var _ runner.UpgradeRepo = (*InstanceUpgradesRepo)(nil)

type InstanceUpgradesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *InstanceUpgradesRepo) Create(ctx context.Context, u *runner.Upgrade) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateInstanceUpgrade(ctx, sqlcgen.CreateInstanceUpgradeParams{
			ID:          u.ID,
			FromVersion: u.FromVersion,
			ToVersion:   u.ToVersion,
			Status:      u.Status,
			Error:       u.Error,
			RequestedBy: u.RequestedBy,
			RunnerID:    u.RunnerID,
			CreatedAt:   u.CreatedAt.Unix(),
			UpdatedAt:   u.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert instance upgrade %s: %w", u.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *InstanceUpgradesRepo) GetByID(ctx context.Context, id string) (*runner.Upgrade, error) {
	row, err := r.q.GetInstanceUpgrade(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get instance upgrade %s: %w", id, notFoundIfNoRows(err))
	}
	return toUpgrade(row), nil
}

func (r *InstanceUpgradesRepo) Latest(ctx context.Context) (*runner.Upgrade, error) {
	row, err := r.q.GetLatestInstanceUpgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("get latest instance upgrade: %w", notFoundIfNoRows(err))
	}
	return toUpgrade(row), nil
}

func (r *InstanceUpgradesRepo) ListUnresolved(ctx context.Context) ([]*runner.Upgrade, error) {
	rows, err := r.q.ListUnresolvedInstanceUpgrades(ctx)
	if err != nil {
		return nil, fmt.Errorf("list unresolved instance upgrades: %w", err)
	}
	out := make([]*runner.Upgrade, 0, len(rows))
	for _, row := range rows {
		out = append(out, toUpgrade(row))
	}
	return out, nil
}

func (r *InstanceUpgradesRepo) SetStatus(ctx context.Context, id, status, errMsg string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetInstanceUpgradeStatus(ctx, sqlcgen.SetInstanceUpgradeStatusParams{
			Status: status, Error: errMsg, UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set instance upgrade %s status: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set instance upgrade %s status: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toUpgrade(row sqlcgen.InstanceUpgrade) *runner.Upgrade {
	return &runner.Upgrade{
		ID:          row.ID,
		FromVersion: row.FromVersion,
		ToVersion:   row.ToVersion,
		Status:      row.Status,
		Error:       row.Error,
		RequestedBy: row.RequestedBy,
		RunnerID:    row.RunnerID,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

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

var _ runner.MachineRepo = (*MachinesRepo)(nil)

// MachinesRepo persists machines.
type MachinesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *MachinesRepo) Create(ctx context.Context, m *runner.Machine) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateMachine(ctx, sqlcgen.CreateMachineParams{
			ID: m.ID, Name: m.Name, StackRoot: m.StackRoot, ReportedHostname: m.ReportedHostname,
			FirstSeen: m.FirstSeen.Unix(), LastSeen: m.LastSeen.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert machine %s: %w", m.Name, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *MachinesRepo) Get(ctx context.Context, id string) (*runner.Machine, error) {
	row, err := r.q.GetMachine(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get machine %s: %w", id, notFoundIfNoRows(err))
	}
	return toMachine(row), nil
}

func (r *MachinesRepo) GetByName(ctx context.Context, name string) (*runner.Machine, error) {
	row, err := r.q.GetMachineByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get machine %s: %w", name, notFoundIfNoRows(err))
	}
	return toMachine(row), nil
}

func (r *MachinesRepo) List(ctx context.Context) ([]*runner.Machine, error) {
	rows, err := r.q.ListMachines(ctx)
	if err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}
	out := make([]*runner.Machine, 0, len(rows))
	for _, row := range rows {
		out = append(out, toMachine(row))
	}
	return out, nil
}

func (r *MachinesRepo) Rename(ctx context.Context, id, name string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RenameMachine(ctx, sqlcgen.RenameMachineParams{Name: name, ID: id})
		if err != nil {
			return fmt.Errorf("rename machine %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("rename machine %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *MachinesRepo) SetStackRoot(ctx context.Context, id, stackRoot string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetMachineStackRoot(ctx, sqlcgen.SetMachineStackRootParams{StackRoot: stackRoot, ID: id})
		if err != nil {
			return fmt.Errorf("set machine %s stack root: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set machine %s stack root: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *MachinesRepo) Touch(ctx context.Context, id, reportedHostname string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).TouchMachine(ctx, sqlcgen.TouchMachineParams{
			LastSeen: at.Unix(), ReportedHostname: reportedHostname, ID: id,
		})
		if err != nil {
			return fmt.Errorf("touch machine %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("touch machine %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toMachine(row sqlcgen.Machine) *runner.Machine {
	return &runner.Machine{
		ID:               row.ID,
		Name:             row.Name,
		StackRoot:        row.StackRoot,
		ReportedHostname: row.ReportedHostname,
		FirstSeen:        time.Unix(row.FirstSeen, 0).UTC(),
		LastSeen:         time.Unix(row.LastSeen, 0).UTC(),
	}
}

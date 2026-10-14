package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
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

func (r *MachinesRepo) ListIn(ctx context.Context, ids []string) ([]*runner.Machine, error) {
	rows, err := r.q.ListMachinesIn(ctx, idsJSON(ids))
	if err != nil {
		return nil, fmt.Errorf("list machines in a set: %w", err)
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

func (r *MachinesRepo) Dependents(ctx context.Context, name string) (runner.MachineDependents, error) {
	stacks, err := r.q.ListMachineStackNames(ctx, name)
	if err != nil {
		return runner.MachineDependents{}, fmt.Errorf("list stacks on machine %s: %w", name, err)
	}
	gateways, err := r.q.ListMachineGatewayNetworks(ctx, name)
	if err != nil {
		return runner.MachineDependents{}, fmt.Errorf("list gateways on machine %s: %w", name, err)
	}
	return runner.MachineDependents{Stacks: stacks, Gateways: gateways}, nil
}

// Delete leaves the machine's rules to the machines_rules_ad trigger.
func (r *MachinesRepo) Delete(ctx context.Context, m *runner.Machine, at time.Time, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.DeleteMachine(ctx, m.ID)
		if err != nil {
			return fmt.Errorf("delete machine %s: %w", m.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("delete machine %s: %w", m.ID, apperrs.ErrNotFound)
		}
		revoked := sqlcgen.RevokeMachineRunnerCredentialsParams{RevokedAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, MachineID: m.ID}
		if err := q.RevokeMachineRunnerCredentials(ctx, revoked); err != nil {
			return fmt.Errorf("revoke credentials of runners on machine %s: %w", m.ID, err)
		}
		if err := q.DeleteMachineRunners(ctx, m.ID); err != nil {
			return fmt.Errorf("delete runners on machine %s: %w", m.ID, err)
		}
		if err := q.DeleteMachineEnrollments(ctx, m.Name); err != nil {
			return fmt.Errorf("delete enrollment codes for machine %s: %w", m.Name, err)
		}
		return insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload)
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

package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/runner"
)

var _ runner.Repo = (*RunnersRepo)(nil)

type RunnersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *RunnersRepo) Create(ctx context.Context, rn *runner.Runner) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateRunner(ctx, sqlcgen.CreateRunnerParams{
			ID:        rn.ID,
			Name:      rn.Name,
			Version:   rn.Version,
			LastSeen:  rn.LastSeen.Unix(),
			Connected: int64(boolInt(rn.Connected)),
			CreatedAt: rn.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert runner %s: %w", rn.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *RunnersRepo) GetByID(ctx context.Context, id string) (*runner.Runner, error) {
	row, err := r.q.GetRunner(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get runner %s: %w", id, notFoundIfNoRows(err))
	}
	return toRunner(row), nil
}

func (r *RunnersRepo) List(ctx context.Context) ([]*runner.Runner, error) {
	rows, err := r.q.ListRunners(ctx)
	if err != nil {
		return nil, fmt.Errorf("list runners: %w", err)
	}
	var out []*runner.Runner
	for _, row := range rows {
		out = append(out, toRunner(row))
	}
	return out, nil
}

func (r *RunnersRepo) Heartbeat(ctx context.Context, id string, at time.Time) error {
	return r.update(ctx, id, at.Unix(), true)
}

func (r *RunnersRepo) SetConnected(ctx context.Context, id string, connected bool) error {
	return r.update(ctx, id, time.Now().Unix(), connected)
}

func (r *RunnersRepo) SetVersion(ctx context.Context, id string, version string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetRunnerVersion(ctx, sqlcgen.SetRunnerVersionParams{Version: version, ID: id})
		if err != nil {
			return fmt.Errorf("set runner %s version: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set runner %s version: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *RunnersRepo) SetMachine(ctx context.Context, id string, machineID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetRunnerMachine(ctx, sqlcgen.SetRunnerMachineParams{MachineID: machineID, ID: id})
		if err != nil {
			return fmt.Errorf("set runner %s machine: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set runner %s machine: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *RunnersRepo) update(ctx context.Context, id string, lastSeen int64, connected bool) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateRunnerHeartbeat(ctx, sqlcgen.UpdateRunnerHeartbeatParams{
			LastSeen: lastSeen, Connected: int64(boolInt(connected)), ID: id,
		})
		if err != nil {
			return fmt.Errorf("update runner %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("update runner %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *RunnersRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteRunner(ctx, id)
		if err != nil {
			return fmt.Errorf("delete runner %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete runner %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// Secret generates and persists a value on first use; the UPDATE only applies while empty, so no clobbering a winner.
func (r *RunnersRepo) Secret(ctx context.Context) (string, error) {
	secret, err := r.readSecret(ctx)
	if err != nil {
		return "", fmt.Errorf("read runner secret: %w", err)
	}
	if secret != "" {
		return secret, nil
	}
	generated, err := generateRunnerSecret()
	if err != nil {
		return "", fmt.Errorf("generate runner secret: %w", err)
	}
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).SeedRunnerSecret(ctx, generated); err != nil {
			return fmt.Errorf("seed runner secret: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	secret, err = r.readSecret(ctx)
	if err != nil {
		return "", fmt.Errorf("read runner secret: %w", err)
	}
	return secret, nil
}

// SetSecret overwrites the runner-connection secret, e.g. seeding it from
// NEXUL_RUNNER_SECRET at boot.
func (r *RunnersRepo) SetSecret(ctx context.Context, secret string) error {
	if secret == "" {
		return fmt.Errorf("set runner secret: %w", apperrs.ErrInvalid)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).SetRunnerSecret(ctx, secret); err != nil {
			return fmt.Errorf("set runner secret: %w", err)
		}
		return nil
	})
}

func (r *RunnersRepo) readSecret(ctx context.Context) (string, error) {
	return r.q.GetRunnerSecret(ctx)
}

func generateRunnerSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func toRunner(row sqlcgen.Runner) *runner.Runner {
	return &runner.Runner{
		ID:        row.ID,
		Name:      row.Name,
		Version:   row.Version,
		LastSeen:  time.Unix(row.LastSeen, 0).UTC(),
		Connected: row.Connected == 1,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		MachineID: row.MachineID,
	}
}

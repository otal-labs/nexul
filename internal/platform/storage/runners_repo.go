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

var _ runner.Repo = (*RunnersRepo)(nil)

type RunnersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *RunnersRepo) Create(ctx context.Context, rn *runner.Runner) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateRunner(ctx, sqlcgen.CreateRunnerParams{
			ID:          rn.ID,
			Name:        rn.Name,
			Version:     rn.Version,
			LastSeen:    rn.LastSeen.Unix(),
			Connected:   int64(boolInt(rn.Connected)),
			CreatedAt:   rn.CreatedAt.Unix(),
			MachineID:   rn.MachineID,
			OwnerUserID: rn.OwnerUserID,
			ComputerID:  rn.ComputerID,
		})
		if err != nil {
			return fmt.Errorf("insert runner %s: %w", rn.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *RunnersRepo) GetByName(ctx context.Context, name string) (*runner.Runner, error) {
	row, err := r.q.GetRunnerByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get runner named %s: %w", name, notFoundIfNoRows(err))
	}
	return toRunner(row), nil
}

func (r *RunnersRepo) GetByComputer(ctx context.Context, computerID string) (*runner.Runner, error) {
	row, err := r.q.GetRunnerByComputer(ctx, computerID)
	if err != nil {
		return nil, fmt.Errorf("get runner of computer %s: %w", computerID, notFoundIfNoRows(err))
	}
	return toRunner(row), nil
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

func (r *RunnersRepo) CreateEnrollment(ctx context.Context, e *runner.EnrollmentCode) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.PruneRunnerEnrollmentCodes(ctx, e.CreatedAt.Unix()); err != nil {
			return fmt.Errorf("prune runner enrollment codes: %w", err)
		}
		err := q.CreateRunnerEnrollmentCode(ctx, sqlcgen.CreateRunnerEnrollmentCodeParams{
			CodeHash: e.CodeHash, Name: e.Name, Machine: e.Machine, CreatedAt: e.CreatedAt.Unix(), ExpiresAt: e.ExpiresAt.Unix(),
			OwnerUserID: e.OwnerUserID, ComputerID: e.ComputerID,
		})
		if err != nil {
			return fmt.Errorf("insert runner enrollment code: %w", classifyWriteErr(err))
		}
		return nil
	})
}

func (r *RunnersRepo) GetEnrollment(ctx context.Context, codeHash string, now time.Time) (*runner.EnrollmentCode, error) {
	row, err := r.q.GetRunnerEnrollmentCode(ctx, sqlcgen.GetRunnerEnrollmentCodeParams{CodeHash: codeHash, ExpiresAt: now.Unix()})
	if err != nil {
		return nil, fmt.Errorf("get runner enrollment code: %w", notFoundIfNoRows(err))
	}
	return &runner.EnrollmentCode{
		CodeHash: row.CodeHash, Name: row.Name, Machine: row.Machine, OwnerUserID: row.OwnerUserID, ComputerID: row.ComputerID,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), ExpiresAt: time.Unix(row.ExpiresAt, 0).UTC(),
	}, nil
}

func (r *RunnersRepo) Enroll(ctx context.Context, codeHash string, rn *runner.Runner, credentialHash string, now time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.ConsumeRunnerEnrollmentCode(ctx, sqlcgen.ConsumeRunnerEnrollmentCodeParams{CodeHash: codeHash, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("consume runner enrollment code: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("consume runner enrollment code: %w", apperrs.ErrNotFound)
		}
		err = q.CreateRunner(ctx, sqlcgen.CreateRunnerParams{
			ID: rn.ID, Name: rn.Name, Version: rn.Version, LastSeen: rn.LastSeen.Unix(),
			Connected: int64(boolInt(rn.Connected)), CreatedAt: rn.CreatedAt.Unix(), MachineID: rn.MachineID,
			OwnerUserID: rn.OwnerUserID, ComputerID: rn.ComputerID,
		})
		if err != nil {
			return fmt.Errorf("insert runner %s: %w", rn.Name, classifyWriteErr(err))
		}
		err = q.CreateRunnerCredential(ctx, sqlcgen.CreateRunnerCredentialParams{
			CredentialHash: credentialHash, RunnerID: rn.ID, RunnerName: rn.Name, CreatedAt: now.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert runner %s credential: %w", rn.Name, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *RunnersRepo) GetCredential(ctx context.Context, hash string) (*runner.Credential, error) {
	row, err := r.q.GetRunnerCredential(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get runner credential: %w", notFoundIfNoRows(err))
	}
	return &runner.Credential{
		RunnerID: row.RunnerID, RunnerName: row.RunnerName,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), Revoked: row.RevokedAt.Valid,
	}, nil
}

func (r *RunnersRepo) Remove(ctx context.Context, id string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.RevokeRunnerCredentials(ctx, sqlcgen.RevokeRunnerCredentialsParams{
			RevokedAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, RunnerID: id,
		})
		if err != nil {
			return fmt.Errorf("revoke runner %s credentials: %w", id, err)
		}
		n, err := q.DeleteRunner(ctx, id)
		if err != nil {
			return fmt.Errorf("delete runner %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete runner %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func toRunner(row sqlcgen.Runner) *runner.Runner {
	return &runner.Runner{
		ID:          row.ID,
		Name:        row.Name,
		Version:     row.Version,
		LastSeen:    time.Unix(row.LastSeen, 0).UTC(),
		Connected:   row.Connected == 1,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		MachineID:   row.MachineID,
		OwnerUserID: row.OwnerUserID,
		ComputerID:  row.ComputerID,
	}
}

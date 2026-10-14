package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
			EnrolledBy:  rn.EnrolledBy,
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

func (r *RunnersRepo) ListByOwner(ctx context.Context, userID string) ([]*runner.Runner, error) {
	rows, err := r.q.ListRunnersByOwner(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list runners of %s: %w", userID, err)
	}
	var out []*runner.Runner
	for _, row := range rows {
		out = append(out, toRunner(row))
	}
	return out, nil
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

func (r *RunnersRepo) ListOn(ctx context.Context, machineIDs []string) ([]*runner.Runner, error) {
	rows, err := r.q.ListRunnersOn(ctx, idsJSON(machineIDs))
	if err != nil {
		return nil, fmt.Errorf("list runners on machines: %w", err)
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
			ID: e.ID, CodeHash: e.CodeHash, Name: e.Name, Machine: e.Machine, CreatedAt: e.CreatedAt.Unix(), ExpiresAt: e.ExpiresAt.Unix(),
			OwnerUserID: e.OwnerUserID, ComputerID: e.ComputerID, EnrolledBy: e.EnrolledBy,
		})
		if err != nil {
			return fmt.Errorf("insert runner enrollment code: %w", classifyWriteErr(err))
		}
		return nil
	})
}

func (r *RunnersRepo) DeleteComputerEnrollments(ctx context.Context, computerID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteRunnerEnrollmentCodesByComputer(ctx, computerID); err != nil {
			return fmt.Errorf("delete enrollment codes of computer %s: %w", computerID, err)
		}
		return nil
	})
}

func (r *RunnersRepo) DeleteOwnerEnrollments(ctx context.Context, userID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteRunnerEnrollmentCodesByOwner(ctx, userID); err != nil {
			return fmt.Errorf("delete enrollment codes of %s: %w", userID, err)
		}
		return nil
	})
}

func (r *RunnersRepo) GetEnrollment(ctx context.Context, codeHash string, now time.Time) (*runner.EnrollmentCode, error) {
	row, err := r.q.GetRunnerEnrollmentCode(ctx, sqlcgen.GetRunnerEnrollmentCodeParams{CodeHash: codeHash, ExpiresAt: now.Unix()})
	if err != nil {
		return nil, fmt.Errorf("get runner enrollment code: %w", notFoundIfNoRows(err))
	}
	return toEnrollmentCode(row), nil
}

func (r *RunnersRepo) GetEnrollmentByID(ctx context.Context, id string, now time.Time) (*runner.EnrollmentCode, error) {
	row, err := r.q.GetRunnerEnrollmentCodeByID(ctx, sqlcgen.GetRunnerEnrollmentCodeByIDParams{ID: id, ExpiresAt: now.Unix()})
	if err != nil {
		return nil, fmt.Errorf("get runner enrollment code %s: %w", id, notFoundIfNoRows(err))
	}
	return toEnrollmentCode(row), nil
}

func (r *RunnersRepo) ListEnrollments(ctx context.Context, userID string, now time.Time) ([]*runner.EnrollmentCode, error) {
	rows, err := r.q.ListRunnerEnrollmentCodes(ctx, sqlcgen.ListRunnerEnrollmentCodesParams{Now: now.Unix(), UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("list runner enrollment codes: %w", err)
	}
	out := make([]*runner.EnrollmentCode, len(rows))
	for i, row := range rows {
		out[i] = toEnrollmentCode(row)
	}
	return out, nil
}

func (r *RunnersRepo) RevokeEnrollment(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteRunnerEnrollmentCodeByID(ctx, id)
		if err != nil {
			return fmt.Errorf("revoke runner enrollment code %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("revoke runner enrollment code %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toEnrollmentCode(row sqlcgen.RunnerEnrollmentCode) *runner.EnrollmentCode {
	return &runner.EnrollmentCode{
		ID: row.ID, CodeHash: row.CodeHash, Name: row.Name, Machine: row.Machine, OwnerUserID: row.OwnerUserID, ComputerID: row.ComputerID,
		EnrolledBy: row.EnrolledBy,
		CreatedAt:  time.Unix(row.CreatedAt, 0).UTC(), ExpiresAt: time.Unix(row.ExpiresAt, 0).UTC(),
	}
}

func (r *RunnersRepo) Enroll(ctx context.Context, codeHash string, rn *runner.Runner, credentialHash string, now time.Time, created *runner.CreatedMachine) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.ConsumeRunnerEnrollmentCode(ctx, sqlcgen.ConsumeRunnerEnrollmentCodeParams{CodeHash: codeHash, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("consume runner enrollment code: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("consume runner enrollment code: %w", apperrs.ErrNotFound)
		}
		if created != nil {
			if err := createMachineWithOwner(ctx, q, created, now); err != nil {
				return err
			}
		}
		err = q.CreateRunner(ctx, sqlcgen.CreateRunnerParams{
			ID: rn.ID, Name: rn.Name, Version: rn.Version, LastSeen: rn.LastSeen.Unix(),
			Connected: int64(boolInt(rn.Connected)), CreatedAt: rn.CreatedAt.Unix(), MachineID: rn.MachineID,
			OwnerUserID: rn.OwnerUserID, ComputerID: rn.ComputerID, EnrolledBy: rn.EnrolledBy,
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

// createMachineWithOwner writes a machine and the rule giving its first person every machine action, inside the
// enrollment's transaction so a machine never exists without the rule that lets its adder see it.
func createMachineWithOwner(ctx context.Context, q *sqlcgen.Queries, created *runner.CreatedMachine, now time.Time) error {
	m := created.Machine
	err := q.CreateMachine(ctx, sqlcgen.CreateMachineParams{
		ID: m.ID, Name: m.Name, StackRoot: m.StackRoot, ReportedHostname: m.ReportedHostname,
		FirstSeen: m.FirstSeen.Unix(), LastSeen: m.LastSeen.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert machine %s: %w", m.Name, classifyWriteErr(err))
	}
	if created.Owner == "" {
		return nil
	}
	allow, err := json.Marshal(permissions.SetOf(permissions.MachineActions...))
	if err != nil {
		return fmt.Errorf("encode the rule of machine %s: %w", m.Name, err)
	}
	_, err = q.UpsertOverwrite(ctx, sqlcgen.UpsertOverwriteParams{
		ResourceType: "machine", ResourceID: m.ID, UserID: nullString(created.Owner), Allow: string(allow), Deny: "[]", Now: now.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert the rule of machine %s: %w", m.Name, classifyWriteErr(err))
	}
	return nil
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
		EnrolledBy:  row.EnrolledBy,
	}
}

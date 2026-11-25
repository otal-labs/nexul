package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func (r *PairingRepo) InsertShellJob(ctx context.Context, job pairing.ShellJob, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).InsertShellJob(ctx, sqlcgen.InsertShellJobParams{
			ID: job.ID, ComputerID: job.ComputerID, OwnerID: job.OwnerID, RequesterID: job.RequesterID, Adapter: job.Adapter,
			Shell: job.Shell, Command: job.Command, Cwd: job.Cwd, TimeoutSeconds: int64(job.TimeoutSeconds), Status: job.Status,
			CreatedAt: job.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert command %s: %w", job.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) StartShellJob(ctx context.Context, id string, at time.Time, evt eventbus.OutboxEvent) (bool, error) {
	started := false
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).StartShellJob(ctx, sqlcgen.StartShellJobParams{StartedAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, ID: id})
		if err != nil {
			return fmt.Errorf("start command %s: %w", id, err)
		}
		if n == 0 {
			return nil
		}
		started = true
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
	return started, err
}

func (r *PairingRepo) FinishShellJob(ctx context.Context, job pairing.ShellJob, evt eventbus.OutboxEvent) error {
	exit := sql.NullInt64{}
	if job.ExitCode != nil {
		exit = sql.NullInt64{Int64: int64(*job.ExitCode), Valid: true}
	}
	truncated := int64(0)
	if job.OutputTruncated {
		truncated = 1
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).FinishShellJob(ctx, sqlcgen.FinishShellJobParams{
			Status: job.Status, ExitCode: exit, Error: job.Error, Output: job.Output, OutputTruncated: truncated,
			EndedAt: nullUnixPtr(job.EndedAt), ID: job.ID,
		})
		if err != nil {
			return fmt.Errorf("finish command %s: %w", job.ID, err)
		}
		if n == 0 {
			return nil
		}
		return insertOutboxRows(ctx, tx, []eventbus.OutboxEvent{evt})
	})
}

func (r *PairingRepo) GetShellJob(ctx context.Context, id string) (*pairing.ShellJob, error) {
	row, err := r.q.GetShellJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get command %s: %w", id, notFoundIfNoRows(err))
	}
	job := toShellJob(row)
	return &job, nil
}

func (r *PairingRepo) ListShellJobs(ctx context.Context, computerID, requesterID string, w paging.Window) ([]pairing.ShellJob, int, error) {
	total, err := r.q.CountShellJobs(ctx, sqlcgen.CountShellJobsParams{ComputerID: computerID, RequesterID: requesterID})
	if err != nil {
		return nil, 0, fmt.Errorf("count commands on computer %s: %w", computerID, err)
	}
	rows, err := r.q.ListShellJobs(ctx, sqlcgen.ListShellJobsParams{ComputerID: computerID, RequesterID: requesterID, Lim: int64(w.Limit), Off: int64(w.Offset)})
	if err != nil {
		return nil, 0, fmt.Errorf("list commands on computer %s: %w", computerID, err)
	}
	jobs := make([]pairing.ShellJob, 0, len(rows))
	for _, row := range rows {
		job := toShellJob(row)
		job.Output = ""
		jobs = append(jobs, job)
	}
	return jobs, int(total), nil
}

func (r *PairingRepo) PurgeShellOutput(ctx context.Context, before time.Time, limit int) (int64, error) {
	var n int64
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var err error
		n, err = r.q.WithTx(tx).PurgeShellJobOutput(ctx, sqlcgen.PurgeShellJobOutputParams{Before: sql.NullInt64{Int64: before.Unix(), Valid: true}, Lim: int64(limit)})
		return err
	})
	return n, err
}

func (r *PairingRepo) FailUnfinishedShellJobs(ctx context.Context, at time.Time) (int64, error) {
	var n int64
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		var err error
		n, err = r.q.WithTx(tx).FailUnfinishedShellJobs(ctx, sql.NullInt64{Int64: at.Unix(), Valid: true})
		return err
	})
	return n, err
}

func (r *PairingRepo) SetComputerShell(ctx context.Context, userID, computerID string, enabled bool, at time.Time) error {
	on := int64(0)
	if enabled {
		on = 1
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetPairingComputerShell(ctx, sqlcgen.SetPairingComputerShellParams{ShellEnabled: on, UpdatedAt: at.Unix(), ID: computerID, UserID: userID})
		if err != nil {
			return fmt.Errorf("set commands of computer %s: %w", computerID, err)
		}
		if n == 0 {
			return fmt.Errorf("set commands of computer %s: %w", computerID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toShellJob(row sqlcgen.RunnerShellJob) pairing.ShellJob {
	job := pairing.ShellJob{
		ID: row.ID, ComputerID: row.ComputerID, OwnerID: row.OwnerID, RequesterID: row.RequesterID, Adapter: row.Adapter,
		Shell: row.Shell, Command: row.Command, Cwd: row.Cwd, TimeoutSeconds: int(row.TimeoutSeconds), Status: row.Status,
		Error: row.Error, Output: row.Output, OutputTruncated: row.OutputTruncated != 0, OutputPurged: row.OutputPurged != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), StartedAt: unixPtrFromNull(row.StartedAt), EndedAt: unixPtrFromNull(row.EndedAt),
	}
	if row.ExitCode.Valid {
		code := int(row.ExitCode.Int64)
		job.ExitCode = &code
	}
	return job
}

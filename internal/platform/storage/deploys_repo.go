package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ deploy.Repo = (*DeploysRepo)(nil)

type DeploysRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *DeploysRepo) Create(ctx context.Context, d *deploy.Deploy, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateDeploy(ctx, sqlcgen.CreateDeployParams{
			ID:   d.ID,
			Kind: string(d.Kind),
			// StackID is a real FK to stacks(id) (ON DELETE SET NULL): an empty value must be SQL NULL, not
			// the literal string "", or the constraint rejects every deploy created without one (nullString
			// always sets Valid:true and is for read-side lookup params, not this insert).
			StackID:     sql.NullString{String: d.StackID, Valid: d.StackID != ""},
			ServiceID:   d.ServiceID,
			Service:     d.Service,
			Target:      d.Target,
			Image:       d.Image,
			Status:      string(d.Status),
			Strategy:    string(d.Strategy),
			TriggeredBy: d.TriggeredBy,
			RuleID:      d.RuleID,
			RuleName:    d.RuleName,
			TicketID:    d.TicketID,
			PrNumber:    int64(d.PRNumber),
			CreatedAt:   d.CreatedAt.Unix(),
			UpdatedAt:   d.UpdatedAt.Unix(),
			Address:     d.Address,
		})
		if err != nil {
			return fmt.Errorf("insert deploy %s: %w", d.ID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

// CancelRequested enqueues a cancel_requested outbox row; the runner reports back via status_changed.
func (r *DeploysRepo) CancelRequested(ctx context.Context, evt eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload)
	})
}

func (r *DeploysRepo) GetByID(ctx context.Context, id string) (*deploy.Deploy, error) {
	row, err := r.q.GetDeploy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get deploy %s: %w", id, notFoundIfNoRows(err))
	}
	return toDeploy(row), nil
}

func (r *DeploysRepo) List(ctx context.Context) ([]*deploy.Deploy, error) {
	rows, err := r.q.ListDeploys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list deploys: %w", err)
	}
	return toDeploys(rows), nil
}

func (r *DeploysRepo) ListByService(ctx context.Context, service string) ([]*deploy.Deploy, error) {
	rows, err := r.q.ListDeploysByService(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("list deploys for service %s: %w", service, err)
	}
	return toDeploys(rows), nil
}

func (r *DeploysRepo) ListByStackID(ctx context.Context, stackID string) ([]*deploy.Deploy, error) {
	rows, err := r.q.ListDeploysByStackID(ctx, nullString(stackID))
	if err != nil {
		return nil, fmt.Errorf("list deploys for stack %s: %w", stackID, err)
	}
	return toDeploys(rows), nil
}

func (r *DeploysRepo) ListByStatus(ctx context.Context, status deploy.Status) ([]*deploy.Deploy, error) {
	rows, err := r.q.ListDeploysByStatus(ctx, string(status))
	if err != nil {
		return nil, fmt.Errorf("list deploys with status %s: %w", status, err)
	}
	return toDeploys(rows), nil
}

// HasActive reports whether the stack has a pending or running deploy.
func (r *DeploysRepo) HasActive(ctx context.Context, stackID string) (bool, error) {
	n, err := r.q.CountActiveDeploys(ctx, nullString(stackID))
	if err != nil {
		return false, fmt.Errorf("check active deploys for stack %s: %w", stackID, err)
	}
	return n > 0, nil
}

// LastHealthy returns the most recent healthy deploy of a stack, which
// carries the image a rollback redeploys.
func (r *DeploysRepo) LastHealthy(ctx context.Context, stackID string) (*deploy.Deploy, error) {
	row, err := r.q.LastHealthyDeploy(ctx, nullString(stackID))
	if err != nil {
		return nil, fmt.Errorf("last healthy deploy for stack %s: %w", stackID, notFoundIfNoRows(err))
	}
	return toDeploy(row), nil
}

func (r *DeploysRepo) UpdateStatus(ctx context.Context, id string, status deploy.Status, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateDeployStatus(ctx, sqlcgen.UpdateDeployStatusParams{
			Status: string(status), UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("update deploy %s status: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("update deploy %s status: %w", id, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

// SetAddress records the runner-reported docker-network address on a deploy.
func (r *DeploysRepo) SetAddress(ctx context.Context, id, address string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetDeployAddress(ctx, sqlcgen.SetDeployAddressParams{
			Address: address, UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set deploy %s address: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set deploy %s address: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// AppendLogLines inserts one row per line inside one transaction; the deploy FK maps a missing deploy onto
// ErrNotFound, since the only constraint the insert can violate is that one.
func (r *DeploysRepo) AppendLogLines(ctx context.Context, id string, lines []deploy.LogLine, evts ...eventbus.OutboxEvent) error {
	if len(lines) == 0 {
		return nil
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, l := range lines {
			err := q.InsertDeployLogLine(ctx, sqlcgen.InsertDeployLogLineParams{DeployID: id, Ts: l.TS, Phase: l.Phase, Line: l.Text})
			if err == nil {
				continue
			}
			if errors.Is(classifyWriteErr(err), apperrs.ErrConflict) {
				return fmt.Errorf("append deploy %s log: %w", id, apperrs.ErrNotFound)
			}
			return fmt.Errorf("append deploy %s log: %w", id, err)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func insertOutboxRows(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}

func (r *DeploysRepo) ListLogLines(ctx context.Context, id string) ([]deploy.LogLine, error) {
	rows, err := r.q.ListDeployLogLines(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list deploy %s log: %w", id, err)
	}
	out := make([]deploy.LogLine, 0, len(rows))
	for _, row := range rows {
		out = append(out, deploy.LogLine{Seq: row.Seq, TS: row.Ts, Phase: row.Phase, Text: row.Line})
	}
	return out, nil
}

func toDeploy(row sqlcgen.Deploy) *deploy.Deploy {
	d := &deploy.Deploy{
		ID:          row.ID,
		Kind:        deploy.Kind(row.Kind),
		StackID:     row.StackID.String,
		ServiceID:   row.ServiceID,
		Service:     row.Service,
		Target:      row.Target,
		Image:       row.Image,
		Status:      deploy.Status(row.Status),
		Strategy:    deploy.Strategy(row.Strategy),
		Address:     row.Address,
		TriggeredBy: row.TriggeredBy,
		RuleID:      row.RuleID,
		RuleName:    row.RuleName,
		TicketID:    row.TicketID,
		PRNumber:    int(row.PrNumber),
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0).UTC(),
	}
	if d.Kind == "" {
		d.Kind = deploy.KindDeploy
	}
	return d
}

func toDeploys(rows []sqlcgen.Deploy) []*deploy.Deploy {
	var out []*deploy.Deploy
	for _, row := range rows {
		out = append(out, toDeploy(row))
	}
	return out
}

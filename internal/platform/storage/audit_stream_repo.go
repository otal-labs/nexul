package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/otal-labs/nexul/ee/auditlog"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ auditlog.Queue = (*AuditStreamRepo)(nil)

// auditStream is whether an audit row written now is queued for the stream, which the composition root answers
// through the enterprise setting and the license (ADR 0190); unset, nothing is queued.
type auditStream struct {
	on atomic.Pointer[func(context.Context) bool]
}

func (a *auditStream) queues(ctx context.Context) bool {
	if a == nil {
		return false
	}
	on := a.on.Load()
	return on != nil && (*on)(ctx)
}

// SetAuditStreaming tells every audit row's write whether to queue it for the stream, asked before each write's
// transaction begins.
func (s *Store) SetAuditStreaming(on func(context.Context) bool) {
	s.auditStream.on.Store(&on)
}

// appendAudit writes one audit row and, when stream is set, its delivery in the same transaction, so a committed row
// is never lost to the stream.
func appendAudit(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.AppendAuditParams, stream bool) error {
	if err := q.AppendAudit(ctx, row); err != nil {
		return err
	}
	if !stream {
		return nil
	}
	err := q.InsertAuditStreamDelivery(ctx, sqlcgen.InsertAuditStreamDeliveryParams{AuditID: row.ID, NextAttemptAt: row.CreatedAt, CreatedAt: row.CreatedAt})
	if err != nil {
		return fmt.Errorf("queue audit row %s for the stream: %w", row.ID, err)
	}
	return nil
}

// AuditStreamRepo is the stream's queue of deliveries.
type AuditStreamRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Due returns up to limit pending deliveries due by now, oldest audit row first.
func (r *AuditStreamRepo) Due(ctx context.Context, limit int, now time.Time) ([]auditlog.Pending, error) {
	rows, err := r.q.AuditStreamDue(ctx, sqlcgen.AuditStreamDueParams{Now: now.Unix(), MaxRows: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list due audit deliveries: %w", err)
	}
	out := make([]auditlog.Pending, 0, len(rows))
	for _, row := range rows {
		out = append(out, auditlog.Pending{
			ID: row.AuditID, ActorType: row.ActorType, ActorID: row.ActorID, TokenID: row.TokenID.String, Action: row.Action,
			CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), Attempts: int(row.Attempts),
		})
	}
	return out, nil
}

func (r *AuditStreamRepo) write(ctx context.Context, what string, fn func(q *sqlcgen.Queries) error) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := fn(r.q.WithTx(tx)); err != nil {
			return fmt.Errorf("%s: %w", what, err)
		}
		return nil
	})
}

// MarkDelivered records the receiver's 2xx.
func (r *AuditStreamRepo) MarkDelivered(ctx context.Context, auditID string, at time.Time) error {
	return r.write(ctx, "mark audit delivery sent", func(q *sqlcgen.Queries) error {
		return q.MarkAuditStreamDelivered(ctx, sqlcgen.MarkAuditStreamDeliveredParams{DeliveredAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, AuditID: auditID})
	})
}

// MarkFailed keeps the delivery pending until next, with why it failed.
func (r *AuditStreamRepo) MarkFailed(ctx context.Context, auditID string, attempts int, next time.Time, reason string) error {
	return r.write(ctx, "mark audit delivery failed", func(q *sqlcgen.Queries) error {
		return q.MarkAuditStreamFailed(ctx, sqlcgen.MarkAuditStreamFailedParams{Attempts: int64(attempts), NextAttemptAt: next.Unix(), LastError: reason, AuditID: auditID})
	})
}

// MarkDead stops retrying the delivery until an Owner retries it.
func (r *AuditStreamRepo) MarkDead(ctx context.Context, auditID string, attempts int, reason string) error {
	return r.write(ctx, "mark audit delivery dead", func(q *sqlcgen.Queries) error {
		return q.MarkAuditStreamDead(ctx, sqlcgen.MarkAuditStreamDeadParams{Attempts: int64(attempts), LastError: reason, AuditID: auditID})
	})
}

// RetryDead returns every dead delivery to pending, due at now.
func (r *AuditStreamRepo) RetryDead(ctx context.Context, now time.Time) (int, error) {
	var n int64
	err := r.write(ctx, "retry dead audit deliveries", func(q *sqlcgen.Queries) error {
		var err error
		n, err = q.RetryAuditStreamDead(ctx, now.Unix())
		return err
	})
	return int(n), err
}

// Stats counts the pending and dead deliveries and reads when the last one went out and the newest failure.
func (r *AuditStreamRepo) Stats(ctx context.Context) (auditlog.Stats, error) {
	var st auditlog.Stats
	counts, err := r.q.CountAuditStream(ctx)
	if err != nil {
		return st, fmt.Errorf("count audit deliveries: %w", err)
	}
	for _, c := range counts {
		if c.Status == "pending" {
			st.Pending = int(c.N)
		}
		if c.Status == "dead" {
			st.Dead = int(c.N)
		}
	}
	last, err := r.q.LastAuditStreamDelivery(ctx)
	if err != nil {
		return st, fmt.Errorf("read last audit delivery: %w", err)
	}
	at, ok, err := optionalInt(last)
	if err != nil {
		return st, fmt.Errorf("read last audit delivery: %w", err)
	}
	if ok {
		t := time.Unix(at, 0).UTC()
		st.LastDeliveredAt = &t
	}
	reason, err := r.q.LastAuditStreamError(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return st, fmt.Errorf("read last audit delivery error: %w", err)
	}
	st.LastError = reason
	return st, nil
}

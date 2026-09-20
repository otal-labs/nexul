package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/collab"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ collab.Store = (*CollabRepo)(nil)

// CollabRepo persists the opaque Y.js update log and the per-doc session watermark.
type CollabRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// AppendUpdate stores one payload and returns its seq. The snapshot and
// increment kinds share one per-doc counter, so replay ordering is total.
func (r *CollabRepo) AppendUpdate(ctx context.Context, docID, actorID string, kind collab.UpdateKind, payload string) (int64, error) {
	now := time.Now().Unix()
	var seq int64
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		var err error
		seq, err = q.AppendCollabUpdate(ctx, sqlcgen.AppendCollabUpdateParams{
			DocID: docID, Kind: string(kind), ActorID: actorID, Payload: payload, CreatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("append %s update for doc %s: %w", kind, docID, classifyWriteErr(err))
		}
		err = q.TouchCollabSession(ctx, sqlcgen.TouchCollabSessionParams{DocID: docID, LastSeq: seq, LastCommitAt: now})
		if err != nil {
			return fmt.Errorf("touch collab session %s: %w", docID, err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return seq, nil
}

// LoadReplay returns the newest snapshot plus every increment after it, and
// the current seq (the joining client's baseline for its first commit).
func (r *CollabRepo) LoadReplay(ctx context.Context, docID string) (collab.Replay, error) {
	var replay collab.Replay
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		// Newest snapshot first: the replay only needs the latest one.
		snap, err := q.GetLatestCollabSnapshot(ctx, docID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("load snapshot for doc %s: %w", docID, err)
		}
		var snapSeq int64
		if err == nil {
			replay.Snapshot = toStoredUpdate(snap)
			snapSeq = snap.Seq
		}
		rows, err := q.ListCollabIncrementsAfter(ctx, sqlcgen.ListCollabIncrementsAfterParams{DocID: docID, Seq: snapSeq})
		if err != nil {
			return fmt.Errorf("load increments for doc %s: %w", docID, err)
		}
		for _, row := range rows {
			replay.Increments = append(replay.Increments, *toStoredUpdate(row))
		}
		seq, err := q.GetCollabSessionSeq(ctx, docID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("load session seq for doc %s: %w", docID, err)
		}
		replay.Seq = seq
		return nil
	})
	if err != nil {
		return collab.Replay{}, err
	}
	return replay, nil
}

func toStoredUpdate(row sqlcgen.CollabUpdate) *collab.StoredUpdate {
	return &collab.StoredUpdate{
		Seq:       row.Seq,
		Kind:      collab.UpdateKind(row.Kind),
		ActorID:   row.ActorID,
		Payload:   row.Payload,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

// TrimUpdates deletes increments at or below baseSeq, keeping every snapshot; callers trim only to what it covers.
func (r *CollabRepo) TrimUpdates(ctx context.Context, docID string, baseSeq int64) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).TrimCollabUpdates(ctx, sqlcgen.TrimCollabUpdatesParams{DocID: docID, Seq: baseSeq})
		if err != nil {
			return fmt.Errorf("trim updates for doc %s: %w", docID, err)
		}
		return nil
	})
}

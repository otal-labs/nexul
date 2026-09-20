package collab

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// UpdateKind: snapshot rows are full-state commits; update rows are incremental deltas.
type UpdateKind string

const (
	KindUpdate   UpdateKind = "update"
	KindSnapshot UpdateKind = "snapshot"
)

// StoredUpdate's JSON tags matter: the browser's protocol.ts reads these keys verbatim off the wire.
type StoredUpdate struct {
	Seq       int64      `json:"seq"`
	Kind      UpdateKind `json:"kind"`
	ActorID   string     `json:"actor_id"`
	Payload   string     `json:"payload"`
	CreatedAt time.Time  `json:"created_at"`
}

// Replay is safe with overlap since Y.js updates are idempotent under apply.
type Replay struct {
	Snapshot   *StoredUpdate
	Increments []StoredUpdate
	Seq        int64
}

// Store is the consumer-side persistence contract for collab sessions; collab never imports storage (ADR 0017).
type Store interface {
	// AppendUpdate persists one payload and returns its seq; snapshot seqs and update seqs share one counter per doc.
	AppendUpdate(ctx context.Context, docID, actorID string, kind UpdateKind, payload string) (int64, error)
	// LoadReplay returns the stored state for a doc (newest snapshot + newer increments) plus the current seq.
	LoadReplay(ctx context.Context, docID string) (Replay, error)
	// TrimUpdates only trims up to the seq the snapshot's author applied, so no missed payload is dropped.
	TrimUpdates(ctx context.Context, docID string, baseSeq int64) error
}

// AccessChecker enforces the matching permission bit per participant; access is never frontend-only (ADR 0042).
type AccessChecker interface {
	Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error)
}

// DocWriter commits skip heavyweight version rows: no version per keystroke or save.
type DocWriter interface {
	CommitCollab(ctx context.Context, docID, title, body string) error
}

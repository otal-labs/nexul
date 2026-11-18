package issuesync

import (
	"context"
	"time"
)

// SyncStore keeps each repository's sync; a missing one is ErrNotFound.
type SyncStore interface {
	GetSync(ctx context.Context, owner, repo string) (*Sync, error)
	SyncsOfProject(ctx context.Context, projectID string) ([]*Sync, error)
	EnabledSyncs(ctx context.Context) ([]*Sync, error)
	SaveSync(ctx context.Context, s *Sync) error
	MarkSwept(ctx context.Context, owner, repo string, at time.Time) error
}

// LinkStore keeps the issue links; a missing one is ErrNotFound.
type LinkStore interface {
	GetLink(ctx context.Context, owner, repo string, number int) (*Link, error)
	SaveLink(ctx context.Context, l *Link) error
	DeleteLink(ctx context.Context, owner, repo string, number int) error
}

// CommentStore keeps the comment links; a missing one is ErrNotFound.
type CommentStore interface {
	GetCommentLink(ctx context.Context, commentID int64) (*CommentLink, error)
	SaveCommentLink(ctx context.Context, l CommentLink) error
	DeleteCommentLink(ctx context.Context, commentID int64) error
}

// Repo is the sync's storage, implemented in internal/platform/storage.
type Repo interface {
	SyncStore
	LinkStore
	CommentStore
}

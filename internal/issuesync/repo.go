package issuesync

import (
	"context"
	"time"
)

// SyncRepo stores each repository's sync settings.
type SyncRepo interface {
	// GetSync is ErrNotFound for a repository nobody turned sync on for.
	GetSync(ctx context.Context, owner, repo string) (Sync, error)
	ListSyncsByProject(ctx context.Context, projectID string) ([]Sync, error)
	ListEnabledSyncs(ctx context.Context) ([]Sync, error)
	PutSync(ctx context.Context, s Sync) error
	SetSwept(ctx context.Context, owner, repo string, at time.Time) error
}

// LinkRepo stores the issue and comment links.
type LinkRepo interface {
	// GetLink is ErrNotFound for an issue no ticket came from.
	GetLink(ctx context.Context, owner, repo string, number int) (Link, error)
	PutLink(ctx context.Context, l Link) error
	// DeleteLink removes the link and its ticket's comment links; the ticket stays.
	DeleteLink(ctx context.Context, l Link) error
	// GetCommentLink is ErrNotFound for a comment no message came from.
	GetCommentLink(ctx context.Context, commentID int64) (CommentLink, error)
	PutCommentLink(ctx context.Context, l CommentLink) error
	DeleteCommentLink(ctx context.Context, commentID int64) error
}

// Repo is everything the sync stores.
type Repo interface {
	SyncRepo
	LinkRepo
}

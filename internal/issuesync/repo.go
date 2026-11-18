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

// PushRepo stores what waits to be written to GitHub, and finds the links a write follows.
type PushRepo interface {
	// GetLinkByTicket is ErrNotFound for a ticket no issue is linked to.
	GetLinkByTicket(ctx context.Context, ticketID string) (Link, error)
	// GetCommentLinkByMessage is ErrNotFound for a message no comment is linked to.
	GetCommentLinkByMessage(ctx context.Context, messageID string) (CommentLink, error)
	// QueueTicketPush stores p, moving a waiting push's due to p's and keeping a create it asked for.
	QueueTicketPush(ctx context.Context, p TicketPush) error
	QueueCommentPush(ctx context.Context, p CommentPush) error
	DueTicketPushes(ctx context.Context, now time.Time, limit int) ([]TicketPush, error)
	DueCommentPushes(ctx context.Context, now time.Time, limit int) ([]CommentPush, error)
	// NextPushDue is the earliest due of every waiting push, false when none waits.
	NextPushDue(ctx context.Context) (time.Time, bool, error)
	// DeferTicketPush stores p's due unless the ticket was queued again since was, its tried unless one is stored, and
	// its attempts unless more are stored, so a stale copy never loses a first try.
	DeferTicketPush(ctx context.Context, p TicketPush, was time.Time) error
	DeferCommentPush(ctx context.Context, p CommentPush, was time.Time) error
	// DoneTicketPush drops p unless the ticket was queued again since p was read.
	DoneTicketPush(ctx context.Context, p TicketPush) error
	DoneCommentPush(ctx context.Context, p CommentPush) error
}

// Repo is everything the sync stores.
type Repo interface {
	SyncRepo
	LinkRepo
	PushRepo
}

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/issuesync"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ issuesync.Repo = (*IssueSyncRepo)(nil)

// IssueSyncRepo stores each repository's issue sync and the issue and comment links it keeps.
type IssueSyncRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *IssueSyncRepo) GetSync(ctx context.Context, owner, repo string) (*issuesync.Sync, error) {
	row, err := r.q.GetGitHubIssueSync(ctx, sqlcgen.GetGitHubIssueSyncParams{Owner: owner, Repo: repo})
	if err != nil {
		return nil, fmt.Errorf("get issue sync %s/%s: %w", owner, repo, notFoundIfNoRows(err))
	}
	return toIssueSync(row), nil
}

func (r *IssueSyncRepo) SyncsOfProject(ctx context.Context, projectID string) ([]*issuesync.Sync, error) {
	rows, err := r.q.ListGitHubIssueSyncsOfProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list issue syncs of project %s: %w", projectID, err)
	}
	return toIssueSyncs(rows), nil
}

func (r *IssueSyncRepo) EnabledSyncs(ctx context.Context) ([]*issuesync.Sync, error) {
	rows, err := r.q.ListEnabledGitHubIssueSyncs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled issue syncs: %w", err)
	}
	return toIssueSyncs(rows), nil
}

func (r *IssueSyncRepo) SaveSync(ctx context.Context, s *issuesync.Sync) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertGitHubIssueSync(ctx, sqlcgen.UpsertGitHubIssueSyncParams{
			ProjectID: s.ProjectID, Owner: s.Owner, Repo: s.Repo, Label: s.Label, Mode: string(s.Mode),
			Enabled: int64(boolInt(s.Enabled)), LastSweptAt: nullUnixPtr(s.LastSweptAt),
		})
		if err != nil {
			return fmt.Errorf("save issue sync %s/%s: %w", s.Owner, s.Repo, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) MarkSwept(ctx context.Context, owner, repo string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).MarkGitHubIssueSyncSwept(ctx, sqlcgen.MarkGitHubIssueSyncSweptParams{
			LastSweptAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, Owner: owner, Repo: repo,
		})
		if err != nil {
			return fmt.Errorf("mark issue sync %s/%s swept: %w", owner, repo, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) GetLink(ctx context.Context, owner, repo string, number int) (*issuesync.Link, error) {
	row, err := r.q.GetGitHubIssueLink(ctx, sqlcgen.GetGitHubIssueLinkParams{Owner: owner, Repo: repo, IssueNumber: int64(number)})
	if err != nil {
		return nil, fmt.Errorf("get issue link %s/%s#%d: %w", owner, repo, number, notFoundIfNoRows(err))
	}
	return &issuesync.Link{
		Owner: row.Owner, Repo: row.Repo, Number: int(row.IssueNumber), IssueID: row.IssueID, TicketID: row.TicketID,
		Paused: row.SyncPaused != 0, InboundHash: row.LastInboundHash, OutboundHash: row.LastOutboundHash,
		IssueUpdatedAt: time.Unix(row.IssueUpdatedAt, 0).UTC(), LinkedAt: time.Unix(row.LinkedAt, 0).UTC(),
	}, nil
}

func (r *IssueSyncRepo) SaveLink(ctx context.Context, l *issuesync.Link) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertGitHubIssueLink(ctx, sqlcgen.UpsertGitHubIssueLinkParams{
			Owner: l.Owner, Repo: l.Repo, IssueNumber: int64(l.Number), IssueID: l.IssueID, TicketID: l.TicketID,
			SyncPaused: int64(boolInt(l.Paused)), LastInboundHash: l.InboundHash, LastOutboundHash: l.OutboundHash,
			IssueUpdatedAt: unixOrZero(&l.IssueUpdatedAt), LinkedAt: l.LinkedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save issue link %s/%s#%d: %w", l.Owner, l.Repo, l.Number, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) DeleteLink(ctx context.Context, owner, repo string, number int) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeleteGitHubIssueLink(ctx, sqlcgen.DeleteGitHubIssueLinkParams{Owner: owner, Repo: repo, IssueNumber: int64(number)})
		if err != nil {
			return fmt.Errorf("delete issue link %s/%s#%d: %w", owner, repo, number, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) GetCommentLink(ctx context.Context, commentID int64) (*issuesync.CommentLink, error) {
	row, err := r.q.GetGitHubCommentLink(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("get comment link %d: %w", commentID, notFoundIfNoRows(err))
	}
	return &issuesync.CommentLink{CommentID: row.GithubCommentID, MessageID: row.MessageID, TicketID: row.TicketID}, nil
}

func (r *IssueSyncRepo) SaveCommentLink(ctx context.Context, l issuesync.CommentLink) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).InsertGitHubCommentLink(ctx, sqlcgen.InsertGitHubCommentLinkParams{GithubCommentID: l.CommentID, MessageID: l.MessageID, TicketID: l.TicketID})
		if err != nil {
			return fmt.Errorf("save comment link %d: %w", l.CommentID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) DeleteCommentLink(ctx context.Context, commentID int64) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteGitHubCommentLink(ctx, commentID); err != nil {
			return fmt.Errorf("delete comment link %d: %w", commentID, err)
		}
		return nil
	})
}

// GitHubUserID is the account that signed in with the GitHub login, matched ignoring case; found is false for none.
func (r *IssueSyncRepo) GitHubUserID(ctx context.Context, login string) (string, bool, error) {
	id, err := r.q.GitHubIdentityUserID(ctx, login)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("find the account of GitHub login %s: %w", login, err)
	}
	return id, true, nil
}

func toIssueSync(row sqlcgen.GithubIssueSync) *issuesync.Sync {
	return &issuesync.Sync{
		ProjectID: row.ProjectID, Owner: row.Owner, Repo: row.Repo, Label: row.Label, Mode: issuesync.Mode(row.Mode),
		Enabled: row.Enabled != 0, LastSweptAt: unixPtrFromNull(row.LastSweptAt),
	}
}

func toIssueSyncs(rows []sqlcgen.GithubIssueSync) []*issuesync.Sync {
	out := make([]*issuesync.Sync, 0, len(rows))
	for _, row := range rows {
		out = append(out, toIssueSync(row))
	}
	return out
}

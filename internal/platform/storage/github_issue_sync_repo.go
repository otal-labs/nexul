package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/issuesync"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ issuesync.Repo = (*IssueSyncRepo)(nil)

// IssueSyncRepo stores GitHub issue syncs and their issue and comment links.
type IssueSyncRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *IssueSyncRepo) GetSync(ctx context.Context, owner, repo string) (issuesync.Sync, error) {
	row, err := r.q.GetIssueSync(ctx, sqlcgen.GetIssueSyncParams{Owner: owner, Repo: repo})
	if err != nil {
		return issuesync.Sync{}, fmt.Errorf("get issue sync %s/%s: %w", owner, repo, notFoundIfNoRows(err))
	}
	return toIssueSync(row), nil
}

func (r *IssueSyncRepo) ListSyncsByProject(ctx context.Context, projectID string) ([]issuesync.Sync, error) {
	rows, err := r.q.ListIssueSyncsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list issue syncs of project %s: %w", projectID, err)
	}
	return toIssueSyncs(rows), nil
}

func (r *IssueSyncRepo) ListEnabledSyncs(ctx context.Context) ([]issuesync.Sync, error) {
	rows, err := r.q.ListEnabledIssueSyncs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled issue syncs: %w", err)
	}
	return toIssueSyncs(rows), nil
}

func (r *IssueSyncRepo) PutSync(ctx context.Context, s issuesync.Sync) error {
	var swept sql.NullInt64
	if s.LastSweptAt != nil {
		swept = sql.NullInt64{Int64: s.LastSweptAt.Unix(), Valid: true}
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).PutIssueSync(ctx, sqlcgen.PutIssueSyncParams{
			ProjectID: s.ProjectID, Owner: s.Owner, Repo: s.Repo, Label: s.Label, Mode: string(s.Mode),
			Enabled: int64(boolInt(s.Enabled)), LastSweptAt: swept,
		})
		if err != nil {
			return fmt.Errorf("put issue sync %s/%s: %w", s.Owner, s.Repo, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) SetSwept(ctx context.Context, owner, repo string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetIssueSyncSwept(ctx, sqlcgen.SetIssueSyncSweptParams{
			LastSweptAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, Owner: owner, Repo: repo,
		})
		if err != nil {
			return fmt.Errorf("mark issue sync %s/%s swept: %w", owner, repo, err)
		}
		if n == 0 {
			return fmt.Errorf("mark issue sync %s/%s swept: %w", owner, repo, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *IssueSyncRepo) GetLink(ctx context.Context, owner, repo string, number int) (issuesync.Link, error) {
	row, err := r.q.GetIssueLink(ctx, sqlcgen.GetIssueLinkParams{Owner: owner, Repo: repo, IssueNumber: int64(number)})
	if err != nil {
		return issuesync.Link{}, fmt.Errorf("get issue link %s/%s#%d: %w", owner, repo, number, notFoundIfNoRows(err))
	}
	return toIssueLink(row), nil
}

func (r *IssueSyncRepo) GetLinkByTicket(ctx context.Context, ticketID string) (issuesync.Link, error) {
	row, err := r.q.GetIssueLinkByTicket(ctx, ticketID)
	if err != nil {
		return issuesync.Link{}, fmt.Errorf("get issue link of ticket %s: %w", ticketID, notFoundIfNoRows(err))
	}
	return toIssueLink(row), nil
}

func toIssueLink(row sqlcgen.GithubIssueLink) issuesync.Link {
	return issuesync.Link{
		Owner: row.Owner, Repo: row.Repo, IssueNumber: int(row.IssueNumber), IssueID: row.IssueID, TicketID: row.TicketID,
		Paused: row.SyncPaused != 0, InboundHash: row.LastInboundHash, OutboundHash: row.LastOutboundHash, TicketHash: row.LastTicketHash,
		IssueUpdatedAt: time.Unix(row.IssueUpdatedAt, 0).UTC(), LinkedAt: time.Unix(row.LinkedAt, 0).UTC(),
	}
}

func (r *IssueSyncRepo) PutLink(ctx context.Context, l issuesync.Link) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).PutIssueLink(ctx, sqlcgen.PutIssueLinkParams{
			Owner: l.Owner, Repo: l.Repo, IssueNumber: int64(l.IssueNumber), IssueID: l.IssueID, TicketID: l.TicketID,
			SyncPaused: int64(boolInt(l.Paused)), LastInboundHash: l.InboundHash, LastOutboundHash: l.OutboundHash,
			LastTicketHash: l.TicketHash, IssueUpdatedAt: l.IssueUpdatedAt.Unix(), LinkedAt: l.LinkedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("put issue link %s/%s#%d: %w", l.Owner, l.Repo, l.IssueNumber, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) DeleteLink(ctx context.Context, l issuesync.Link) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.DeleteIssueCommentLinksByTicket(ctx, l.TicketID); err != nil {
			return fmt.Errorf("delete comment links of ticket %s: %w", l.TicketID, err)
		}
		if err := q.DeleteIssueLink(ctx, sqlcgen.DeleteIssueLinkParams{Owner: l.Owner, Repo: l.Repo, IssueNumber: int64(l.IssueNumber)}); err != nil {
			return fmt.Errorf("delete issue link %s/%s#%d: %w", l.Owner, l.Repo, l.IssueNumber, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) GetCommentLink(ctx context.Context, commentID int64) (issuesync.CommentLink, error) {
	row, err := r.q.GetIssueCommentLink(ctx, commentID)
	if err != nil {
		return issuesync.CommentLink{}, fmt.Errorf("get comment link %d: %w", commentID, notFoundIfNoRows(err))
	}
	return issuesync.CommentLink{GitHubCommentID: row.GithubCommentID, MessageID: row.MessageID, TicketID: row.TicketID}, nil
}

func (r *IssueSyncRepo) PutCommentLink(ctx context.Context, l issuesync.CommentLink) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).PutIssueCommentLink(ctx, sqlcgen.PutIssueCommentLinkParams{
			GithubCommentID: l.GitHubCommentID, MessageID: l.MessageID, TicketID: l.TicketID,
		})
		if err != nil {
			return fmt.Errorf("put comment link %d: %w", l.GitHubCommentID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) DeleteCommentLink(ctx context.Context, commentID int64) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteIssueCommentLink(ctx, commentID); err != nil {
			return fmt.Errorf("delete comment link %d: %w", commentID, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) GetCommentLinkByMessage(ctx context.Context, messageID string) (issuesync.CommentLink, error) {
	row, err := r.q.GetIssueCommentLinkByMessage(ctx, messageID)
	if err != nil {
		return issuesync.CommentLink{}, fmt.Errorf("get comment link of message %s: %w", messageID, notFoundIfNoRows(err))
	}
	return issuesync.CommentLink{GitHubCommentID: row.GithubCommentID, MessageID: row.MessageID, TicketID: row.TicketID}, nil
}

func (r *IssueSyncRepo) QueueTicketPush(ctx context.Context, p issuesync.TicketPush) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).QueueIssuePush(ctx, sqlcgen.QueueIssuePushParams{
			TicketID: p.TicketID, Owner: p.Owner, Repo: p.Repo, CreateIssue: int64(boolInt(p.Create)), DueAt: p.Due.UnixMilli(),
		})
		if err != nil {
			return fmt.Errorf("queue issue push of ticket %s: %w", p.TicketID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) QueueCommentPush(ctx context.Context, p issuesync.CommentPush) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).QueueCommentPush(ctx, sqlcgen.QueueCommentPushParams{MessageID: p.MessageID, TicketID: p.TicketID, DueAt: p.Due.UnixMilli()})
		if err != nil {
			return fmt.Errorf("queue comment push of message %s: %w", p.MessageID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *IssueSyncRepo) DueTicketPushes(ctx context.Context, now time.Time, limit int) ([]issuesync.TicketPush, error) {
	rows, err := r.q.DueIssuePushes(ctx, sqlcgen.DueIssuePushesParams{DueAt: now.UnixMilli(), Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list due issue pushes: %w", err)
	}
	out := make([]issuesync.TicketPush, 0, len(rows))
	for _, row := range rows {
		out = append(out, issuesync.TicketPush{
			TicketID: row.TicketID, Owner: row.Owner, Repo: row.Repo, Create: row.CreateIssue != 0,
			Due: time.UnixMilli(row.DueAt).UTC(), Tried: fromMilli(row.TriedAt), Attempts: int(row.Attempts),
		})
	}
	return out, nil
}

func (r *IssueSyncRepo) DueCommentPushes(ctx context.Context, now time.Time, limit int) ([]issuesync.CommentPush, error) {
	rows, err := r.q.DueCommentPushes(ctx, sqlcgen.DueCommentPushesParams{DueAt: now.UnixMilli(), Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list due comment pushes: %w", err)
	}
	out := make([]issuesync.CommentPush, 0, len(rows))
	for _, row := range rows {
		out = append(out, issuesync.CommentPush{
			MessageID: row.MessageID, TicketID: row.TicketID, Due: time.UnixMilli(row.DueAt).UTC(), Tried: fromMilli(row.TriedAt), Attempts: int(row.Attempts),
		})
	}
	return out, nil
}

func (r *IssueSyncRepo) NextPushDue(ctx context.Context) (time.Time, bool, error) {
	tickets, err := r.q.NextIssuePushDue(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("next issue push due: %w", err)
	}
	comments, err := r.q.NextCommentPushDue(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("next comment push due: %w", err)
	}
	t, tok, err := optionalInt(tickets)
	if err != nil {
		return time.Time{}, false, err
	}
	c, cok, err := optionalInt(comments)
	if err != nil {
		return time.Time{}, false, err
	}
	if !tok && !cok {
		return time.Time{}, false, nil
	}
	if !tok || (cok && c < t) {
		t = c
	}
	return time.UnixMilli(t).UTC(), true, nil
}

func (r *IssueSyncRepo) DeferTicketPush(ctx context.Context, p issuesync.TicketPush, was time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeferIssuePush(ctx, sqlcgen.DeferIssuePushParams{
			Was: was.UnixMilli(), Due: p.Due.UnixMilli(), TriedAt: toMilli(p.Tried), Attempts: int64(p.Attempts), TicketID: p.TicketID,
		})
		if err != nil {
			return fmt.Errorf("defer issue push of ticket %s: %w", p.TicketID, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) DeferCommentPush(ctx context.Context, p issuesync.CommentPush, was time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeferCommentPush(ctx, sqlcgen.DeferCommentPushParams{
			Was: was.UnixMilli(), Due: p.Due.UnixMilli(), TriedAt: toMilli(p.Tried), Attempts: int64(p.Attempts), MessageID: p.MessageID,
		})
		if err != nil {
			return fmt.Errorf("defer comment push of message %s: %w", p.MessageID, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) DoneTicketPush(ctx context.Context, p issuesync.TicketPush) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteIssuePush(ctx, sqlcgen.DeleteIssuePushParams{TicketID: p.TicketID, DueAt: p.Due.UnixMilli()}); err != nil {
			return fmt.Errorf("drop issue push of ticket %s: %w", p.TicketID, err)
		}
		return nil
	})
}

func (r *IssueSyncRepo) DoneCommentPush(ctx context.Context, p issuesync.CommentPush) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteCommentPush(ctx, sqlcgen.DeleteCommentPushParams{MessageID: p.MessageID, DueAt: p.Due.UnixMilli()}); err != nil {
			return fmt.Errorf("drop comment push of message %s: %w", p.MessageID, err)
		}
		return nil
	})
}

func toMilli(t time.Time) sql.NullInt64 {
	if t.IsZero() {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.UnixMilli(), Valid: true}
}

func fromMilli(v sql.NullInt64) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return time.UnixMilli(v.Int64).UTC()
}

func toIssueSyncs(rows []sqlcgen.GithubIssueSync) []issuesync.Sync {
	out := make([]issuesync.Sync, 0, len(rows))
	for _, row := range rows {
		out = append(out, toIssueSync(row))
	}
	return out
}

func toIssueSync(row sqlcgen.GithubIssueSync) issuesync.Sync {
	s := issuesync.Sync{
		ProjectID: row.ProjectID, Owner: row.Owner, Repo: row.Repo, Label: row.Label, Mode: issuesync.Mode(row.Mode),
		Enabled: row.Enabled != 0,
	}
	if row.LastSweptAt.Valid {
		swept := time.Unix(row.LastSweptAt.Int64, 0).UTC()
		s.LastSweptAt = &swept
	}
	return s
}

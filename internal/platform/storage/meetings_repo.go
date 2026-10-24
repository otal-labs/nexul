package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/calendar"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ calendar.Repo = (*MeetingsRepo)(nil)

// MeetingsRepo stores the calendar's meetings with their invitees, and the due-date reminders already sent.
type MeetingsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *MeetingsRepo) Get(ctx context.Context, id string) (*calendar.Meeting, error) {
	row, err := r.q.MeetingGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get meeting %s: %w", id, notFoundIfNoRows(err))
	}
	ms, err := r.withInvitees(ctx, []sqlcgen.Meeting{row})
	if err != nil {
		return nil, err
	}
	return ms[0], nil
}

func (r *MeetingsRepo) ListVisible(ctx context.Context, workspaceID, userID string, f calendar.RangeFilter) ([]*calendar.Meeting, error) {
	rows, err := r.q.MeetingListVisible(ctx, sqlcgen.MeetingListVisibleParams{
		WorkspaceID: workspaceID, UserID: userID, FromAt: f.From.Unix(), ToAt: f.To.Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("list meetings: %w", err)
	}
	return r.withInvitees(ctx, rows)
}

// PageVisible pages the meetings a person organises or is invited to.
// hand-written: sqlc cannot express a WHERE clause whose range ends are each optional.
func (r *MeetingsRepo) PageVisible(ctx context.Context, workspaceID, userID string, f calendar.RangeFilter, w paging.Window) ([]*calendar.Meeting, int, error) {
	q := pageQuery{from: "meetings", id: "id", order: "starts_at, id"}
	q.where("workspace_id = ?", workspaceID)
	q.where("(organiser_id = ? OR id IN (SELECT meeting_id FROM meeting_invitees WHERE user_id = ?))", userID, userID)
	if !f.To.IsZero() {
		q.where("starts_at < ?", f.To.Unix())
	}
	if !f.From.IsZero() {
		q.where("(ends_at > ? OR (repeats != 'none' AND (repeat_until IS NULL OR repeat_until + (ends_at - starts_at) > ?)))", f.From.Unix(), f.From.Unix())
	}
	ids, total, err := q.page(ctx, r.db, w)
	if err != nil {
		return nil, 0, fmt.Errorf("page meetings: %w", err)
	}
	if len(ids) == 0 {
		return []*calendar.Meeting{}, total, nil
	}
	rows, err := r.q.MeetingsByIDs(ctx, idsJSON(ids))
	if err != nil {
		return nil, 0, fmt.Errorf("read meetings: %w", err)
	}
	byID := make(map[string]sqlcgen.Meeting, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	ordered := make([]sqlcgen.Meeting, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			ordered = append(ordered, row)
		}
	}
	ms, err := r.withInvitees(ctx, ordered)
	if err != nil {
		return nil, 0, err
	}
	return ms, total, nil
}

func (r *MeetingsRepo) Replacements(ctx context.Context, seriesIDs []string) ([]*calendar.Meeting, error) {
	if len(seriesIDs) == 0 {
		return nil, nil
	}
	rows, err := r.q.MeetingReplacements(ctx, idsJSON(seriesIDs))
	if err != nil {
		return nil, fmt.Errorf("read changed occurrences: %w", err)
	}
	return r.withInvitees(ctx, rows)
}

func (r *MeetingsRepo) Save(ctx context.Context, meetings []*calendar.Meeting, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, m := range meetings {
			if err := q.MeetingUpsert(ctx, meetingParams(m)); err != nil {
				return fmt.Errorf("save meeting %s: %w", m.ID, classifyWriteErr(err))
			}
			if err := q.MeetingInviteesDeleteExcept(ctx, sqlcgen.MeetingInviteesDeleteExceptParams{MeetingID: m.ID, UserIds: idsJSON(m.InviteeIDs())}); err != nil {
				return fmt.Errorf("uninvite from meeting %s: %w", m.ID, err)
			}
			for _, inv := range m.Invitees {
				err := q.MeetingInviteeInsert(ctx, sqlcgen.MeetingInviteeInsertParams{
					MeetingID: m.ID, UserID: inv.UserID, Reply: string(inv.Reply), RepliedAt: nullUnixPtr(inv.RepliedAt),
				})
				if err != nil {
					return fmt.Errorf("invite %s to meeting %s: %w", inv.UserID, m.ID, classifyWriteErr(err))
				}
			}
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *MeetingsRepo) SaveReply(ctx context.Context, meetingID, userID string, reply calendar.Reply, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MeetingReplySave(ctx, sqlcgen.MeetingReplySaveParams{
			Reply: string(reply), RepliedAt: sql.NullInt64{Int64: at.Unix(), Valid: true}, MeetingID: meetingID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("save reply to meeting %s: %w", meetingID, err)
		}
		if n == 0 {
			return fmt.Errorf("save reply to meeting %s: %w", meetingID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *MeetingsRepo) NextReminder(ctx context.Context) (time.Time, bool, error) {
	v, err := r.q.MeetingNextReminder(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("next meeting reminder: %w", err)
	}
	at, ok, err := optionalInt(v)
	if err != nil || !ok {
		return time.Time{}, false, err
	}
	return time.Unix(at, 0).UTC(), true, nil
}

func (r *MeetingsRepo) RemindersDue(ctx context.Context, now time.Time) ([]*calendar.Meeting, error) {
	rows, err := r.q.MeetingRemindersDue(ctx, sql.NullInt64{Int64: now.Unix(), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("meeting reminders due: %w", err)
	}
	return r.withInvitees(ctx, rows)
}

func (r *MeetingsRepo) MarkReminded(ctx context.Context, m *calendar.Meeting, sent time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MeetingMarkReminded(ctx, sqlcgen.MeetingMarkRemindedParams{
			NextRemindAt: nullUnixPtr(m.NextRemindAt), NextOccurrence: nullUnixPtr(m.NextOccurrence), ID: m.ID,
			NextRemindAt_2: sql.NullInt64{Int64: sent.Unix(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("mark meeting %s reminded: %w", m.ID, err)
		}
		if n == 0 {
			return nil
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *MeetingsRepo) DueTickets(ctx context.Context, fromDate, toDate string) ([]calendar.DueTicket, error) {
	rows, err := r.q.MeetingDueTickets(ctx, sqlcgen.MeetingDueTicketsParams{FromDate: fromDate, ToDate: toDate})
	if err != nil {
		return nil, fmt.Errorf("tickets due: %w", err)
	}
	out := make([]calendar.DueTicket, len(rows))
	for i, row := range rows {
		out[i] = calendar.DueTicket{
			TicketID: row.ID, ProjectID: row.ProjectID.String, WorkspaceID: row.WorkspaceID,
			Key: fmt.Sprintf("%s-%d", row.Prefix, row.Number), Title: row.Title, DueOn: row.DueOn,
			DeveloperID: row.DeveloperID, Timezone: row.Timezone,
		}
	}
	return out, nil
}

func (r *MeetingsRepo) MarkTicketReminded(ctx context.Context, ticketID, dueOn string, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).MeetingTicketReminded(ctx, sqlcgen.MeetingTicketRemindedParams{TicketID: ticketID, DueOn: dueOn, SentAt: at.Unix()})
		if err != nil {
			return fmt.Errorf("mark ticket %s reminded: %w", ticketID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

// withInvitees converts rows to meetings, reading every row's invitees in one statement.
func (r *MeetingsRepo) withInvitees(ctx context.Context, rows []sqlcgen.Meeting) ([]*calendar.Meeting, error) {
	out := make([]*calendar.Meeting, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]string, len(rows))
	byID := make(map[string]*calendar.Meeting, len(rows))
	for i, row := range rows {
		out[i] = toMeeting(row)
		ids[i] = row.ID
		byID[row.ID] = out[i]
	}
	invitees, err := r.q.MeetingInviteesOf(ctx, idsJSON(ids))
	if err != nil {
		return nil, fmt.Errorf("read meeting invitees: %w", err)
	}
	for _, inv := range invitees {
		m := byID[inv.MeetingID]
		m.Invitees = append(m.Invitees, calendar.Invitee{UserID: inv.UserID, Reply: calendar.Reply(inv.Reply), RepliedAt: unixPtrFromNull(inv.RepliedAt)})
	}
	return out, nil
}

func toMeeting(row sqlcgen.Meeting) *calendar.Meeting {
	return &calendar.Meeting{
		ID: row.ID, WorkspaceID: row.WorkspaceID, OrganiserID: row.OrganiserID, Title: row.Title, Notes: row.Notes,
		StartsAt: time.Unix(row.StartsAt, 0).UTC(), EndsAt: time.Unix(row.EndsAt, 0).UTC(), Timezone: row.Timezone,
		VoiceChannelID: row.VoiceChannelID, Link: row.Link, Repeats: calendar.Repeat(row.Repeats),
		RepeatUntil: unixPtrFromNull(row.RepeatUntil), RemindMinutes: int(row.RemindMinutes), SeriesID: row.SeriesID,
		ReplacesAt: unixPtrFromNull(row.ReplacesAt), CancelledAt: unixPtrFromNull(row.CancelledAt),
		NextRemindAt: unixPtrFromNull(row.NextRemindAt), NextOccurrence: unixPtrFromNull(row.NextOccurrence),
		Invitees: []calendar.Invitee{}, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func meetingParams(m *calendar.Meeting) sqlcgen.MeetingUpsertParams {
	return sqlcgen.MeetingUpsertParams{
		ID: m.ID, WorkspaceID: m.WorkspaceID, OrganiserID: m.OrganiserID, Title: m.Title, Notes: m.Notes,
		StartsAt: m.StartsAt.Unix(), EndsAt: m.EndsAt.Unix(), Timezone: m.Timezone, VoiceChannelID: m.VoiceChannelID,
		Link: m.Link, Repeats: string(m.Repeats), RepeatUntil: nullUnixPtr(m.RepeatUntil), RemindMinutes: int64(m.RemindMinutes),
		SeriesID: m.SeriesID, ReplacesAt: nullUnixPtr(m.ReplacesAt), CancelledAt: nullUnixPtr(m.CancelledAt),
		NextRemindAt: nullUnixPtr(m.NextRemindAt), NextOccurrence: nullUnixPtr(m.NextOccurrence),
		CreatedAt: m.CreatedAt.Unix(), UpdatedAt: m.UpdatedAt.Unix(),
	}
}

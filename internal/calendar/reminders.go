package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	// reminderRetry is how soon a failed pass runs again.
	reminderRetry = 5 * time.Second
	// lateReminder is how long after an occurrence starts its reminder still goes, for a 0-minute one or a slow pass.
	lateReminder = 5 * time.Minute
)

// RunReminderLoop sends each reminder when it falls due, sleeping until the next one or a commit that may plan an
// earlier one, until ctx ends.
func (s *Service) RunReminderLoop(ctx context.Context, wake func() <-chan struct{}, logger *slog.Logger) {
	for {
		committed := wake()
		wait, err := s.SendReminders(ctx)
		if err != nil {
			logger.Error("send calendar reminders", "error", err)
			wait = reminderRetry
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-committed:
		case <-timer.C:
		}
		timer.Stop()
	}
}

// SendReminders sends every meeting and due-date reminder due by now and returns how long until the next one; with
// none planned it waits for the next day, when another ticket's date may come into reach.
func (s *Service) SendReminders(ctx context.Context) (time.Duration, error) {
	now := s.now().UTC()
	if err := s.sendMeetingReminders(ctx, now); err != nil {
		return 0, err
	}
	nextTicket, err := s.sendDueDateReminders(ctx, now)
	if err != nil {
		return 0, err
	}
	next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	if nextTicket.Before(next) {
		next = nextTicket
	}
	at, ok, err := s.repo.NextReminder(ctx)
	if err != nil {
		return 0, fmt.Errorf("next meeting reminder: %w", err)
	}
	if ok && at.Before(next) {
		next = at
	}
	return max(next.Sub(now), 0), nil
}

func (s *Service) sendMeetingReminders(ctx context.Context, now time.Time) error {
	due, err := s.repo.RemindersDue(ctx, now)
	if err != nil {
		return fmt.Errorf("meeting reminders due: %w", err)
	}
	if len(due) == 0 {
		return nil
	}
	replaced, err := s.replaced(ctx, due)
	if err != nil {
		return err
	}
	for _, m := range due {
		if err := s.remind(ctx, m, now, replaced[m.ID]); err != nil {
			return err
		}
	}
	return nil
}

// remind sends m's planned reminder unless its occurrence is long started, then plans the next one.
func (s *Service) remind(ctx context.Context, m *Meeting, now time.Time, replaced map[int64]bool) error {
	sent := *m.NextRemindAt
	starts := now
	if m.NextOccurrence != nil {
		starts = *m.NextOccurrence
	}
	m.planReminder(starts, replaced)
	if now.After(starts.Add(lateReminder)) {
		return s.repo.MarkReminded(ctx, m, sent)
	}
	ev := ReminderEvent{Meeting: refOf(m), StartsAt: starts, UserIDs: m.Attendees()}
	if err := s.repo.MarkReminded(ctx, m, sent, outboxEvent(TopicReminderDue, ev)); err != nil {
		return fmt.Errorf("send reminder of meeting %s: %w", m.ID, err)
	}
	return nil
}

// sendDueDateReminders tells each developer the day before their ticket is due and returns when the next such
// reminder falls; tickets due between yesterday and the day after tomorrow cover every timezone.
func (s *Service) sendDueDateReminders(ctx context.Context, now time.Time) (time.Time, error) {
	today := now.Truncate(24 * time.Hour)
	dts, err := s.repo.DueTickets(ctx, today.AddDate(0, 0, -1).Format(time.DateOnly), today.AddDate(0, 0, 2).Format(time.DateOnly))
	if err != nil {
		return time.Time{}, fmt.Errorf("tickets due soon: %w", err)
	}
	next := now.Add(48 * time.Hour)
	for _, dt := range dts {
		remind, dueEnd, ok := dueReminderAt(dt.DueOn, dt.Timezone)
		if !ok {
			continue
		}
		if now.Before(remind) {
			next = minTime(next, remind)
			continue
		}
		if !now.Before(dueEnd) {
			if err := s.repo.MarkTicketReminded(ctx, dt.TicketID, dt.DueOn, now); err != nil {
				return time.Time{}, fmt.Errorf("pass over the due date of ticket %s: %w", dt.TicketID, err)
			}
			continue
		}
		ev := DueDateReminderEvent{
			TicketID: dt.TicketID, ProjectID: dt.ProjectID, WorkspaceID: dt.WorkspaceID, Key: dt.Key, Title: dt.Title,
			DueOn: dt.DueOn, UserIDs: []string{dt.DeveloperID},
		}
		if err := s.repo.MarkTicketReminded(ctx, dt.TicketID, dt.DueOn, now, outboxEvent(TopicDueDateReminder, ev)); err != nil {
			return time.Time{}, fmt.Errorf("send the due date reminder of ticket %s: %w", dt.TicketID, err)
		}
	}
	return next, nil
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

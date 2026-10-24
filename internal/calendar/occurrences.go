package calendar

import (
	"time"
)

// maxSteps bounds one expansion, so a daily series far in the past never loops for long.
const maxSteps = 5000

// location is the meeting's timezone, UTC when it no longer loads.
func (m *Meeting) location() *time.Location {
	loc, err := time.LoadLocation(m.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// slot is the series' kth start, stepped in its own timezone so it keeps its wall-clock time; ok is false for a month
// without the series' day.
func (m *Meeting) slot(start time.Time, k int) (time.Time, bool) {
	switch m.Repeats {
	case RepeatDaily:
		return start.AddDate(0, 0, k), true
	case RepeatWeekly:
		return start.AddDate(0, 0, 7*k), true
	case RepeatMonthly:
		s := start.AddDate(0, k, 0)
		return s, s.Day() == start.Day()
	}
	return start, k == 0
}

// firstStep is a step at or before the first slot that can end after from, skipping a long past without walking it.
func (m *Meeting) firstStep(from time.Time) int {
	behind := from.Sub(m.EndsAt)
	if behind <= 0 {
		return 0
	}
	var k int
	switch m.Repeats {
	case RepeatDaily:
		k = int(behind / (24 * time.Hour))
	case RepeatWeekly:
		k = int(behind / (7 * 24 * time.Hour))
	case RepeatMonthly:
		k = int(behind / (31 * 24 * time.Hour))
	}
	return max(k-1, 0)
}

// Occurrences are the times m takes place that overlap [from, to), in order, leaving out the slots replaced holds;
// a zero to stops after the first.
func (m *Meeting) Occurrences(from, to time.Time, replaced map[int64]bool) []Occurrence {
	length := m.EndsAt.Sub(m.StartsAt)
	start := m.StartsAt.In(m.location())
	var out []Occurrence
	for k, steps := m.firstStep(from), 0; steps < maxSteps; k, steps = k+1, steps+1 {
		s, ok := m.slot(start, k)
		if !to.IsZero() && !s.Before(to) {
			break
		}
		if m.RepeatUntil != nil && s.After(*m.RepeatUntil) {
			break
		}
		if !m.Repeating() && k > 0 {
			break
		}
		if !ok || replaced[s.Unix()] || !s.Add(length).After(from) {
			continue
		}
		out = append(out, Occurrence{MeetingID: m.ID, Slot: s.UTC(), StartsAt: s.UTC(), EndsAt: s.Add(length).UTC()})
		if to.IsZero() {
			break
		}
	}
	return out
}

// HasSlot says the series puts an occurrence at slot.
func (m *Meeting) HasSlot(slot time.Time, replaced map[int64]bool) bool {
	got := m.Occurrences(slot, slot.Add(time.Second), replaced)
	return len(got) == 1 && got[0].Slot.Equal(slot)
}

// planReminder sets when m's next reminder goes: before the first occurrence that starts after after, or none for a
// cancelled meeting or one with nothing left to remind of.
func (m *Meeting) planReminder(after time.Time, replaced map[int64]bool) {
	m.NextRemindAt, m.NextOccurrence = nil, nil
	if m.Cancelled() {
		return
	}
	// An occurrence ending after after plus its length is one starting after after.
	next := m.Occurrences(after.Add(m.EndsAt.Sub(m.StartsAt)), time.Time{}, replaced)
	if len(next) == 0 {
		return
	}
	at := next[0].StartsAt.Add(-time.Duration(m.RemindMinutes) * time.Minute)
	slot := next[0].Slot
	m.NextRemindAt, m.NextOccurrence = &at, &slot
}

// dueReminderAt is when a ticket's developer hears of its due date: 09:00 the day before, in their own timezone.
func dueReminderAt(dueOn, timezone string) (remind, dueEnd time.Time, ok bool) {
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		loc = time.UTC
	}
	day, err := time.ParseInLocation(time.DateOnly, dueOn, loc)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	return day.AddDate(0, 0, -1).Add(9 * time.Hour), day.AddDate(0, 0, 1), true
}

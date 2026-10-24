import { dayKey, occurrencesByDay, type CalendarRange, type Meeting, type Occurrence } from "@nexul/client-core/calendar";

import { formatDayLabel } from "@/lib/time";
import type { Ticket } from "@/models/Ticket";

const AGENDA_DAYS = 30;

export interface AgendaWindow {
  // The meeting range: this morning's midnight to the midnight after the last day, in the phone's own zone.
  from: string;
  to: string;
  // The due range: the first and last day, both included.
  fromDay: string;
  toDay: string;
}

// Stable for the whole day, so a query keyed by it does not refetch on every render.
export const agendaWindow = (now: Date): AgendaWindow => {
  const first = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const after = new Date(first.getFullYear(), first.getMonth(), first.getDate() + AGENDA_DAYS);
  const last = new Date(first.getFullYear(), first.getMonth(), first.getDate() + AGENDA_DAYS - 1);
  return { from: first.toISOString(), to: after.toISOString(), fromDay: dayKey(first), toDay: dayKey(last) };
};

export type AgendaItem =
  | { kind: "due"; ticket: Ticket }
  | { kind: "meeting"; meeting: Meeting; occurrence: Occurrence };

export interface AgendaDay {
  key: string;
  label: string;
  items: AgendaItem[];
}

const dateOf = (key: string): Date => {
  const [year = 0, month = 1, day = 1] = key.split("-").map(Number);
  return new Date(year, month - 1, day);
};

const dayLabel = (key: string, now: Date): string => {
  const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
  if (key === dayKey(tomorrow)) return "Tomorrow";
  return formatDayLabel(dateOf(key).toISOString(), now.getTime());
};

// Meetings by the local day each occurrence starts on and due tickets by their due day, a day's due tickets first
// (they last all day) and then its meetings in start order; a day with neither is left out.
export const agendaDays = (range: CalendarRange, due: Ticket[], now: Date): AgendaDay[] => {
  const meetings = new Map(range.meetings.map((meeting) => [meeting.id, meeting]));
  const byDay = new Map<string, AgendaItem[]>();
  for (const ticket of due) {
    if (!ticket.due_on) continue;
    byDay.set(ticket.due_on, [...(byDay.get(ticket.due_on) ?? []), { kind: "due", ticket }]);
  }
  for (const [key, occurrences] of occurrencesByDay(range.occurrences)) {
    const items = occurrences.flatMap((occurrence): AgendaItem[] => {
      const meeting = meetings.get(occurrence.meeting_id);
      return meeting ? [{ kind: "meeting", meeting, occurrence }] : [];
    });
    byDay.set(key, [...(byDay.get(key) ?? []), ...items]);
  }
  return [...byDay.entries()]
    .filter(([, items]) => items.length > 0)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, items]) => ({ key, label: dayLabel(key, now), items }));
};

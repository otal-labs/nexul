import { useMemo } from "react";

import { dayKey, occurrencesByDay, viewDays, viewRange, type Meeting, type Occurrence } from "@nexul/client-core/calendar";

import { useFetchCalendarRange, useFetchDueTickets } from "@/hooks/CalendarHooks";
import { useCalendarStore } from "@/stores/calendarStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Ticket } from "@/models/Ticket";

// One time a meeting takes place, with the meeting it belongs to.
export interface MeetingEntry {
  occurrence: Occurrence;
  meeting: Meeting;
}

export interface DayEntries {
  due: Ticket[];
  meetings: MeetingEntry[];
}

const noEntries: DayEntries = { due: [], meetings: [] };

export const entriesOn = (byDay: Map<string, DayEntries>, day: Date): DayEntries => byDay.get(dayKey(day)) ?? noEntries;

export interface ShownEntries extends DayEntries {
  hiddenDue: number;
  hiddenMeetings: number;
}

// Due tickets lead but leave a busy day two lines of meetings at least; a band with room to spare gives it to the other.
export const shownIn = (entries: DayEntries, lines: number): ShownEntries => {
  const dueRoom = Math.max(Math.min(2, lines - 2), lines - entries.meetings.length);
  const due = entries.due.slice(0, dueRoom);
  const meetings = entries.meetings.slice(0, lines - due.length);
  return { due, meetings, hiddenDue: entries.due.length - due.length, hiddenMeetings: entries.meetings.length - meetings.length };
};

// What the view shows; its days come from the view alone, so the grid stands while its entries load.
export const useCalendarEntries = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const view = useCalendarStore((s) => s.view);
  const anchor = useCalendarStore((s) => s.anchor);
  const days = useMemo(() => viewDays(view, anchor), [view, anchor]);
  const { from, to } = useMemo(() => viewRange(view, anchor), [view, anchor]);
  const first = dayKey(days[0] ?? anchor);
  const last = dayKey(days[days.length - 1] ?? anchor);
  const range = useFetchCalendarRange(workspaceId, from.toISOString(), to.toISOString());
  const due = useFetchDueTickets(workspaceId, first, last);

  const byDay = useMemo(() => {
    const meetings = new Map((range.data?.meetings ?? []).map((m) => [m.id, m]));
    const out = new Map<string, DayEntries>();
    for (const [key, occurrences] of occurrencesByDay(range.data?.occurrences ?? [])) {
      const entries = occurrences.flatMap((occurrence) => {
        const meeting = meetings.get(occurrence.meeting_id);
        return meeting ? [{ occurrence, meeting }] : [];
      });
      out.set(key, { due: [], meetings: entries });
    }
    for (const ticket of due.data ?? []) {
      const day = out.get(ticket.due_on) ?? { due: [], meetings: [] };
      out.set(ticket.due_on, { ...day, due: [...day.due, ticket] });
    }
    return out;
  }, [range.data, due.data]);

  return { view, days, byDay, error: range.error ?? due.error };
};

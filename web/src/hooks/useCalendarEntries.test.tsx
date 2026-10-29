// @vitest-environment node
import { describe, expect, it } from "vitest";

import { shownIn, type DayEntries, type MeetingEntry } from "@/hooks/useCalendarEntries";
import { dueTicket, meetingOf, occurrenceOf } from "@/test/calendarApi";

const day = (due: number, meetings: number): DayEntries => ({
  due: Array.from({ length: due }, (_, i) => dueTicket({ id: `t-${i}` })),
  meetings: Array.from({ length: meetings }, (_, i): MeetingEntry => {
    const meeting = meetingOf({ id: `m-${i}` });
    return { meeting, occurrence: occurrenceOf(meeting) };
  }),
});

const counts = (entries: DayEntries, lines: number) => {
  const shown = shownIn(entries, lines);
  return [shown.due.length, shown.meetings.length, shown.hiddenDue, shown.hiddenMeetings];
};

describe("shownIn", () => {
  it("keeps three lines for a busy day's meetings and counts what each band hides", () => {
    expect(counts(day(3, 6), 5)).toEqual([2, 3, 1, 3]);
  });

  it("gives due tickets the lines meetings don't need", () => {
    expect(counts(day(4, 1), 5)).toEqual([4, 1, 0, 0]);
    expect(counts(day(6, 0), 4)).toEqual([4, 0, 2, 0]);
  });

  it("gives meetings the lines due tickets don't need", () => {
    expect(counts(day(0, 7), 4)).toEqual([0, 4, 0, 3]);
  });
});

import { dayKey, type CalendarView } from "@nexul/client-core/calendar";

import { formatDateParts } from "@/utils/TimeUtility";

const timeFormat = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" });
const dayFormat = new Intl.DateTimeFormat(undefined, { weekday: "long", day: "numeric", month: "long" });
const shortDayOptions: Intl.DateTimeFormatOptions = { day: "numeric", month: "short" };
const monthFormat = new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" });
const weekdayFormat = new Intl.DateTimeFormat(undefined, { weekday: "short" });
const weekdayNameFormat = new Intl.DateTimeFormat(undefined, { weekday: "long" });

// Every time on the calendar is shown in the zone the browser runs in, zero-padded so a column of times lines up.
export const formatTime = (iso: string): string => timeFormat.format(new Date(iso));

export const formatTimeRange = (startsAt: string, endsAt: string): string => `${formatTime(startsAt)}–${formatTime(endsAt)}`;

// How many local days later a meeting ends than it starts: 1 for one that runs past midnight.
export const daysLater = (startsAt: string, endsAt: string): number => {
  const [start, end] = [new Date(startsAt), new Date(endsAt)];
  const midnight = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  // An end at midnight sharp closes the day it started.
  const last = new Date(end.getTime() - 1);
  return Math.max(0, Math.round((midnight(last) - midnight(start)) / 86_400_000));
};

interface Span {
  starts_at: string;
  ends_at: string;
}

// For spans in start order, whether each starts before an earlier one has ended.
export const overlapsEarlier = (spans: Span[]): boolean[] => {
  let end = -Infinity;
  return spans.map((span) => {
    const clash = Date.parse(span.starts_at) < end;
    end = Math.max(end, Date.parse(span.ends_at));
    return clash;
  });
};

// "Monday 26 October".
export const formatLongDay = (iso: string): string => dayFormat.format(new Date(iso));

export const formatWeekday = (day: Date): string => weekdayFormat.format(day);

export const formatWeekdayName = (day: Date): string => weekdayNameFormat.format(day);

export const formatDayOfMonth = (day: Date): string => String(day.getDate());

export const formatShortDay = (day: Date): string => formatDateParts(day, shortDayOptions);

// "October 2026" for a month, "26 Oct – 1 Nov" for a week.
export const periodTitle = (view: CalendarView, anchor: Date, days: Date[]): string => {
  if (view === "month") return monthFormat.format(anchor);
  const [first, last] = [days[0], days[days.length - 1]];
  return first && last ? `${formatDateParts(first, shortDayOptions)} – ${formatDateParts(last, shortDayOptions)}` : "";
};

// The reader's local date and time as the form's inputs write them, and back to the instant they name.
export const dateInput = (iso: string): string => dayKey(new Date(iso));

export const timeInput = (iso: string): string => {
  const d = new Date(iso);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
};

const parts = (value: string): number[] => value.split(/[-:]/).map(Number);

export const instantOf = (date: string, time: string): string => {
  const [y = 0, m = 1, d = 1] = parts(date);
  const [hh = 0, mm = 0] = parts(time);
  return new Date(y, m - 1, d, hh, mm).toISOString();
};

// A series that ends on a day runs through that whole day.
export const endOfDayInstant = (date: string): string => {
  const [y = 0, m = 1, d = 1] = parts(date);
  return new Date(y, m - 1, d, 23, 59, 59).toISOString();
};

export const sameDay = (a: Date, b: Date): boolean => dayKey(a) === dayKey(b);

import { dayKey, type CalendarView } from "@nexul/client-core/calendar";

const timeFormat = new Intl.DateTimeFormat(undefined, { hour: "numeric", minute: "2-digit" });
const dayFormat = new Intl.DateTimeFormat(undefined, { weekday: "long", day: "numeric", month: "long" });
const shortDayFormat = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" });
const monthFormat = new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" });
const weekdayFormat = new Intl.DateTimeFormat(undefined, { weekday: "short" });

// Every time on the calendar is shown in the zone the browser runs in.
export const formatTime = (iso: string): string => timeFormat.format(new Date(iso));

export const formatTimeRange = (startsAt: string, endsAt: string): string => `${formatTime(startsAt)} – ${formatTime(endsAt)}`;

// "Monday 26 October".
export const formatLongDay = (iso: string): string => dayFormat.format(new Date(iso));

export const formatWeekday = (day: Date): string => weekdayFormat.format(day);

export const formatDayOfMonth = (day: Date): string => String(day.getDate());

export const formatShortDay = (day: Date): string => shortDayFormat.format(day);

// "October 2026" for a month, "26 Oct – 1 Nov" for a week.
export const periodTitle = (view: CalendarView, anchor: Date, days: Date[]): string => {
  if (view === "month") return monthFormat.format(anchor);
  const [first, last] = [days[0], days[days.length - 1]];
  return first && last ? `${shortDayFormat.format(first)} – ${shortDayFormat.format(last)}` : "";
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

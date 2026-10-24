const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

// Compact buckets for dense phone rows: "now", "5m", "3h", "2d".
export const formatRelativeTime = (iso: string, now: number = Date.now()): string => {
  const diffMs = now - new Date(iso).getTime();
  if (diffMs < MINUTE_MS) return "now";
  if (diffMs < HOUR_MS) return `${Math.floor(diffMs / MINUTE_MS)}m`;
  if (diffMs < DAY_MS) return `${Math.floor(diffMs / HOUR_MS)}h`;
  return `${Math.floor(diffMs / DAY_MS)}d`;
};

const pad = (n: number, width = 2): string => String(n).padStart(width, "0");

// Local wall-clock time at millisecond precision for a deploy log line, matching the web log panel.
export const formatLogTimestamp = (ts: number): string => {
  const d = new Date(ts);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${pad(d.getMilliseconds(), 3)}`;
};

// 24-hour wall clock, the minute a bot embed's footer names.
const formatClockTime = (d: Date): string => `${pad(d.getHours())}:${pad(d.getMinutes())}`;

const calendarDay = (d: Date): number => new Date(d).setHours(0, 0, 0, 0);

// Discord's footer form: "Today at 21:11", "Yesterday at 21:11", else the date and the minute.
export const formatCalendarTime = (iso: string, now: number = Date.now()): string => {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  const days = Math.round((calendarDay(new Date(now)) - calendarDay(date)) / DAY_MS);
  if (days === 0) return `Today at ${formatClockTime(date)}`;
  if (days === 1) return `Yesterday at ${formatClockTime(date)}`;
  return `${date.toLocaleDateString()} ${formatClockTime(date)}`;
};

// The clock time beside a chat author's name.
export const formatMessageTime = (iso: string): string => formatClockTime(new Date(iso));

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "June", "July", "Aug", "Sept", "Oct", "Nov", "Dec"];

// A chat day divider, as the web writes it: "Today", "Yesterday", "Mon 21 Sept", the year only when it is not this one.
export const formatDayLabel = (iso: string, now: number = Date.now()): string => {
  const date = new Date(iso);
  const days = Math.round((calendarDay(new Date(now)) - calendarDay(date)) / DAY_MS);
  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  const label = `${WEEKDAYS[date.getDay()]} ${date.getDate()} ${MONTHS[date.getMonth()]}`;
  return date.getFullYear() === new Date(now).getFullYear() ? label : `${label} ${date.getFullYear()}`;
};

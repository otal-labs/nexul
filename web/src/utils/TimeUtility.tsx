const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

// Uncapped; surfaces needing a day-based cutover compute `daysAgo` and layer their own fallback on top.
export const formatRelativeTime = (ts: string): string => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  const diffMs = Date.now() - date.getTime();
  if (diffMs < MINUTE_MS) return "just now";
  if (diffMs < HOUR_MS) return `${Math.floor(diffMs / MINUTE_MS)}m ago`;
  if (diffMs < DAY_MS) return `${Math.floor(diffMs / HOUR_MS)}h ago`;
  return `${Math.floor(diffMs / DAY_MS)}d ago`;
};

// The shared gate callers use to decide when to switch from the relative form above to their own fallback.
export const daysAgo = (ts: string): number => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return NaN;
  return Math.floor((Date.now() - date.getTime()) / DAY_MS);
};

// The mirror of daysAgo for expiry-style countdowns; negative once past.
export const daysUntil = (ts: string): number => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return NaN;
  return Math.ceil((date.getTime() - Date.now()) / DAY_MS);
};

// Automation run durations are always sub-minute (30s default timeout), so only two buckets are needed.
export const formatDurationMs = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
};

// The exact moment as a tooltip: the relative forms above lose it.
export const formatFullTime = (ts: string): string => {
  const date = new Date(ts);
  return Number.isNaN(date.getTime()) ? ts : date.toLocaleString();
};

// 24-hour wall clock, short enough for the gutter a grouped message reveals it in.
export const formatClockTime = (ts: string): string => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hourCycle: "h23" });
};

const calendarDay = (date: Date): number => new Date(date).setHours(0, 0, 0, 0);

// Discord's footer form: "Today at 21:11", "Yesterday at 21:11", else the date and the minute.
export const formatCalendarTime = (ts: string): string => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  const clock = formatClockTime(ts);
  const days = Math.round((calendarDay(new Date()) - calendarDay(date)) / DAY_MS);
  if (days === 0) return `Today at ${clock}`;
  if (days === 1) return `Yesterday at ${clock}`;
  return `${date.toLocaleDateString()} ${clock}`;
};

// A day for a list's meta line: "8 Oct", with the year only when it isn't this one.
export const formatShortDate = (ts: string): string => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  const sameYear = date.getFullYear() === new Date().getFullYear();
  return date.toLocaleDateString(undefined, { day: "numeric", month: "short", ...(sameYear ? {} : { year: "numeric" }) });
};

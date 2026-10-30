const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

// en-GB writes September "Sept", four letters where every other month has three; it is cut back so a column of dates keeps one width.
export const formatDateParts = (date: Date, options: Intl.DateTimeFormatOptions, locale?: string): string =>
  new Intl.DateTimeFormat(locale, options)
    .formatToParts(date)
    .map((part) => (part.type === "month" && part.value === "Sept" ? "Sep" : part.value))
    .join("");

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

// The same buckets without the "ago", for a meta column: "now", "5m", "3h", "2d".
export const formatAge = (ts: string): string => formatRelativeTime(ts).replace("just now", "now").replace(" ago", "");

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

// How long a call has run: m:ss, then h:mm:ss from the first hour.
export const formatCallElapsed = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600);
  const mm = Math.floor((seconds % 3600) / 60);
  const ss = String(seconds % 60).padStart(2, "0");
  if (hours === 0) return `${mm}:${ss}`;
  return `${hours}:${String(mm).padStart(2, "0")}:${ss}`;
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
  return formatDateParts(date, { day: "numeric", month: "short", ...(sameYear ? {} : { year: "numeric" }) });
};

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

// A right-aligned time column: "29 Oct 15:20", a one-digit day padded with a figure space and every month three letters,
// so the column keeps one width.
export const formatColumnTime = (ts: string): string => {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  return `${String(date.getDate()).padStart(2, "\u2007")} ${MONTHS[date.getMonth()]} ${formatClockTime(ts)}`;
};

// A version's time: "Today 14:23", "Yesterday 14:23", else the short date and the clock, "25 Oct 14:23".
export const formatVersionTime = (ts: string): string => {
  const calendar = formatCalendarTime(ts);
  if (calendar === ts) return calendar;
  if (/^(Today|Yesterday) at /.test(calendar)) return calendar.replace(" at ", " ");
  return `${formatShortDate(ts)} ${formatClockTime(ts)}`;
};

// A YYYY-MM-DD day as the reader's calendar writes it, "30 Oct 2026"; the day is read in local time, where Date would take it as UTC and show the evening before.
export const formatDay = (day: string): string => {
  const [year, month, date] = day.split("-").map(Number);
  if (!year || !month || !date) return day;
  return formatDateParts(new Date(year, month - 1, date), { day: "numeric", month: "short", year: "numeric" });
};

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

const DISCORD_STYLES: Record<string, Intl.DateTimeFormatOptions> = {
  t: { timeStyle: "short" },
  T: { timeStyle: "medium" },
  d: { dateStyle: "short" },
  D: { dateStyle: "long" },
  f: { dateStyle: "long", timeStyle: "short" },
  F: { dateStyle: "full", timeStyle: "short" },
};

const RELATIVE_UNITS: [string, number][] = [
  ["year", 365 * DAY_MS],
  ["month", 30 * DAY_MS],
  ["day", DAY_MS],
  ["hour", HOUR_MS],
  ["minute", MINUTE_MS],
];

// Hermes has no Intl.RelativeTimeFormat, so the "R" style is spelled out in English.
const formatRelativeDistance = (date: Date, now: number): string => {
  const diffMs = date.getTime() - now;
  const [unit, size] = RELATIVE_UNITS.find(([, ms]) => Math.abs(diffMs) >= ms) ?? ["second", 1000];
  const n = Math.round(Math.abs(diffMs) / size);
  const span = `${n} ${unit}${n === 1 ? "" : "s"}`;
  return diffMs >= 0 ? `in ${span}` : `${span} ago`;
};

// A Discord <t:unix:style> code in the reader's own zone; no style reads as Discord's default "f".
export const formatDiscordTimestamp = (date: Date, style: string | undefined, now: number = Date.now()): string => {
  if (style === "R") return formatRelativeDistance(date, now);
  return date.toLocaleString(undefined, DISCORD_STYLES[style ?? "f"]);
};

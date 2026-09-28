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

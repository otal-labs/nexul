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

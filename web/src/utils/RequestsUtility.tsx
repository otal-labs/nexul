import type { StackRequests } from "@/models/Stack";

// A stack that has gone this many minutes without a count reads "No request data since".
export const MISSING_AFTER_MINUTES = 5;

const MINUTE_MS = 60_000;

export interface RequestTotals {
  requests: number;
  errors: number;
  // The 5xx share of the counted requests, 0 when there were none.
  share: number;
  counted: number;
}

export const requestTotals = (w: StackRequests): RequestTotals => {
  let requests = 0;
  let errors = 0;
  let counted = 0;
  w.requests.forEach((r, i) => {
    if (r === null) return;
    counted += 1;
    requests += r;
    errors += w.errors_5xx[i] ?? 0;
  });
  return { requests, errors, share: requests > 0 ? errors / requests : 0, counted };
};

export const minuteAt = (w: StackRequests, index: number): string => new Date(new Date(w.from).getTime() + index * MINUTE_MS).toISOString();

// The first minute of a gap that runs to the end of the hour and has lasted five minutes; undefined when the hour has
// no counted minute at all (that is the empty state) or the gap is shorter.
export const missingSince = (w: StackRequests): string | undefined => {
  const last = w.requests.reduce<number>((at, r, i) => (r === null ? at : i), -1);
  const gap = w.requests.length - 1 - last;
  if (last < 0 || gap < MISSING_AFTER_MINUTES) return undefined;
  return minuteAt(w, last + 1);
};

// "0.4%", "12%": one decimal under 10%, none above, "<0.1%" for a trickle.
export const formatShare = (share: number): string => {
  const percent = share * 100;
  if (percent > 0 && percent < 0.1) return "<0.1%";
  return `${percent >= 10 ? Math.round(percent) : Number(percent.toFixed(1))}%`;
};

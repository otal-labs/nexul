export type Measure = (text: string) => number;

const ELLIPSIS = "…";

// The largest n in [low, high] whose candidate still fits, or low - 1 when none does; candidates grow with n.
const largestFitting = (low: number, high: number, fits: (n: number) => boolean): number => {
  let best = low - 1;
  while (low <= high) {
    const mid = (low + high) >> 1;
    if (fits(mid)) {
      best = mid;
      low = mid + 1;
      continue;
    }
    high = mid - 1;
  }
  return best;
};

// "head…tail" cut on a character, so the ellipsis touches the kept tail with no gap.
export const fitTail = (head: string, tail: string, width: number, measure: Measure): string => {
  if (measure(head + tail) <= width) return head + tail;
  const n = largestFitting(0, head.length - 1, (k) => measure(head.slice(0, k) + ELLIPSIS + tail) <= width);
  return head.slice(0, Math.max(n, 0)) + ELLIPSIS + tail;
};

// Whole words and an ellipsis, so a cut never leaves a one-letter fragment; a first word too long is cut by character.
export const fitWords = (text: string, width: number, measure: Measure): string => {
  if (measure(text) <= width) return text;
  const cuts = [...text.matchAll(/\s+/g)].map((m) => m.index);
  const words = largestFitting(0, cuts.length - 1, (i) => measure(text.slice(0, cuts[i]) + ELLIPSIS) <= width);
  if (words >= 0) return text.slice(0, cuts[words]) + ELLIPSIS;
  const n = largestFitting(1, text.length - 1, (k) => measure(text.slice(0, k) + ELLIPSIS) <= width);
  return text.slice(0, Math.max(n, 1)) + ELLIPSIS;
};

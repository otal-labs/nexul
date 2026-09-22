import { useNow } from "@/hooks/useNow";

// Whole seconds since startedAt, re-read once a second while ticking; a finished span reads once and stops.
export const useElapsedSeconds = (startedAt: number, ticking = true): number => {
  const now = useNow(ticking);
  return Math.max(0, Math.round((now - startedAt) / 1_000));
};

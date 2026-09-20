import { useEffect, useState } from "react";

// Whole seconds since startedAt, re-read once a second while ticking; a finished span reads once and stops.
export const useElapsedSeconds = (startedAt: number, ticking = true): number => {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!ticking) return;
    const timer = setInterval(() => setNow(Date.now()), 1_000);
    return () => clearInterval(timer);
  }, [ticking]);
  return Math.max(0, Math.round((now - startedAt) / 1_000));
};

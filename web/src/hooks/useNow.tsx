import { useEffect, useState } from "react";

// Wall-clock milliseconds, re-read once a second while ticking; a finished span reads once and stops.
export const useNow = (ticking: boolean): number => {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!ticking) return;
    const timer = setInterval(() => setNow(Date.now()), 1_000);
    return () => clearInterval(timer);
  }, [ticking]);
  return now;
};

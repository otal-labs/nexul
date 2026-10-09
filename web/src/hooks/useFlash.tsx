import { useCallback, useEffect, useRef, useState } from "react";

const HOLD_MS = 1400;

// A state that holds for a moment, then clears: "Saved" after a save lands, the check after a copy.
export const useFlash = (): [boolean, () => void] => {
  const [on, setOn] = useState(false);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(timer.current), []);
  const flash = useCallback(() => {
    setOn(true);
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setOn(false), HOLD_MS);
  }, []);
  return [on, flash];
};

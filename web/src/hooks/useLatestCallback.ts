import { useCallback, useLayoutEffect, useRef } from "react";

// One identity for the component's lifetime that always calls the latest fn, so a memoized child never re-renders for a new closure.
export const useLatestCallback = <Args extends unknown[], Result>(fn: (...args: Args) => Result) => {
  const latest = useRef(fn);
  useLayoutEffect(() => {
    latest.current = fn;
  });
  return useCallback((...args: Args) => latest.current(...args), []);
};

import { useCallback, useSyncExternalStore } from "react";

// Whether the viewport is at least this wide; true where the browser has no matchMedia, so a wide layout is the default.
export const useMinWidth = (px: number): boolean => {
  const query = `(min-width: ${px}px)`;
  const subscribe = useCallback(
    (notify: () => void) => {
      const list = window.matchMedia?.(query);
      list?.addEventListener("change", notify);
      return () => list?.removeEventListener("change", notify);
    },
    [query],
  );
  return useSyncExternalStore(subscribe, () => window.matchMedia?.(query).matches ?? true);
};

import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useRef } from "react";

export const HOVER_INTENT_MS = 150;

// Resting on a row, or focusing it, loads what it opens, so the open usually finds it cached; a pointer passing over does not.
export const useHoverPrefetch = (prefetch: (client: QueryClient) => unknown) => {
  const client = useQueryClient();
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const cancel = () => clearTimeout(timer.current);
  const run = () => {
    cancel();
    void prefetch(client);
  };
  return {
    onPointerEnter: () => {
      cancel();
      timer.current = setTimeout(run, HOVER_INTENT_MS);
    },
    onPointerLeave: cancel,
    onFocus: run,
  };
};

import { useCallback, useRef } from "react";

import { rowGlide } from "@/lib/motion";

export const leavingRowClass = "transition-[opacity,scale,background-color] duration-150 ease-standard data-[leaving]:scale-[0.98] data-[leaving]:opacity-0";

// Wraps a list whose rows leave on request (a revoked token, a signed-out device): ref the wrapper, prepare() on leave.
export const useRowGlide = () => {
  const glide = useRef<ReturnType<typeof rowGlide> | undefined>(undefined);
  const ref = useCallback((el: HTMLElement | null) => {
    glide.current?.disconnect();
    glide.current = el ? rowGlide(el) : undefined;
  }, []);
  const prepare = useCallback(() => glide.current?.prepare(), []);
  return { ref, prepare };
};

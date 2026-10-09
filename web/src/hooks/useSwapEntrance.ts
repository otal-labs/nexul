import { useLayoutEffect, useRef } from "react";

import { swapIn, type SwapAxis } from "@/lib/motion";

// Plays the swap entrance on the target when `key` changes after mount; `order` is the picked item's place in its list.
export const useSwapEntrance = (key: string, order: () => number, target: () => Element | null | undefined, axis: () => SwapAxis) => {
  const previous = useRef<number | null>(null);
  useLayoutEffect(() => {
    const from = previous.current;
    const to = order();
    previous.current = to;
    if (from === null || from === to) return;
    swapIn(target(), to - from, axis());
    // Keyed on the picked item alone; its order, the target and the axis are read at the moment it changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
};

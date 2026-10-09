import { useCallback, type HTMLAttributes } from "react";

import { enterList, type Arrival } from "@/lib/motion";

interface EnterListProps extends HTMLAttributes<HTMLElement> {
  as?: "ul" | "ol" | "div";
  /** How a row added after mount comes in: back from a filter (pop) or news arriving (rise). */
  arrival?: Arrival;
}

// The one list entrance: rows cascade in on mount and rows added later pop in (lib/motion.ts); a row opts out with data-no-enter.
export const EnterList = ({ as: Tag = "ul", arrival = "pop", ...props }: EnterListProps) => {
  // Fixed for the list's life: the observer it starts keeps the mode it was mounted with.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const ref = useCallback((el: HTMLElement | null) => (el ? enterList(el, arrival) : undefined), []);
  return <Tag ref={ref} data-enter-list="" {...props} />;
};

import { useEffect, useState, type KeyboardEvent, type RefObject } from "react";

import type { HistoryCompare } from "@/hooks/HistoryCompareHooks";

const STEP_KEYS: Record<string, number> = { "[": -1, "]": 1 };

// The stepper over a version's changes: its position, the hop to a change, and the `[` and `]` keys that make one.
export const useChangeStepper = (scroller: RefObject<HTMLElement | null>, { runs, changes }: Pick<HistoryCompare, "runs" | "changes">) => {
  const [step, setStep] = useState(1);

  const run = (n: number) => scroller.current?.querySelector<HTMLElement>(`[data-change="${n}"]`);

  // New text to read opens on its first change, or at the top when there is none.
  useEffect(() => {
    const first = scroller.current?.querySelector<HTMLElement>('[data-change="1"]');
    if (first) {
      first.scrollIntoView?.({ block: "start" });
      return;
    }
    scroller.current?.scrollTo?.({ top: 0 });
  }, [scroller, runs]);

  const hop = (to: number) => {
    const next = Math.min(Math.max(to, 1), changes);
    const target = run(next);
    setStep(next);
    target?.scrollIntoView?.({ block: "start" });
    target?.focus({ preventScroll: true });
  };

  const onKeyDown = (event: KeyboardEvent) => {
    const direction = STEP_KEYS[event.key];
    if (!direction || changes === 0 || event.metaKey || event.ctrlKey || event.altKey) return;
    event.preventDefault();
    hop(step + direction);
  };

  return { step, hop, onKeyDown, reset: () => setStep(1) };
};

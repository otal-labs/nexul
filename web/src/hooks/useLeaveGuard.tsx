import { useEffect } from "react";
import { useBlocker } from "react-router";

import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useLatestCallback } from "@/hooks/useLatestCallback";

// Leaving a page with held changes asks first; a path that starts with stayWithin is the same page (its tabs).
export const useLeaveGuard = (dirty: boolean, stayWithin: string, onDiscard: () => void) => {
  const blocker = useBlocker(({ nextLocation }) => dirty && !nextLocation.pathname.startsWith(stayWithin));
  const { open } = useConfirmationDialog();
  const discard = useLatestCallback(onDiscard);
  useEffect(() => {
    if (blocker.state !== "blocked") return;
    void open({ title: "Discard changes?", message: "Your ticks on this page haven't been saved.", confirmLabel: "Discard changes" }).then((ok) => {
      if (!ok) {
        blocker.reset();
        return;
      }
      discard();
      blocker.proceed();
    });
  }, [blocker, open, discard]);
};

import { useState } from "react";

import { SweepDigestView } from "@/components/sweep/SweepDigestView";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { useFetchLastSweeps } from "@/hooks/SweepHooks";
import { countsPhrase, inSentence, type Sweep } from "@/models/Schedule";
import { formatCalendarTime } from "@/utils/TimeUtility";

// "Last sweep today at 23:00: 4 finished, 1 asked", or "Sweeping since today at 23:00" while it goes.
const lineOf = (sweep: Sweep) => {
  if (!sweep.finished_at) return `Sweeping since ${inSentence(formatCalendarTime(sweep.fired_at))}`;
  const counts = countsPhrase(sweep.counts);
  return `Last sweep ${inSentence(formatCalendarTime(sweep.fired_at))}${counts ? `: ${counts}` : ": nothing matched"}`;
};

// A schedule auto play's latest sweep under its clauses, opening the sweep's digest; nothing until it has swept.
export const LastSweepLine = ({ playId, autoPlayId }: { playId: string; autoPlayId: string }) => {
  const { data } = useFetchLastSweeps(playId);
  const [open, setOpen] = useState(false);
  const sweep = data?.[autoPlayId];
  if (!sweep) return null;
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        className="inset-focus mb-2 ml-3.5 rounded-sm text-left text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
      >
        {lineOf(sweep)}
      </button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[calc(100dvh-4rem)] overflow-y-auto bg-popover sm:max-w-[min(40rem,calc(100%-2rem))]">
          <DialogTitle className="sr-only">Sweep</DialogTitle>
          <SweepDigestView sweepId={sweep.id} />
        </DialogContent>
      </Dialog>
    </>
  );
};

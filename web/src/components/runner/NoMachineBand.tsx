import { GroupDot } from "@/components/runner/GroupDot";
import type { GroupTone } from "@/utils/RunnerFleetUtility";
import type { ReactNode } from "react";

interface NoMachineBandProps {
  state: { tone: GroupTone; words: string };
  toggle?: ReactNode;
}

// The last group's band, two lines tall like a machine's so the bands keep one rhythm.
export const NoMachineBand = ({ state, toggle }: NoMachineBandProps) => (
  <div className="sticky top-0 z-10 flex items-start gap-3 border-b border-border group-has-[>ul[hidden]]/fleet:border-b-0 bg-card bg-linear-to-b from-well-shade to-well-shade px-4 py-2.5">
    <GroupDot tone={state.tone} />
    <div className="min-w-0 flex-1">
      <div className="flex flex-wrap items-baseline gap-x-2.5">
        <h2 id="machine-none" className="text-sm/5 font-semibold">
          No machine
        </h2>
        <span className="text-xs text-muted-foreground tabular-nums">{state.words}</span>
      </div>
      <p className="text-xs text-muted-foreground">Made before machines, or for one you can't see</p>
    </div>
    <div className="-my-1.5 flex shrink-0 items-center gap-0.5">{toggle}</div>
  </div>
);

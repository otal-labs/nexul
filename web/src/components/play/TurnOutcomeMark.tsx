import { CircleCheck, CircleDashed, Loader2, TriangleAlert } from "lucide-react";

import type { TurnOutcome } from "@nexul/client-core/agentSteps";

interface TurnOutcomeMarkProps {
  outcome: TurnOutcome;
}

// The 16px mark that leads a turn's header and tops its rule; only a failed step takes a status hue. The header's
// words carry the state, so the mark is decoration.
export const TurnOutcomeMark = ({ outcome }: TurnOutcomeMarkProps) => (
  <>
    {outcome === "running" && <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground motion-reduce:animate-none" aria-hidden />}
    {outcome === "failed" && <TriangleAlert className="size-4 shrink-0 text-destructive" aria-hidden />}
    {outcome === "unfinished" && <CircleDashed className="size-4 shrink-0 text-muted-foreground" aria-hidden />}
    {outcome === "done" && <CircleCheck className="size-4 shrink-0 text-muted-foreground" aria-hidden />}
  </>
);

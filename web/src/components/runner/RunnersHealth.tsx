import type { ReactNode } from "react";

import { useRunnerQueue, useRunners } from "@/hooks/RunnerHooks";
import { cn } from "@/lib/utils";
import { fleetCounts } from "@/utils/RunnerFleetUtility";

interface ClauseProps {
  count: ReactNode;
  label: string;
  dot?: string;
}

// Clauses after the first open with a muted ·, so a clause without a dot still reads as its own.
const Clause = ({ count, label, dot }: ClauseProps) => (
  <span className="inline-flex items-baseline gap-1.5 whitespace-nowrap [&+&]:before:text-muted-foreground/50 [&+&]:before:content-['·']">
    {dot && <span aria-hidden className={cn("size-2 shrink-0 self-center rounded-full", dot)} />}
    <span>{count}</span>
    {label}
  </span>
);

const Figure = ({ children }: { children: ReactNode }) => <span className="font-mono tabular-nums">{children}</span>;

// The fleet in one line, each clause led by its status; a clause whose count is zero is left out.
export const RunnersHealth = () => {
  const { data: runners } = useRunners();
  const { data: queue } = useRunnerQueue();
  if (!runners) return null;
  const c = fleetCounts(runners);
  const waiting = queue?.length ?? 0;
  return (
    <span className="flex flex-wrap items-baseline gap-x-1.5 gap-y-1" aria-label="Fleet health">
      {c.total === 0 && "No runners yet"}
      {c.total > 0 && (
        <Clause count={<><Figure>{c.online}</Figure> of <Figure>{c.total}</Figure></>} label="online" dot={c.online > 0 ? "bg-success" : "ring-[1.5px] ring-muted-foreground/60 ring-inset"} />
      )}
      {c.busy > 0 && <Clause count={<Figure>{c.busy}</Figure>} label="busy" dot="bg-info animate-[status-pulse_2.4s_ease-standard_infinite]" />}
      {c.offline > 0 && <Clause count={<Figure>{c.offline}</Figure>} label="offline" dot="ring-[1.5px] ring-muted-foreground/60 ring-inset" />}
      {waiting > 0 && <Clause count={<Figure>{waiting}</Figure>} label="waiting" />}
    </span>
  );
};

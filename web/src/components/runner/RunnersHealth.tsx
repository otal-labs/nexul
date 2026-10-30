import type { ReactNode } from "react";

import { useOpenEnrollments, useRunnerQueue, useRunners } from "@/hooks/RunnerHooks";
import { cn } from "@/lib/utils";
import { fleetCounts } from "@/utils/RunnerFleetUtility";

interface ClauseProps {
  children: ReactNode;
  mark: string;
}

const hollow = "ring-[1.5px] ring-muted-foreground/60 ring-inset";

// Every clause is led by its mark; clauses after the first open with a muted ·.
const Clause = ({ children, mark }: ClauseProps) => (
  <span className="inline-flex items-baseline gap-1.5 whitespace-nowrap [&+&]:before:text-muted-foreground/50 [&+&]:before:content-['·']">
    <span aria-hidden className={cn("size-2 shrink-0 self-center rounded-full", mark)} />
    <span>{children}</span>
  </span>
);

// The fleet in one line of words with tabular figures; a clause whose count is zero is left out.
export const RunnersHealth = () => {
  const { data: runners } = useRunners();
  const { data: queue } = useRunnerQueue();
  const { data: codes } = useOpenEnrollments(false);
  if (!runners) return null;
  const c = fleetCounts(runners);
  const waiting = queue?.length ?? 0;
  const unused = codes?.length ?? 0;
  return (
    <span className="flex flex-wrap items-baseline gap-x-1.5 gap-y-1 tabular-nums" aria-label="Fleet health">
      {c.total === 0 && "No runners yet"}
      {c.total > 0 && <Clause mark={c.online > 0 ? "bg-success" : hollow}>{c.online} of {c.total} online</Clause>}
      {c.busy > 0 && <Clause mark="bg-info animate-[status-pulse_2.4s_ease-standard_infinite]">{c.busy} busy</Clause>}
      {c.offline > 0 && <Clause mark={hollow}>{c.offline} offline</Clause>}
      {unused > 0 && <Clause mark="border border-dashed border-muted-foreground">{unused} unused code{unused === 1 ? "" : "s"}</Clause>}
      {waiting > 0 && <Clause mark="bg-muted-foreground/70">{waiting} waiting</Clause>}
    </span>
  );
};

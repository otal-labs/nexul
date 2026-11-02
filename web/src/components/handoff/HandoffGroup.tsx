import { ChevronRight } from "lucide-react";

import { formatElapsed, turnStartMs } from "@nexul/client-core/agentSteps";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

import { HandoffRow } from "@/components/handoff/HandoffRow";
import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { cn } from "@/lib/utils";
import type { Handoff, HandoffState } from "@/models/Handoff";

interface HandoffGroupProps {
  handoffs: Handoff[];
}

const STATE_WORDS: [HandoffState, string][] = [
  ["running", "working"],
  ["failed", "failed"],
  ["left_running", "left running"],
  ["interrupted", "interrupted"],
  ["done", "done"],
];

const MAX_MARKS = 4;

// "2 working, 1 done": how many agents stand in each state, the ones still running first.
const tally = (handoffs: Handoff[]): string =>
  STATE_WORDS.map(([state, word]) => [handoffs.filter((h) => h.state === state).length, word] as const)
    .filter(([n]) => n > 0)
    .map(([n, word]) => `${n} ${word}`)
    .join(", ");

const known = (ms: number | null): ms is number => ms !== null;

// The agents an Agent reply handed work to as one fold, opening to a row per agent.
export const HandoffGroup = ({ handoffs }: HandoffGroupProps) => {
  const running = handoffs.some((h) => h.state === "running");
  const starts = handoffs.map((h) => turnStartMs(h.steps)).filter(known);
  const ends = handoffs.map((h) => turnStartMs([...h.steps].reverse())).filter(known);
  const startMs = starts.length > 0 ? Math.min(...starts) : null;
  const ticking = useElapsedSeconds(startMs ?? 0, running && startMs !== null);
  const finished = startMs === null || ends.length === 0 ? null : Math.max(0, Math.round((Math.max(...ends) - startMs) / 1_000));
  const seconds = running && startMs !== null ? ticking : finished;
  const count = `${handoffs.length} ${handoffs.length === 1 ? "sub-agent" : "sub-agents"}`;
  return (
    <Collapsible className="mx-1 mt-1 max-w-md overflow-hidden rounded-lg border border-border bg-card">
      <CollapsibleTrigger
        aria-label={`${count}, ${tally(handoffs)}${seconds === null ? "" : `, ${formatElapsed(seconds)}`}`}
        className="group flex w-full items-center gap-2.5 px-2.5 py-1.5 text-left transition-colors duration-150 ease-standard hover:bg-accent">
        <span className="flex shrink-0 -space-x-1" aria-hidden>
          {handoffs.slice(0, MAX_MARKS).map((h) => (
            <span key={h.id} className="flex size-6 items-center justify-center rounded-full bg-surface-2 ring-2 ring-card">
              <HarnessProviderMark driver={h.driver} className="size-3.5" />
            </span>
          ))}
        </span>
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-xs leading-4 font-semibold text-foreground">{count}</span>
          <span className={cn("truncate text-xs leading-4", running ? "text-info" : "text-muted-foreground")}>{tally(handoffs)}</span>
        </span>
        {seconds !== null && <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">{formatElapsed(seconds)}</span>}
        <ChevronRight
          className="size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90"
          aria-hidden
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        {handoffs.map((handoff) => (
          <HandoffRow key={handoff.id} handoff={handoff} />
        ))}
      </CollapsibleContent>
    </Collapsible>
  );
};

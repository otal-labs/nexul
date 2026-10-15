import { ChevronRight } from "lucide-react";

import { formatElapsed, groupSteps, liveStepLabel, turnPhrase, turnSpanSeconds, turnStartMs } from "@nexul/client-core/agentSteps";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

import { TurnStepGroup } from "@/components/play/TurnStepGroup";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import type { ActivityEntry } from "@/models/Trail";

interface TrailTurnGroupProps {
  entries: ActivityEntry[];
  running: boolean;
  from: string | null;
  until: string | null;
}

// One turn of the Agent as a fold: while it runs, the live step sweeps under a highlight beside the clock; once done,
// "Worked for" and what it did. Open while it runs, folded once over, and always openable.
export const TrailTurnGroup = ({ entries, running, from, until }: TrailTurnGroupProps) => {
  const startMs = turnStartMs(entries, from);
  const ticking = useElapsedSeconds(startMs ?? 0, running && startMs !== null);
  const seconds = running ? (startMs === null ? null : ticking) : turnSpanSeconds(entries, until, from);
  const last = entries[entries.length - 1];
  const live = running && last?.kind === "tool_call" ? last : undefined;
  const phrase = turnPhrase(entries);
  const worked = seconds === null ? "Worked" : `Worked for ${formatElapsed(seconds)}`;

  return (
    <Collapsible defaultOpen={running} className="px-1 py-1">
      <CollapsibleTrigger className="group inline-flex max-w-full flex-wrap items-center gap-x-1.5 rounded-md text-left text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
        {running && <span className="shimmer-text">{liveStepLabel(entries)}</span>}
        {!running && <span className="tabular-nums">{worked}</span>}{" "}
        {running && seconds !== null && <span className="tabular-nums">· {formatElapsed(seconds)}</span>}
        {!running && phrase !== "" && <span className="text-muted-foreground/70">· {phrase}</span>}
        <ChevronRight className="size-3.5 shrink-0 transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90" aria-hidden />
      </CollapsibleTrigger>
      <CollapsibleContent>
        {entries.length > 0 && (
          <ul className="mt-1.5 mb-1 ml-1.5 flex flex-col gap-2 border-l border-border pl-3.5">
            {groupSteps(entries).map((group, i) => (
              <TurnStepGroup key={`${group.act}-${i}`} group={group} live={live} entrance={running} />
            ))}
          </ul>
        )}
      </CollapsibleContent>
    </Collapsible>
  );
};

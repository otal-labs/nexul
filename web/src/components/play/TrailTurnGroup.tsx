import { ChevronRight } from "lucide-react";

import {
  formatElapsed,
  groupSteps,
  liveStepLabel,
  runDirectory,
  turnOutcome,
  turnPhrase,
  turnSpanSeconds,
  turnStartMs,
} from "@nexul/client-core/agentSteps";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

import { TurnOutcomeMark } from "@/components/play/TurnOutcomeMark";
import { TurnStepGroup } from "@/components/play/TurnStepGroup";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import type { ActivityEntry } from "@/models/Trail";

interface TrailTurnGroupProps {
  entries: ActivityEntry[];
  running: boolean;
  from: string | null;
  until: string | null;
}

// One turn of the Agent as a fold led by its outcome mark: while it runs, the live step sweeps under a highlight beside
// the clock; once done, "Worked for" in ink and what it did. Open while it runs, folded once over, and always openable;
// opened, the steps hang off a rule dropped from the mark.
export const TrailTurnGroup = ({ entries, running, from, until }: TrailTurnGroupProps) => {
  const startMs = turnStartMs(entries, from);
  const ticking = useElapsedSeconds(startMs ?? 0, running && startMs !== null);
  const seconds = running ? (startMs === null ? null : ticking) : turnSpanSeconds(entries, until, from);
  const last = entries[entries.length - 1];
  const live = running && last?.kind === "tool_call" ? last : undefined;
  const phrase = turnPhrase(entries);
  const worked = seconds === null ? "Worked" : `Worked for ${formatElapsed(seconds)}`;
  const root = runDirectory(entries);
  const outcome = turnOutcome(entries, running);

  return (
    <Collapsible defaultOpen={running} className="px-1 py-1">
      <CollapsibleTrigger className="group flex max-w-full items-start gap-2 rounded-md text-left text-sm leading-5 text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
        <span className="flex h-5 shrink-0 items-center">
          <TurnOutcomeMark outcome={outcome} />
        </span>
        <span className="min-w-0">
          {running && <span className="shimmer-text">{liveStepLabel(entries)}</span>}
          {!running && <span className="font-medium text-foreground tabular-nums">{worked}</span>}
          {running && seconds !== null && (
            <>
              {" · "}
              <span className="tabular-nums">{formatElapsed(seconds)}</span>
            </>
          )}
          {!running && phrase !== "" && ` · ${phrase}`}
          {outcome === "failed" && <span className="sr-only">{", a step failed"}</span>}
        </span>
        <span className="flex h-5 shrink-0 items-center">
          <ChevronRight className="size-3.5 transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90" aria-hidden />
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        {entries.length > 0 && (
          <ul className="mt-1 mb-1 ml-[7px] flex flex-col gap-1.5 border-l border-border py-0.5 pl-4">
            {groupSteps(entries).map((group, i) => (
              <TurnStepGroup key={`${group.act}-${i}`} group={group} live={live} root={root} entrance={running} />
            ))}
          </ul>
        )}
      </CollapsibleContent>
    </Collapsible>
  );
};

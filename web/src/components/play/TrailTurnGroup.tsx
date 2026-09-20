import { ChevronRight } from "lucide-react";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

import { TrailActionRow } from "@/components/play/TrailActionRow";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import type { ActivityEntry } from "@/models/Trail";
import { formatElapsed, turnCounts, turnSpanSeconds, turnStartMs, turnSummary } from "@/utils/TrailTranscriptUtility";

interface TrailTurnGroupProps {
  entries: ActivityEntry[];
  running: boolean;
  from: string | null;
  until: string | null;
}

const headline = (running: boolean, seconds: number | null): string => {
  const verb = running ? "Working" : "Worked";
  if (seconds === null) return verb;
  return `${verb} for ${formatElapsed(seconds)}`;
};

// One turn of the Agent: a "Worked for" header with the step counts under it, expanding to the rows in time order.
// Open while it runs (the newest row spins at the bottom), collapsed once the run is over.
export const TrailTurnGroup = ({ entries, running, from, until }: TrailTurnGroupProps) => {
  const startMs = turnStartMs(entries, from);
  const ticking = useElapsedSeconds(startMs ?? 0, running && startMs !== null);
  const seconds = running ? (startMs === null ? null : ticking) : turnSpanSeconds(entries, until, from);

  return (
    <Collapsible defaultOpen={running} className="px-1 py-1">
      <CollapsibleTrigger className="group flex w-full items-center gap-1.5 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-ring">
        <ChevronRight
          className="size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90"
          aria-hidden
        />
        <span className="text-sm font-medium">{headline(running, seconds)}</span>
      </CollapsibleTrigger>
      <p className="pl-5 text-xs text-muted-foreground">{turnSummary(turnCounts(entries))}</p>
      <CollapsibleContent>
        {entries.length > 0 && (
          <ul className="mt-1.5 flex flex-col gap-1">
            {entries.map((entry, i) => (
              <TrailActionRow
                key={`${entry.call_id || entry.at}-${i}`}
                entry={entry}
                index={i}
                live={running && i === entries.length - 1}
                entrance={running}
              />
            ))}
          </ul>
        )}
      </CollapsibleContent>
    </Collapsible>
  );
};

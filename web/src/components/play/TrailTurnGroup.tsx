import { ChevronRight } from "lucide-react";

import { formatElapsed, runDirectory, segmentSteps, turnOutcome, turnPhrase, turnSpanSeconds, turnStartMs } from "@nexul/client-core/agentSteps";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

import { TurnFeedItem } from "@/components/play/TurnFeedItem";
import { TurnOutcomeMark } from "@/components/play/TurnOutcomeMark";
import { TurnRunLine } from "@/components/play/TurnRunLine";
import { TurnStepRail } from "@/components/play/TurnStepRail";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import type { ActivityEntry } from "@/models/Trail";

interface TrailTurnGroupProps {
  entries: ActivityEntry[];
  running: boolean;
  from: string | null;
  until: string | null;
}

// One turn of the Agent as a fold over a feed of what it said, the tools between two sentences one line each.
export const TrailTurnGroup = ({ entries, running, from, until }: TrailTurnGroupProps) => {
  const startMs = turnStartMs(entries, from);
  const ticking = useElapsedSeconds(startMs ?? 0, running && startMs !== null);
  const seconds = running ? (startMs === null ? null : ticking) : turnSpanSeconds(entries, until, from);
  const phrase = turnPhrase(entries);
  const worked = seconds === null ? "Worked" : `Worked for ${formatElapsed(seconds)}`;
  const working = seconds === null ? "Working" : `Working for ${formatElapsed(seconds)}`;
  const root = runDirectory(entries);
  const feed = segmentSteps(entries, running);
  const outcome = turnOutcome(entries, running);
  // A finished turn the Agent said nothing in lists its steps at once, rather than one line repeating the header.
  const quiet = !running && feed.length === 1 && feed[0]?.kind === "turn";

  return (
    <Collapsible defaultOpen={running} className="px-1 py-1">
      <CollapsibleTrigger className="group flex max-w-full items-start gap-2 rounded-md text-left text-sm leading-5 text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
        {!running && (
          <span className="flex h-5 shrink-0 items-center">
            <TurnOutcomeMark outcome={outcome} />
          </span>
        )}
        <span className="min-w-0">
          {running && <span className="tabular-nums">{working}</span>}
          {!running && <span className="font-medium text-foreground tabular-nums">{worked}</span>}
          {!running && phrase !== "" && ` · ${phrase}`}
          {outcome === "failed" && <span className="sr-only">{", a step failed"}</span>}
        </span>
        <span className="flex h-5 shrink-0 items-center">
          <ChevronRight className="size-3.5 transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90" aria-hidden />
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        {quiet && <TurnStepRail entries={entries} live={undefined} root={root} entrance={false} />}
        {!quiet && (entries.length > 0 || running) && (
          <div className="mt-1 flex flex-col border-t border-border pt-1">
            {running && feed.length === 0 && <TurnRunLine entries={[]} running root={root} />}
            {feed.map((segment, i) => (
              <TurnFeedItem key={segment.kind === "reply" ? `say-${segment.entry.at}-${i}` : `run-${i}`} segment={segment} root={root} />
            ))}
          </div>
        )}
      </CollapsibleContent>
    </Collapsible>
  );
};

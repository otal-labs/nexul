import { ChevronRight, Loader2, TriangleAlert } from "lucide-react";

import { groupSteps, isFailedStep, listedAct, liveStepLabel, stepAct, turnPhrase } from "@nexul/client-core/agentSteps";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

import { TurnStepIcon } from "@/components/play/TurnStepIcon";
import { TurnStepList } from "@/components/play/TurnStepList";
import { TurnStepRail } from "@/components/play/TurnStepRail";
import type { ActivityEntry } from "@/models/Trail";

interface TurnRunLineProps {
  // entries are the tools the Agent ran between two of its sentences.
  entries: ActivityEntry[];
  running: boolean;
  root: string | null;
}

const sentence = (phrase: string): string => `${phrase.charAt(0).toUpperCase()}${phrase.slice(1)}`;

// The tools between two of the Agent's sentences as one line: the step in flight while it runs, else what they did.
export const TurnRunLine = ({ entries, running, root }: TurnRunLineProps) => {
  const last = entries[entries.length - 1];
  const live = running && last?.kind === "tool_call" ? last : undefined;
  const acts = new Set(entries.map(stepAct));
  const only = acts.size === 1 ? entries[0] : undefined;
  const failed = entries.some(isFailedStep);
  const groups = groupSteps(entries);
  // A run of one kind of step opens straight to its names, since the line already says how many.
  const lone = groups.length === 1 && groups[0] ? listedAct(groups[0]) : null;
  return (
    <Collapsible className="px-1 py-0.5">
      <CollapsibleTrigger
        disabled={entries.length === 0}
        className="group flex max-w-full items-start gap-2 rounded-md text-left text-[13px] leading-5 text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground disabled:hover:text-muted-foreground"
      >
        <span className="flex h-5 shrink-0 items-center">
          {running && <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" aria-hidden />}
          {!running && failed && <TriangleAlert className="size-3.5 text-destructive" aria-hidden />}
          {!running && !failed && <TurnStepIcon act={only ? stepAct(only) : "tool"} tool={only?.tool} />}
        </span>
        <span className="min-w-0">
          {running && <span className="shimmer-text">{liveStepLabel(entries)}</span>}
          {!running && sentence(turnPhrase(entries))}
          {failed && <span className="sr-only">{", a step failed"}</span>}
        </span>
        {entries.length > 0 && (
          <span className="flex h-5 shrink-0 items-center">
            <ChevronRight className="size-3.5 transition-transform duration-150 ease-standard group-data-[state=open]:rotate-90" aria-hidden />
          </span>
        )}
      </CollapsibleTrigger>
      <CollapsibleContent>
        {lone !== null && <TurnStepList act={lone} entries={entries} live={live} root={root} bare />}
        {lone === null && <TurnStepRail entries={entries} live={live} root={root} entrance={running} />}
      </CollapsibleContent>
    </Collapsible>
  );
};

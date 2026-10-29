import { useState } from "react";

import { groupHeadline, isFailedStep, stepAct, stepPhrase } from "@nexul/client-core/agentSteps";

import { TrailStepDetail } from "@/components/play/TrailStepDetail";
import { TurnStepIcon } from "@/components/play/TurnStepIcon";
import { TurnStepName } from "@/components/play/TurnStepName";
import type { ActivityEntry } from "@/models/Trail";
import { cn } from "@/lib/utils";

export type ListedAct = "explore" | "edit" | "command";

interface TurnStepListProps {
  act: ListedAct;
  entries: ActivityEntry[];
  live: ActivityEntry | undefined;
  root: string | null;
}

// A search reads as its quoted pattern, a file as its name alone, a command as commandText reads it.
const nameOf = (entry: ActivityEntry, root: string | null): string => {
  const { verb, object } = stepPhrase(entry, false, root);
  if (stepAct(entry) === "explore" && verb.startsWith("Searched")) return `“${object}”`;
  return object;
};

// Neighbouring reads, searches, edits or commands as one line saying how many, over their names: files side by side,
// commands one to a line; a name opens its step's detail under them.
export const TurnStepList = ({ act, entries, live, root }: TurnStepListProps) => {
  const [openAt, setOpenAt] = useState<number | null>(null);
  const opened = openAt === null ? undefined : entries[openAt];
  const running = live !== undefined && entries.includes(live);
  return (
    <div className="text-[13px] leading-5">
      <p className="flex items-center gap-2 text-muted-foreground">
        <TurnStepIcon act={act} tool={undefined} />
        {groupHeadline(act, entries, running)}
      </p>
      <ul className={cn("pl-[22px] font-mono text-xs leading-5 text-foreground/80", act === "command" ? "flex flex-col" : "flex flex-wrap items-center gap-x-2.5")}>
        {entries.map((entry, i) => (
          <TurnStepName
            key={`${entry.call_id || entry.at}-${i}`}
            label={nameOf(entry, root)}
            live={entry === live}
            failed={isFailedStep(entry)}
            open={openAt === i}
            onToggle={entry.detail ? () => setOpenAt(openAt === i ? null : i) : undefined}
          />
        ))}
      </ul>
      {opened?.detail && (
        <div className="pl-[22px]">
          <TrailStepDetail detail={opened.detail} at={opened.at} />
        </div>
      )}
    </div>
  );
};

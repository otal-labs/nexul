import { useState } from "react";

import { editCount, exploreCount, isFailedStep, stepAct, stepPhrase } from "@nexul/client-core/agentSteps";

import { StepStatusMark } from "@/components/play/StepStatusMark";
import { TrailStepDetail } from "@/components/play/TrailStepDetail";
import { TurnFileChip } from "@/components/play/TurnFileChip";
import type { ActivityEntry } from "@/models/Trail";

interface TurnFilesGroupProps {
  act: "explore" | "edit";
  entries: ActivityEntry[];
  live: ActivityEntry | undefined;
}

const title = (act: TurnFilesGroupProps["act"], entries: ActivityEntry[], running: boolean): string => {
  if (act === "explore") return `${running ? "Exploring" : "Explored"} ${exploreCount(entries)}`;
  return `${running ? "Editing" : "Edited"} ${editCount(entries)}`;
};

// A search reads as its quoted pattern, a file as its name alone.
const chipLabel = (entry: ActivityEntry): string => {
  const { verb, object } = stepPhrase(entry, false);
  if (stepAct(entry) === "explore" && verb.startsWith("Searched")) return `“${object}”`;
  return object;
};

// Neighbouring reads, searches or edits as one line saying how many, over the names; a name opens its step's detail under them.
export const TurnFilesGroup = ({ act, entries, live }: TurnFilesGroupProps) => {
  const [openAt, setOpenAt] = useState<number | null>(null);
  const opened = openAt === null ? undefined : entries[openAt];
  const running = live !== undefined && entries.includes(live);
  const failed = entries.some(isFailedStep);
  return (
    <div className="text-sm text-foreground/80">
      <p className="flex items-center gap-1.5">
        {title(act, entries, running)}
        {failed && <span className="text-muted-foreground">·</span>}
        <StepStatusMark live={false} failed={failed} />
      </p>
      <ul className="mt-0.5 flex flex-wrap items-center gap-x-2.5 gap-y-0.5 font-mono text-xs text-muted-foreground">
        {entries.map((entry, i) => (
          <TurnFileChip
            key={`${entry.call_id || entry.at}-${i}`}
            label={chipLabel(entry)}
            live={entry === live}
            failed={isFailedStep(entry)}
            open={openAt === i}
            onToggle={entry.detail ? () => setOpenAt(openAt === i ? null : i) : undefined}
          />
        ))}
      </ul>
      {opened?.detail && <TrailStepDetail detail={opened.detail} at={opened.at} />}
    </div>
  );
};

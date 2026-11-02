import { groupSteps } from "@nexul/client-core/agentSteps";

import { TurnStepGroup } from "@/components/play/TurnStepGroup";
import type { ActivityEntry } from "@/models/Trail";

interface TurnStepRailProps {
  entries: ActivityEntry[];
  live: ActivityEntry | undefined;
  root: string | null;
  entrance: boolean;
}

// A run's steps hung off a hairline rule, neighbouring reads, edits and commands under one count.
export const TurnStepRail = ({ entries, live, root, entrance }: TurnStepRailProps) => (
  <ul className="mt-1 mb-1 ml-[7px] flex flex-col gap-1.5 border-l border-border py-0.5 pl-4">
    {groupSteps(entries).map((group, i) => (
      <TurnStepGroup key={`${group.act}-${i}`} group={group} live={live} root={root} entrance={entrance} />
    ))}
  </ul>
);

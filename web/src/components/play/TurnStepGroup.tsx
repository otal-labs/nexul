import type { StepGroup } from "@nexul/client-core/agentSteps";

import { TurnCommandStep } from "@/components/play/TurnCommandStep";
import { TurnFilesGroup } from "@/components/play/TurnFilesGroup";
import { TurnStepLine } from "@/components/play/TurnStepLine";
import { cn } from "@/lib/utils";
import type { ActivityEntry } from "@/models/Trail";

interface TurnStepGroupProps {
  group: StepGroup<ActivityEntry>;
  // live is the unanswered call of a running turn, the one step that spins.
  live: ActivityEntry | undefined;
  // entrance settles the group in as it arrives; off when a finished turn is opened.
  entrance: boolean;
}

export const TurnStepGroup = ({ group, live, entrance }: TurnStepGroupProps) => {
  const first = group.entries[0];
  return (
    <li className={cn("list-none", entrance && "settle-in")}>
      {(group.act === "explore" || group.act === "edit") && <TurnFilesGroup act={group.act} entries={group.entries} live={live} />}
      {group.act === "command" && first && <TurnCommandStep entry={first} live={first === live} />}
      {(group.act === "tool" || group.act === "say") && first && <TurnStepLine entry={first} live={first === live} />}
    </li>
  );
};

import type { StepGroup } from "@nexul/client-core/agentSteps";

import { TurnStepLine } from "@/components/play/TurnStepLine";
import { TurnStepList, type ListedAct } from "@/components/play/TurnStepList";
import { cn } from "@/lib/utils";
import type { ActivityEntry } from "@/models/Trail";

interface TurnStepGroupProps {
  group: StepGroup<ActivityEntry>;
  // live is the unanswered call of a running turn, the one step that spins.
  live: ActivityEntry | undefined;
  // root is the run's directory, cut from the front of its commands.
  root: string | null;
  // entrance settles the group in as it arrives; off when a finished turn is opened.
  entrance: boolean;
}

// Files always list under a count; a command does once a neighbour joins it, and stands as one row alone.
const listedAct = (group: StepGroup<ActivityEntry>): ListedAct | null => {
  if (group.act === "explore" || group.act === "edit") return group.act;
  if (group.act === "command" && group.entries.length > 1) return "command";
  return null;
};

export const TurnStepGroup = ({ group, live, root, entrance }: TurnStepGroupProps) => {
  const first = group.entries[0];
  const listed = listedAct(group);
  return (
    <li className={cn("list-none", entrance && "settle-in")}>
      {listed !== null && <TurnStepList act={listed} entries={group.entries} live={live} root={root} />}
      {listed === null && first && <TurnStepLine entry={first} live={first === live} root={root} />}
    </li>
  );
};

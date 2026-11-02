import { listedAct, type StepGroup } from "@nexul/client-core/agentSteps";

import { TurnStepLine } from "@/components/play/TurnStepLine";
import { TurnStepList } from "@/components/play/TurnStepList";
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

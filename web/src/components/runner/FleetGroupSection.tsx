import { useState } from "react";

import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { FleetCodeRow } from "@/components/runner/FleetCodeRow";
import { FleetRunnerRow } from "@/components/runner/FleetRunnerRow";
import { GroupToggle } from "@/components/runner/GroupToggle";
import { MachineBand } from "@/components/runner/MachineBand";
import { NoMachineBand } from "@/components/runner/NoMachineBand";
import { groupState, type FleetGroup } from "@/utils/RunnerFleetUtility";

interface FleetGroupSectionProps {
  group: FleetGroup;
}

// One machine's band over its runners and unused codes, folded at first when every runner is offline; the last group, with no machine, holds what none claims.
export const FleetGroupSection = ({ group }: FleetGroupSectionProps) => {
  const { machine, runners, codes } = group;
  const [open, setOpen] = useState(() => !(runners.length > 0 && runners.every((r) => !r.connected)));
  const rows = runners.length + codes.length;
  const id = machine ? `machine-${machine.id}` : "machine-none";
  const toggle = rows > 0 && (
    <GroupToggle open={open} rows={rows} name={machine?.name ?? "No machine"} controls={`${id}-rows`} onToggle={() => setOpen(!open)} />
  );
  return (
    <section aria-labelledby={id} className="group/fleet">
      {machine && <MachineBand machine={machine} runners={runners} codes={codes.length} toggle={toggle} />}
      {!machine && <NoMachineBand state={groupState(runners, codes.length)} toggle={toggle} />}
      <ul id={`${id}-rows`} hidden={!open} className="divide-y divide-border">
        {runners.map((runner) => (
          <FleetRunnerRow key={runner.id} runner={runner} machine={machine} />
        ))}
        {codes.map((code) => (
          <FleetCodeRow key={code.id} code={code} />
        ))}
        {machine && rows === 0 && (
          <li className="flex items-baseline gap-2 py-2.5 pr-4 pl-9 text-[13px] leading-5 text-muted-foreground">
            No runners yet
            <AddRunnerDialog machine={machine} triggerVariant="link" />
          </li>
        )}
      </ul>
    </section>
  );
};

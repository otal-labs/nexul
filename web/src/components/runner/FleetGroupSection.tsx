import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { FleetCodeRow } from "@/components/runner/FleetCodeRow";
import { FleetRunnerRow } from "@/components/runner/FleetRunnerRow";
import { MachineBand } from "@/components/runner/MachineBand";
import type { FleetGroup } from "@/utils/RunnerFleetUtility";

interface FleetGroupSectionProps {
  group: FleetGroup;
}

// One machine's band over its runners and unused codes; the last group, with no machine, holds what none claims.
export const FleetGroupSection = ({ group }: FleetGroupSectionProps) => {
  const { machine, runners, codes } = group;
  return (
    <section aria-labelledby={machine ? `machine-${machine.id}` : "machine-none"}>
      {machine && <MachineBand machine={machine} runners={runners} />}
      {!machine && (
        <div className="flex items-baseline gap-3 border-b border-border bg-well-shade px-4 py-2.5">
          <span aria-hidden className="w-2 shrink-0" />
          <h2 id="machine-none" className="text-sm/5 font-semibold">
            No machine
          </h2>
          <span className="-ml-0.5 text-xs text-muted-foreground">not tied to a machine</span>
        </div>
      )}
      <ul className="divide-y divide-border">
        {runners.map((runner) => (
          <FleetRunnerRow key={runner.id} runner={runner} machine={machine} />
        ))}
        {codes.map((code) => (
          <FleetCodeRow key={code.id} code={code} />
        ))}
        {machine && runners.length + codes.length === 0 && (
          <li className="flex items-baseline gap-2 py-2.5 pr-4 pl-9 text-[13px] leading-5 text-muted-foreground">
            No runners yet
            <AddRunnerDialog machine={machine} triggerVariant="link" />
          </li>
        )}
      </ul>
    </section>
  );
};

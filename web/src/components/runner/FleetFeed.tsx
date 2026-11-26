import { FleetGroupSection } from "@/components/runner/FleetGroupSection";
import type { FleetGroup } from "@/utils/RunnerFleetUtility";

interface FleetFeedProps {
  groups: FleetGroup[];
}

// Every machine in one list, so the runner rows share their columns from the first machine to the last; the values name themselves, so no column head.
export const FleetFeed = ({ groups }: FleetFeedProps) => (
  <div className="@container divide-y divide-border overflow-clip rounded-lg border border-border bg-card">
    {groups.map((group) => (
      <FleetGroupSection key={group.machine?.id ?? "none"} group={group} />
    ))}
  </div>
);

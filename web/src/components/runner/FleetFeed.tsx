import { FleetGroupSection } from "@/components/runner/FleetGroupSection";
import { fleetTracks } from "@/components/runner/fleetTracks";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import type { FleetGroup } from "@/utils/RunnerFleetUtility";

interface FleetFeedProps {
  groups: FleetGroup[];
}

// Every machine in one list, so the runner rows share their columns from the first machine to the last.
export const FleetFeed = ({ groups }: FleetFeedProps) => (
  <div className="@container divide-y divide-border overflow-hidden rounded-lg border border-border bg-card">
    <div className={cn(fleetTracks, "h-8 items-center gap-x-4 px-4", microheaderClass)} aria-hidden>
      <span className="pl-5">Runner</span>
      <span>Version</span>
      <span>Doing</span>
      <span />
    </div>
    {groups.map((group) => (
      <FleetGroupSection key={group.machine?.id ?? "none"} group={group} />
    ))}
  </div>
);

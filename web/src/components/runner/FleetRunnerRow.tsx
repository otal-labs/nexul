import { RunnerActivity } from "@/components/runner/RunnerActivity";
import { RunnerMenu } from "@/components/runner/RunnerMenu";
import { RunnerName } from "@/components/runner/RunnerName";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import { RunnerVersion } from "@/components/runner/RunnerVersion";
import { fleetTracks } from "@/components/runner/fleetTracks";
import type { Machine, Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";

interface FleetRunnerRowProps {
  runner: Runner;
  machine?: Machine | undefined;
}

// One line on the list's shared tracks; a name that only repeats its machine's reads "Runner".
export const FleetRunnerRow = ({ runner, machine }: FleetRunnerRowProps) => (
  <li className={cn(fleetTracks, "items-baseline gap-x-4 px-4 py-2.5 leading-5 transition-colors duration-150 ease-standard hover:bg-accent/40")}>
    <span className="flex min-w-0 items-baseline gap-3">
      <span className="flex self-center">
        <RunnerStatusDot connected={runner.connected} />
      </span>
      <RunnerName name={runner.name && runner.name !== machine?.name ? runner.name : "Runner"} muted={!runner.connected} />
    </span>
    <RunnerVersion version={runner.version} />
    <RunnerActivity runner={runner} />
    <RunnerMenu runner={runner} machine={machine} />
  </li>
);

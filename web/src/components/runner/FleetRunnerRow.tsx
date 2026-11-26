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

// One line on the list's shared tracks, stepped in under its band so its dot sits under the machine's name.
export const FleetRunnerRow = ({ runner, machine }: FleetRunnerRowProps) => (
  <li className={cn(fleetTracks, "items-baseline gap-x-4 py-2.5 pr-4 pl-11 leading-5 transition-colors duration-150 ease-standard hover:bg-accent")}>
    <span className="flex min-w-0 items-baseline gap-3">
      <span className="flex self-center">
        <RunnerStatusDot connected={runner.connected} />
      </span>
      <RunnerName name={runner.name} muted={!runner.connected} />
    </span>
    <RunnerActivity runner={runner} />
    <span className="flex min-w-0 justify-end">
      <RunnerVersion version={runner.version} />
    </span>
    <RunnerMenu runner={runner} machine={machine} />
  </li>
);

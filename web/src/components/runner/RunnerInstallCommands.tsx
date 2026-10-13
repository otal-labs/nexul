import { RunnerConnectedCheck } from "@/components/runner/RunnerConnectedCheck";
import { RunnerInstallCommand } from "@/components/runner/RunnerInstallCommand";
import type { RunnerEnrollment } from "@/models/Runner";

interface RunnerInstallCommandsProps {
  enrollment: RunnerEnrollment;
  // The name the person typed; the runner that connects under it ticks the check.
  name: string;
}

export const RunnerInstallCommands = ({ enrollment, name }: RunnerInstallCommandsProps) => (
  <div className="space-y-4">
    <RunnerInstallCommand label="Linux / macOS" command={enrollment.commands.unix} />
    <RunnerInstallCommand label="Windows" command={enrollment.commands.windows} />
    <p className="text-xs text-muted-foreground">
      Run one on the machine to install the runner as a service. Its enrollment code works once, within the hour.
    </p>
    <RunnerConnectedCheck name={name} />
  </div>
);

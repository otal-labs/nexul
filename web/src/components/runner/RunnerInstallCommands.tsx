import { RunnerInstallCommand } from "@/components/runner/RunnerInstallCommand";
import type { RunnerEnrollment } from "@/models/Runner";
import { RunnerPlatform, withGitToken } from "@/utils/RunnerInstallCommand";

interface RunnerInstallCommandsProps {
  enrollment: RunnerEnrollment;
  gitToken: string;
}

export const RunnerInstallCommands = ({ enrollment, gitToken }: RunnerInstallCommandsProps) => (
  <div className="space-y-4">
    <RunnerInstallCommand
      label="Linux / macOS"
      command={withGitToken(enrollment.commands.unix, RunnerPlatform.Unix, gitToken)}
    />
    <RunnerInstallCommand
      label="Windows"
      command={withGitToken(enrollment.commands.windows, RunnerPlatform.Windows, gitToken)}
    />
    <p className="text-xs text-muted-foreground">
      Run one of these on the machine; it installs the runner as a service. The code in it works once and
      expires within the hour.
    </p>
  </div>
);

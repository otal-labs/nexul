import { RunnerInstallCommand } from "@/components/runner/RunnerInstallCommand";
import type { AutomationHostEnrollment } from "@/models/AutomationHost";

interface AutomationHostInstallCommandsProps {
  enrollment: AutomationHostEnrollment;
}

export const AutomationHostInstallCommands = ({ enrollment }: AutomationHostInstallCommandsProps) => (
  <div className="space-y-4">
    <RunnerInstallCommand label="Linux / macOS" command={enrollment.commands.unix} />
    <RunnerInstallCommand label="Windows" command={enrollment.commands.windows} />
    <p className="text-xs text-muted-foreground">
      Run one of these on the machine; it installs the automations host as a service. The code in it works once
      and expires within the hour.
    </p>
  </div>
);

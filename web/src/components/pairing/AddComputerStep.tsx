import { CommandSnippet } from "@/components/pairing/CommandSnippet";
import { ComputerChecks } from "@/components/pairing/ComputerChecks";
import { ConnectionPanel } from "@/components/pairing/ConnectionPanel";
import type { Computer, ComputerEnrollment } from "@/models/Pairing";
import { formatClockTime } from "@/utils/TimeUtility";
import { computerCommands, PAIRING_GUIDE } from "@/utils/TunnelInstallCommands";

interface AddComputerStepProps {
  enrollment: ComputerEnrollment;
  // The computer as the list last read it, which every pushed frame refreshes.
  computer: Computer;
}

// The command to run on the computer beside the three checks it turns green, in the order they happen.
export const AddComputerStep = ({ enrollment, computer }: AddComputerStepProps) => (
  <div className="@container">
    <div className="grid grid-cols-1 gap-5 @2xl:grid-cols-[minmax(0,1fr)_18rem]">
      <div className="min-w-0 space-y-3">
        <p className="text-sm text-muted-foreground">
          Run this in a terminal on the computer you're adding, from your own account. It asks for your password once, for{" "}
          <code className="font-mono text-xs text-foreground">sudo</code>, to install a background service, and installs everything for the
          account that typed <code className="font-mono text-xs text-foreground">sudo</code>, never for root.
        </p>
        <CommandSnippet commands={computerCommands(enrollment.commands)} label="Install command" guide={PAIRING_GUIDE} />
        <p className="text-xs text-muted-foreground">
          It works once, until <span className="font-mono tabular-nums">{formatClockTime(enrollment.expires_at)}</span>. T3 Code is installed
          too when the computer has none.
        </p>
      </div>
      <div className="min-w-0 space-y-3">
        <ConnectionPanel connected={computer.runner?.connected === true} hostname={computer.facts?.hostname ?? ""} />
        <ComputerChecks computer={computer} />
      </div>
    </div>
  </div>
);

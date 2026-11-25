import { ComputerCommands } from "@/components/settings/ComputerCommands";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { runnerPresence } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";
import { ComputerPermission, sharedComputerAbilities, sharedComputerName } from "@/models/SharedComputer";

interface SharedComputerRowProps {
  computer: Computer;
}

// Someone else's computer shared with the viewer: its name and whether it is online, nothing of its owner's setup, and
// with Run commands the commands they ran there.
export const SharedComputerRow = ({ computer }: SharedComputerRowProps) => {
  const presence = runnerPresence(computer.runner);
  return (
    <li className="py-3">
      <p className="line-clamp-2 text-sm font-medium break-words">{sharedComputerName(computer)}</p>
      <p className="mt-1 flex min-w-0 items-center">
        <SettingsStatus tone={presence.tone} detail={computer.shared && sharedComputerAbilities(computer.shared)}>
          {presence.text}
        </SettingsStatus>
      </p>
      {computer.shared?.permissions.includes(ComputerPermission.RunCommands) && (
        <dl className="mt-2 rounded-md bg-surface-2 p-3">
          <ComputerCommands computer={computer} />
        </dl>
      )}
    </li>
  );
};

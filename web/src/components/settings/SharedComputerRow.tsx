import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { Computer } from "@/models/Pairing";
import { sharedComputerAbilities, sharedComputerName } from "@/models/SharedComputer";

interface SharedComputerRowProps {
  computer: Computer;
}

// Someone else's computer shared with the viewer: its name and whether it is online, nothing of its owner's setup.
export const SharedComputerRow = ({ computer }: SharedComputerRowProps) => {
  const online = computer.runner?.connected === true;
  return (
    <li className="bg-card px-3 py-3">
      <p className="line-clamp-2 text-sm font-medium break-words">{sharedComputerName(computer)}</p>
      <p className="mt-1 flex min-w-0 items-center">
        <SettingsStatus tone={online ? "success" : "muted"} detail={computer.shared && sharedComputerAbilities(computer.shared)}>
          {online ? "Online" : "Offline"}
        </SettingsStatus>
      </p>
    </li>
  );
};

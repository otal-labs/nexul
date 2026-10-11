import { Folder } from "lucide-react";

import { DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { useFetchHarnessProjects } from "@/hooks/PairingHooks";
import type { Computer } from "@/models/Pairing";

interface PickComputerGroupProps {
  computer: Computer;
  connected: boolean;
  separated: boolean;
  onPick: (computerId: string, harnessProjectId: string) => void;
}

// One computer under its name, its T3 projects as the rows: a run needs both, and an offline computer can list neither.
export const PickComputerGroup = ({ computer, connected, separated, onPick }: PickComputerGroupProps) => {
  const { data: projects, isPending, error } = useFetchHarnessProjects(connected ? computer.id : "");

  return (
    <DropdownMenuGroup>
      {separated && <DropdownMenuSeparator />}
      <DropdownMenuLabel className="truncate">{computer.name}</DropdownMenuLabel>
      {!connected && <DropdownMenuItem disabled>Offline</DropdownMenuItem>}
      {connected && isPending && <DropdownMenuItem disabled>Loading T3 projects</DropdownMenuItem>}
      {connected && error && <DropdownMenuItem disabled>Couldn't list its T3 projects</DropdownMenuItem>}
      {projects && projects.length === 0 && <DropdownMenuItem disabled>No T3 projects open</DropdownMenuItem>}
      {projects?.map((project) => (
        <DropdownMenuItem key={project.id} onSelect={() => onPick(computer.id, project.id)}>
          <Folder aria-hidden />
          <span className="min-w-0 flex-1 truncate">{project.title}</span>
        </DropdownMenuItem>
      ))}
    </DropdownMenuGroup>
  );
};

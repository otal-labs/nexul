import { DownloadIcon, MoreHorizontalIcon, PlusIcon, TerminalIcon } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { RowActionItems } from "@/components/listpane/RowActionItems";
import { dropdownParts, offersMenu, type RowMenu } from "@/components/listpane/rowMenu";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { MachineCommandsSheet } from "@/components/runner/MachineCommandsSheet";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useActionState } from "@/hooks/useActionState";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Machine } from "@/models/Runner";

interface MachineMenuProps {
  machine: Machine;
  menu: RowMenu;
  manages: boolean;
  // Left out while the machine has no rows, whose empty row offers it.
  addsRunners: boolean;
}

// Add runner…, Import and Commands… first, then the menu every row shares: Permissions… and Delete. A viewer offered
// none of them gets no ⋯.
export const MachineMenu = ({ machine, menu, manages, addsRunners }: MachineMenuProps) => {
  const wsPath = useWorkspacePath();
  const navigate = useNavigate();
  const [commandsOpen, setCommandsOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const addGate = { permission: "machines:write", answer: machineAnswer(machine, "machines:write"), verb: "add runners" } as const;
  const importGate = {
    permission: ["machines:write", "stacks:write"],
    answer: machineAnswer(machine, "machines:write"),
    verb: "import from machines",
  } as const;
  const commandsGate = {
    permission: "machines:shell",
    answer: machineAnswer(machine, "machines:shell"),
    verb: "run commands on this machine",
  } as const;
  const imports = useActionState(importGate).state !== "hidden";
  const commands = useActionState(commandsGate).state !== "hidden";
  const adds = useActionState(addGate).state !== "hidden" && addsRunners;
  if (!adds && !imports && !commands && !offersMenu(menu, manages)) return null;
  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${machine.name}`}>
            <MoreHorizontalIcon className="size-4" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-48">
          {addsRunners && (
            <ActionMenuItem ui={dropdownParts} {...addGate} icon={PlusIcon} onSelect={() => setAddOpen(true)}>
              Add runner…
            </ActionMenuItem>
          )}
          <ActionMenuItem
            ui={dropdownParts}
            {...importGate}
            icon={DownloadIcon}
            onSelect={() => void navigate(wsPath(`/wizard/project/import?machine=${machine.id}`))}
          >
            Import from this machine
          </ActionMenuItem>
          <ActionMenuItem ui={dropdownParts} {...commandsGate} icon={TerminalIcon} onSelect={() => setCommandsOpen(true)}>
            Commands…
          </ActionMenuItem>
          {(adds || imports || commands) && offersMenu(menu, manages) && <DropdownMenuSeparator />}
          {offersMenu(menu, manages) && <RowActionItems ui={dropdownParts} {...menu} />}
        </DropdownMenuContent>
      </DropdownMenu>
      <MachineCommandsSheet machine={machine} open={commandsOpen} onOpenChange={setCommandsOpen} />
      <AddRunnerDialog machine={machine} open={addOpen} onOpenChange={setAddOpen} />
    </>
  );
};

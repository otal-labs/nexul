import { DownloadIcon, MoreHorizontalIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { RowActionItems } from "@/components/listpane/RowActionItems";
import { dropdownParts, offersMenu, type RowMenu } from "@/components/listpane/rowMenu";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useActionState } from "@/hooks/useActionState";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Machine } from "@/models/Runner";

interface MachineMenuProps {
  machine: Machine;
  menu: RowMenu;
  manages: boolean;
}

// Import first, then the menu every row shares: Permissions… and Delete. A viewer offered none of them gets no ⋯.
export const MachineMenu = ({ machine, menu, manages }: MachineMenuProps) => {
  const wsPath = useWorkspacePath();
  const navigate = useNavigate();
  const importGate = { permission: ["machines:write", "stacks:write"], answer: machineAnswer(machine, "machines:write"), verb: "import from machines" } as const;
  const imports = useActionState(importGate).state !== "hidden";
  if (!imports && !offersMenu(menu, manages)) return null;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${machine.name}`}>
          <MoreHorizontalIcon className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-48">
        <ActionMenuItem
          ui={dropdownParts}
          {...importGate}
          icon={DownloadIcon}
          onSelect={() => void navigate(wsPath(`/wizard/project/import?machine=${machine.id}`))}
        >
          Import from this machine
        </ActionMenuItem>
        {imports && offersMenu(menu, manages) && <DropdownMenuSeparator />}
        {offersMenu(menu, manages) && <RowActionItems ui={dropdownParts} {...menu} />}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

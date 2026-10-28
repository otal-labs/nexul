import { useState } from "react";
import { ChevronRight } from "lucide-react";

import { ComputerDetails } from "@/components/settings/ComputerDetails";
import { useComputerOrder } from "@/components/settings/ComputerOrderContext";
import { MoveNotice } from "@/components/settings/MoveNotice";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { useFetchComputerSetup } from "@/hooks/ComputerSetupHooks";
import { useDeleteComputer } from "@/hooks/PairingHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Computer } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface ComputerRowProps {
  computer: Computer;
  onMove: () => void;
}

// A computer paired before the Nexul app: its name, what to do, and its kept details under the fold.
export const ComputerRow = ({ computer, onMove }: ComputerRowProps) => {
  const { data: setup } = useFetchComputerSetup(computer.id);
  const remove = useDeleteComputer();
  const { open: confirm } = useConfirmationDialog();
  const [folded, setFolded] = useState(true);
  const { grip, moveActions } = useComputerOrder();

  const removeComputer = async () => {
    const ok = await confirm({
      title: `Remove ${computer.name}?`,
      message: "Its project links and setup go with it. Add it again with the Nexul app to bring it back.",
      confirmLabel: "Remove computer",
    });
    if (ok) remove.mutate(computer);
  };
  const actions: RowAction[] = [...moveActions, { label: "Remove", destructive: true, onSelect: () => void removeComputer() }];

  return (
    <div className="py-3">
      <div className="flex items-center gap-2">
        {grip}
        <button
          type="button"
          aria-expanded={!folded}
          onClick={() => setFolded(!folded)}
          className="group -ml-1 flex min-w-0 flex-1 items-center gap-1.5 rounded-md px-1 py-0.5 text-left focus-visible:outline-2 focus-visible:outline-focus"
        >
          <ChevronRight
            aria-hidden
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground",
              !folded && "rotate-90",
            )}
          />
          <span className="line-clamp-2 text-sm font-medium break-words" title={computer.name}>
            {computer.name}
          </span>
        </button>
        <RowActionsMenu subject={computer.name} actions={actions} />
      </div>
      <MoveNotice computer={computer} onMove={onMove} className={grip ? "pl-18" : "pl-6"} />
      <div inert={folded} aria-hidden={folded} data-closed={folded || undefined} className="disclosure">
        <div>
          <div className={cn("pt-3", grip ? "pl-18" : "pl-6")}>
            <ComputerDetails computer={computer} setup={setup} />
          </div>
        </div>
      </div>
    </div>
  );
};

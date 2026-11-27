import { PaneHeader } from "@/components/computers/PaneHeader";
import { ComputerDetails } from "@/components/settings/ComputerDetails";
import { MoveNotice } from "@/components/settings/MoveNotice";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { useFetchComputerSetup } from "@/hooks/ComputerSetupHooks";
import { useDeleteComputer } from "@/hooks/PairingHooks";
import { useComputerMoves } from "@/hooks/useComputerMoves";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Computer } from "@/models/Pairing";

interface LegacyComputerPaneProps {
  computer: Computer;
  onMove: () => void;
}

// A computer paired before the Nexul app: its name, what to do, and its kept details.
export const LegacyComputerPane = ({ computer, onMove }: LegacyComputerPaneProps) => {
  const { data: setup } = useFetchComputerSetup(computer.id);
  const remove = useDeleteComputer();
  const { open: confirm } = useConfirmationDialog();
  const moveActions = useComputerMoves(computer.id);

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
    <div className="space-y-6">
      <PaneHeader title={computer.name} meta={<MoveNotice computer={computer} onMove={onMove} />} actions={<RowActionsMenu subject={computer.name} actions={actions} />} />
      <ComputerDetails computer={computer} setup={setup} />
    </div>
  );
};

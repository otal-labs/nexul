import { DangerAction, DangerButton, DangerZone } from "@/components/settings/DangerZone";
import { useDeleteAutomation } from "@/hooks/AutomationHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";

interface AutomationDeleteButtonProps {
  automationId: string;
  onDeleted: () => void;
}

export const AutomationDeleteButton = ({ automationId, onDeleted }: AutomationDeleteButtonProps) => {
  const deleteAutomation = useDeleteAutomation();
  const { open: confirm } = useConfirmationDialog();

  const onClick = async () => {
    const ok = await confirm({
      title: "Delete this automation?",
      message: "Its config, token, runs, and versions go with it. This can't be undone.",
      confirmLabel: "Delete automation",
      destructive: true,
    });
    if (!ok) return;
    await deleteAutomation.mutateAsync(automationId);
    onDeleted();
  };

  return (
    <DangerZone>
      <DangerAction
        title="Delete automation"
        consequence="Removes its config, token, runs, and versions for good. Its token stops working at once."
        action={
          <DangerButton onClick={() => void onClick()} loading={deleteAutomation.isPending}>
            Delete automation
          </DangerButton>
        }
      />
    </DangerZone>
  );
};

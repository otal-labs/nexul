import { ActionButton } from "@/components/action/ActionButton";
import { DangerAction, dangerButtonClass, DangerZone } from "@/components/settings/DangerZone";
import { useDeleteAutomation } from "@/hooks/AutomationHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";

interface AutomationDeleteButtonProps {
  automationId: string;
  /** A shipped default comes back at the next start, so it is switched off, not deleted. */
  shipped?: boolean;
  onDeleted: () => void;
}

export const AutomationDeleteButton = ({ automationId, shipped = false, onDeleted }: AutomationDeleteButtonProps) => {
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
        consequence={shipped ? "Shipped with Nexul, so it comes back at the next start. Switch it off instead." : "Removes its config, token, runs, and versions for good. Its token stops working at once."}
        action={
          <ActionButton
            permission="automations:delete"
            variant="outline"
            size="sm"
            liveClassName={shipped ? "" : dangerButtonClass}
            disabled={shipped}
            onClick={() => void onClick()}
            loading={deleteAutomation.isPending}
          >
            Delete automation
          </ActionButton>
        }
      />
    </DangerZone>
  );
};

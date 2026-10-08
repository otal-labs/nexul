import { Trash2, TriangleAlert } from "lucide-react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
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
      destructive: true,
    });
    if (!ok) return;
    await deleteAutomation.mutateAsync(automationId);
    onDeleted();
  };

  return (
    <SettingsCard
      id="danger-zone"
      title="Danger zone"
      danger
      icon={TriangleAlert}
      description="Deleting the automation removes its config, token, runs, and versions for good."
    >
      <Button type="button" variant="destructive" size="sm" onClick={onClick} loading={deleteAutomation.isPending}>
        <Trash2 className="size-4" />
        Delete automation
      </Button>
    </SettingsCard>
  );
};

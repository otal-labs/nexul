import { Trash2 } from "lucide-react";

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
      message: "This permanently removes its config, token, run history, and version history.",
      destructive: true,
    });
    if (!ok) return;
    await deleteAutomation.mutateAsync(automationId);
    onDeleted();
  };

  return (
    <div className="rounded-lg border border-destructive/30 p-4">
      <Button type="button" variant="destructive" size="sm" onClick={onClick} disabled={deleteAutomation.isPending}>
        <Trash2 className="size-4" />
        Delete automation
      </Button>
    </div>
  );
};

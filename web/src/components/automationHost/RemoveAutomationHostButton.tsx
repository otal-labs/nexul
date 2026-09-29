import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useRemoveAutomationHost } from "@/hooks/AutomationHostHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { AutomationHost } from "@/models/AutomationHost";

interface RemoveAutomationHostButtonProps {
  host: AutomationHost;
}

export const RemoveAutomationHostButton = ({ host }: RemoveAutomationHostButtonProps) => {
  const remove = useRemoveAutomationHost();
  const { open: confirm } = useConfirmationDialog();

  const onRemove = async () => {
    const effect = host.connected
      ? "It uninstalls itself within seconds, and a run in progress stops."
      : "It uninstalls itself the next time it comes online.";
    const ok = await confirm({
      title: `Remove ${host.name}?`,
      message: `${effect} Its automations move back to the instance host. Its credential stops working for good; to bring it back, add it again.`,
      confirmLabel: "Remove automations host",
    });
    if (ok) remove.mutate(host.id);
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-label={`Remove ${host.name}`}
      title={`Remove ${host.name}`}
      className="shrink-0 hover:text-destructive"
      loading={remove.isPending}
      onClick={() => void onRemove()}
    >
      <Trash2 className="size-4" />
    </Button>
  );
};

import { Trash2 } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
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
      message: `${effect} Its automations move to the instance host and its credential stops working. A host added again is a new one, and the automations stay on the instance host until you move them back.`,
      confirmLabel: "Remove automations host",
    });
    if (ok) remove.mutate(host.id);
  };

  return (
    <ActionButton
      permission="machines:write"
      verb="remove automations hosts"
      icon={Trash2}
      type="button"
      variant="ghost"
      size="sm"
      aria-label={`Remove ${host.name}`}
      title={`Remove ${host.name}`}
      className="shrink-0"
      liveClassName="hover:text-destructive"
      loading={remove.isPending}
      onClick={() => void onRemove()}
    />
  );
};

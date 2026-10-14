import { Trash2 } from "lucide-react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { useRemoveAutomationHost } from "@/hooks/AutomationHostHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { AutomationHost } from "@/models/AutomationHost";
import type { Machine } from "@/models/Runner";

interface RemoveAutomationHostButtonProps {
  host: AutomationHost;
  // Its machine's card says what the viewer holds there; a host on a machine Nexul doesn't know answers to the instance.
  machine?: Pick<Machine, "actions"> | undefined;
}

export const RemoveAutomationHostButton = ({ host, machine }: RemoveAutomationHostButtonProps) => {
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
      answer={machine && machineAnswer(machine, "machines:write")}
      verb="remove automations hosts"
      type="button"
      variant="ghost"
      size="icon"
      icon={Trash2}
      aria-label={`Remove ${host.name}`}
      title={`Remove ${host.name}`}
      className="size-8 shrink-0"
      liveClassName="hover:text-destructive"
      loading={remove.isPending}
      onClick={() => void onRemove()}
    />
  );
};

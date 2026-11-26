import { MoreHorizontalIcon, Trash2 } from "lucide-react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { dropdownParts } from "@/components/listpane/rowMenu";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useRemoveAutomationHost } from "@/hooks/AutomationHostHooks";
import { useActionState } from "@/hooks/useActionState";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { AutomationHost } from "@/models/AutomationHost";
import type { Machine } from "@/models/Runner";

interface AutomationHostMenuProps {
  host: AutomationHost;
  // Its machine's card says what the viewer holds there; a host on a machine Nexul doesn't know answers to the instance.
  machine?: Pick<Machine, "actions"> | undefined;
}

// Holds Remove alone, so a viewer who isn't offered it gets an empty cell instead of a ⋯ that opens on nothing.
export const AutomationHostMenu = ({ host, machine }: AutomationHostMenuProps) => {
  const gate = { permission: "machines:write", answer: machine && machineAnswer(machine, "machines:write"), verb: "remove automations hosts" } as const;
  const offered = useActionState(gate).state !== "hidden";
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

  if (!offered) return <span />;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="-my-1 size-7" aria-label={`More actions for ${host.name}`} loading={remove.isPending}>
          <MoreHorizontalIcon className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-40">
        <ActionMenuItem ui={dropdownParts} {...gate} icon={Trash2} destructive onSelect={() => void onRemove()}>
          Remove host
        </ActionMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

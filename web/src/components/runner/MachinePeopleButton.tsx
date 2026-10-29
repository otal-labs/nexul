import { UsersIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useIsInstanceOwner } from "@/hooks/AccessHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { grantKey, machineEntity } from "@/models/PermissionPanel";
import type { Machine } from "@/models/Runner";
import { cn } from "@/lib/utils";

interface MachinePeopleButtonProps {
  machine: Pick<Machine, "id" | "name">;
  className?: string;
}

// Who a machine's rules reach, in words, for whoever manages them; it opens the permissions panel on the machine.
export const MachinePeopleButton = ({ machine, className }: MachinePeopleButtonProps) => {
  const owner = useIsInstanceOwner();
  const { data: grants } = useFetchGrants("machine", machine.id);
  const { targetOf } = usePanelTargets({ workspaceId: "", instance: true, rulesOn: { type: "machine", id: machine.id } });
  const openPanel = usePermissionsDialog();
  const names = grants?.filter((g) => g.allow.length + g.deny.length > 0).map((g) => targetOf(grantKey(g)).name);
  const more = names && names.length > 2 ? ` +${names.length - 2}` : "";
  const alone = owner ? "Only you" : "Only the instance Owner";
  const label = names && names.length > 0 ? `${names.slice(0, 2).join(", ")}${more}` : alone;

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      onClick={() => void openPanel(machineEntity(machine))}
      title="People on this machine"
      aria-label={`People on this machine: ${label}`}
      className={cn("max-w-56 font-normal text-muted-foreground", className)}
    >
      <UsersIcon aria-hidden />
      <span className="truncate">{names ? label : ""}</span>
    </Button>
  );
};

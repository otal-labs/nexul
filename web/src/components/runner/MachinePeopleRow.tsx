import { ChevronRightIcon, PlusIcon, UsersIcon } from "lucide-react";

import { useIsInstanceOwner } from "@/hooks/AccessHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { grantKey, machineEntity } from "@/models/PermissionPanel";
import type { Machine } from "@/models/Runner";

interface MachinePeopleRowProps {
  machine: Pick<Machine, "id" | "name">;
}

// Who a machine's rules name, for whoever manages them; the whole row opens the permissions panel on the machine.
export const MachinePeopleRow = ({ machine }: MachinePeopleRowProps) => {
  const owner = useIsInstanceOwner();
  const { data: grants } = useFetchGrants("machine", machine.id);
  const { targetOf } = usePanelTargets({ workspaceId: "", instance: true });
  const openPanel = usePermissionsDialog();
  const names = grants?.filter((g) => g.allow.length + g.deny.length > 0).map((g) => targetOf(grantKey(g)).name);
  const alone = names?.length === 0;
  const more = names && names.length > 3 ? ` +${names.length - 3} more` : "";

  return (
    <button
      type="button"
      onClick={() => void openPanel(machineEntity(machine))}
      className="flex w-full items-center gap-3 border-t border-border px-4 py-2.5 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <UsersIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-medium">People on this machine</span>
        <span className="block min-h-4 truncate text-xs text-muted-foreground">
          {alone && (owner ? "Only you" : "Only the instance Owner")}
          {names && names.length > 0 && `${names.slice(0, 3).join(", ")}${more}`}
        </span>
      </span>
      {alone && (
        <span className="inline-flex shrink-0 items-center gap-1 text-xs font-medium">
          <PlusIcon className="size-3.5" aria-hidden />
          Add people
        </span>
      )}
      {!alone && <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />}
    </button>
  );
};

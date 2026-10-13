import { ChevronRightIcon, ShieldIcon } from "lucide-react";

import { useFetchGrants } from "@/hooks/PermissionHooks";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import type { InstanceArea } from "@/models/Permission";
import { areaEntity, areaName, grantKey } from "@/models/PermissionPanel";

interface InstanceAreaRowProps {
  area: InstanceArea;
}

// One instance area and who its rules name; the whole row opens the permissions panel on it.
export const InstanceAreaRow = ({ area }: InstanceAreaRowProps) => {
  const { data: grants } = useFetchGrants("instance_area", area.key);
  const { targetOf } = usePanelTargets({ workspaceId: "", instance: true });
  const openPanel = usePermissionsDialog();
  const names = grants?.filter((g) => g.allow.length + g.deny.length > 0).map((g) => targetOf(grantKey(g)).name);
  const more = names && names.length > 3 ? ` +${names.length - 3} more` : "";
  const line = names && (names.length === 0 ? "No rules, so each person's roles decide" : `Rules for ${names.slice(0, 3).join(", ")}${more}`);

  return (
    <li>
      <button
        type="button"
        onClick={() => void openPanel(areaEntity(area))}
        className="-mx-2 flex w-[calc(100%+1rem)] items-center gap-3 rounded-md px-2 py-2 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <span className="flex size-6 shrink-0 items-center justify-center rounded-md bg-surface-2 text-muted-foreground ring-1 ring-border ring-inset">
          <ShieldIcon className="size-[55%]" aria-hidden />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">{areaName(area)}</span>
          <span className="block min-h-4 truncate text-xs text-muted-foreground">{line}</span>
        </span>
        <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </button>
    </li>
  );
};

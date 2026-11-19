import { Lock } from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import { RoleReach } from "@/components/role/RoleReach";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import type { RoleChoice } from "@/models/RoleChoice";

interface RoleCheckRowProps {
  choice: RoleChoice;
  catalog: PermissionInfo[];
  checked: boolean;
  onToggle: (roleId: string) => void;
}

// The whole row is the box's label; a role you can't give keeps its tick, can't change, and says why.
export const RoleCheckRow = ({ choice, catalog, checked, onToggle }: RoleCheckRowProps) => {
  const { role, givable, reason } = choice;
  const id = `role-${role.id}`;
  return (
    <li className="flex min-h-11 items-start gap-3 py-2.5">
      {/* The box sits on the name's 20px line, not the middle of a row its description and reach make taller. */}
      <Checkbox id={id} checked={checked} disabled={!givable} onCheckedChange={() => onToggle(role.id)} className="mt-0.5" />
      <label htmlFor={id} className={cn("min-w-0 flex-1", givable && "cursor-pointer")}>
        <span className={cn("block truncate text-sm font-medium", !givable && "text-muted-foreground")}>{role.name}</span>
        {role.description && <span className="line-clamp-2 text-xs leading-4.5 text-muted-foreground" title={role.description}>{role.description}</span>}
        {givable && (
          <span className="mt-1 block">
            <RoleReach catalog={catalog} permissions={role.permissions} />
          </span>
        )}
        {!givable && (
          <span className="mt-0.5 flex items-start gap-1.5 text-xs text-muted-foreground">
            <Lock aria-hidden className="mt-px size-3 shrink-0" />
            {reason}
          </span>
        )}
      </label>
    </li>
  );
};

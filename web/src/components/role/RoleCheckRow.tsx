import { Lock } from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import { RoleTally } from "@/components/role/RoleTally";
import { cn } from "@/lib/utils";
import type { PermissionDomain } from "@/models/PermissionLevel";
import type { RoleChoice } from "@/models/RoleChoice";

interface RoleCheckRowProps {
  choice: RoleChoice;
  domains: PermissionDomain[];
  checked: boolean;
  onToggle: (roleId: string) => void;
}

// The whole row is the box's label; a role you can't give keeps its tick, can't change, and says why.
export const RoleCheckRow = ({ choice, domains, checked, onToggle }: RoleCheckRowProps) => {
  const { role, givable, reason } = choice;
  const id = `role-${role.id}`;
  return (
    <li className="flex min-h-11 items-center gap-3 py-2.5">
      <Checkbox id={id} checked={checked} disabled={!givable} onCheckedChange={() => onToggle(role.id)} />
      <label htmlFor={id} className={cn("min-w-0 flex-1", givable && "cursor-pointer")}>
        <span className={cn("block truncate text-sm font-medium", !givable && "text-muted-foreground")}>{role.name}</span>
        {role.description && <span className="block text-xs leading-4.5 text-muted-foreground">{role.description}</span>}
        {!givable && (
          <span className="mt-0.5 flex items-start gap-1.5 text-xs text-muted-foreground">
            <Lock aria-hidden className="mt-px size-3 shrink-0" />
            {reason}
          </span>
        )}
      </label>
      {givable && <RoleTally domains={domains} permissions={role.permissions} />}
    </li>
  );
};

import { Check } from "lucide-react";

import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";

interface RoleAbilityRowProps {
  ability: PermissionInfo;
  held: boolean;
}

// A yes-or-no permission: a check, or a muted No.
export const RoleAbilityRow = ({ ability, held }: RoleAbilityRowProps) => (
  <div className="flex min-h-9 items-center justify-between gap-3 border-b border-border py-1.5">
    <dt className={cn("min-w-0 truncate text-sm", !held && "text-muted-foreground")} title={ability.label}>
      {ability.label}
    </dt>
    <dd className="shrink-0 text-xs">
      {held && <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-label="Yes" />}
      {!held && <span className="text-muted-foreground">No</span>}
    </dd>
  </div>
);

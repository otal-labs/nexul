import { Check } from "lucide-react";

import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";

interface RoleAbilityRowProps {
  ability: PermissionInfo;
  held: boolean;
}

// A yes-or-no permission: a check and Yes in the level slot, or a muted No.
export const RoleAbilityRow = ({ ability, held }: RoleAbilityRowProps) => (
  <div className="flex min-h-10 break-inside-avoid items-baseline justify-between gap-3 border-b border-border py-2.5">
    <dt className={cn("min-w-0 truncate text-sm", !held && "text-muted-foreground")} title={ability.label}>
      {ability.label}
    </dt>
    <dd className={cn("flex w-20 shrink-0 items-baseline gap-2 text-xs", held ? "text-foreground" : "text-muted-foreground")}>
      {held && <Check aria-hidden className="size-3.5 self-center" strokeWidth={2.5} />}
      {held ? "Yes" : "No"}
    </dd>
  </div>
);

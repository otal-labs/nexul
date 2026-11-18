import { NoRolesChip, RoleChip } from "@/components/role/RoleChip";
import { cn } from "@/lib/utils";
import type { TeamRole } from "@/models/Team";

const SHOWN = 3;

// Below these widths of its line the row shows one chip fewer, so every chip it shows stays whole and the rest go to the count.
const HIDE_THIRD = "@max-[26rem]:hidden";
const HIDE_SECOND = "@max-[17rem]:hidden";

const Count = ({ roles, className }: { roles: TeamRole[]; className?: string }) => (
  <span className={cn("shrink-0 font-mono text-micro text-muted-foreground tabular-nums", className)} title={roles.map((role) => role.name).join(", ")}>
    +{roles.length}
  </span>
);

// A person's roles on one line, set on the line's baseline: as many whole chips as fit, up to three, then the rest as a mono
// count with every name in its tooltip. Only a lone chip too long for the line truncates.
export const RoleChipRow = ({ roles }: { roles: TeamRole[] }) => {
  const shown = roles.slice(0, SHOWN);
  return (
    <span className="@container flex min-w-0 flex-1 items-baseline gap-1 overflow-hidden">
      {roles.length === 0 && <NoRolesChip />}
      {shown.map((role, index) => (
        <span key={role.id} className={cn("flex min-w-0", index > 0 && "shrink-0", index === 1 && HIDE_SECOND, index === 2 && HIDE_THIRD)}>
          <RoleChip name={role.name} />
        </span>
      ))}
      {roles.length > 3 && <Count roles={roles.slice(3)} className="@max-[26rem]:hidden" />}
      {roles.length >= 3 && <Count roles={roles.slice(2)} className="hidden @max-[26rem]:inline @max-[17rem]:hidden" />}
      {roles.length >= 2 && <Count roles={roles.slice(1)} className="hidden @max-[17rem]:inline" />}
    </span>
  );
};

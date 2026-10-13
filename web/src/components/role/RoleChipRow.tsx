import { NoRolesChip, RoleChip } from "@/components/role/RoleChip";
import type { TeamRole } from "@/models/Team";

const SHOWN = 3;

// A person's roles on one line: three chips, then the rest as a mono count with every name in its tooltip.
export const RoleChipRow = ({ roles }: { roles: TeamRole[] }) => {
  const rest = roles.slice(SHOWN);
  return (
    <span className="flex min-w-0 items-center gap-1 overflow-hidden">
      {roles.length === 0 && <NoRolesChip />}
      {roles.slice(0, SHOWN).map((role) => (
        <RoleChip key={role.id} name={role.name} />
      ))}
      {rest.length > 0 && (
        <span className="shrink-0 font-mono text-[11px] text-muted-foreground tabular-nums" title={rest.map((role) => role.name).join(", ")}>
          +{rest.length}
        </span>
      )}
    </span>
  );
};

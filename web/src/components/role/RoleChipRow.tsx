import { NoRolesChip, RoleChip } from "@/components/role/RoleChip";
import { useFittingChildren } from "@/hooks/useFittingChildren";
import { cn } from "@/lib/utils";
import type { TeamRole } from "@/models/Team";

const SHOWN = 3;

// A person's roles on one line, set on the line's baseline: as many whole chips as the line holds, up to three, then the rest as
// a mono count with every hidden name in its tooltip. Only a first chip too long for the line truncates.
export const RoleChipRow = ({ roles }: { roles: TeamRole[] }) => {
  const candidates = roles.slice(0, SHOWN);
  // The chips wrap onto a clipped second line; only those on the first count as shown.
  const [ref, fitting, , end] = useFittingChildren<HTMLSpanElement>(candidates.length);
  const visible = candidates.slice(0, Math.max(1, fitting));
  const hidden = roles.filter((role) => !visible.includes(role));
  return (
    <span className="relative flex min-w-0 flex-1 items-baseline gap-1">
      {roles.length === 0 && <NoRolesChip />}
      {roles.length > 0 && (
        // The right padding keeps room for the count, which sits 4px after the last whole chip, not at the line's end.
        <span ref={ref} className={cn("relative flex h-5 min-w-0 flex-1 flex-wrap items-baseline gap-1 overflow-hidden", hidden.length > 0 && "pr-8")}>
          {candidates.map((role) => (
            <span key={role.id} className="flex min-w-0">
              <RoleChip name={role.name} className="max-w-full" />
            </span>
          ))}
        </span>
      )}
      {hidden.length > 0 && (
        <span
          className="absolute top-0 flex h-5 items-center pt-px font-mono text-micro text-muted-foreground tabular-nums"
          style={{ left: `calc(${end}px + 0.25rem)` }}
          title={hidden.map((role) => role.name).join(", ")}
        >
          +{hidden.length}
        </span>
      )}
    </span>
  );
};

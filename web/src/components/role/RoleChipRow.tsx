import { NoRolesChip, RoleChip } from "@/components/role/RoleChip";
import { useFittingChildren } from "@/hooks/useFittingChildren";
import type { TeamRole } from "@/models/Team";

const SHOWN = 3;
// A name longer than this can't show whole beside others, so it goes to the count instead.
const WHOLE = 22;

// A person's roles on one line, set on the line's baseline: as many whole chips as the line holds, up to three, then the rest as
// a mono count with every hidden name in its tooltip. Only a lone chip too long for the line truncates.
export const RoleChipRow = ({ roles }: { roles: TeamRole[] }) => {
  const whole = roles.filter((role) => role.name.length <= WHOLE);
  const candidates = (whole.length > 0 ? whole : roles).slice(0, SHOWN);
  // The chips wrap onto a clipped second line; only those on the first count as shown.
  const [ref, fitting] = useFittingChildren<HTMLSpanElement>(candidates.length);
  const visible = candidates.slice(0, Math.max(1, fitting));
  const hidden = roles.filter((role) => !visible.includes(role));
  return (
    <span className="flex min-w-0 flex-1 items-baseline gap-1">
      {roles.length === 0 && <NoRolesChip />}
      {roles.length > 0 && (
        <span ref={ref} className="relative flex h-5 min-w-0 flex-wrap items-baseline gap-1 overflow-hidden">
          {candidates.map((role) => (
            <span key={role.id} className="flex min-w-0">
              <RoleChip name={role.name} />
            </span>
          ))}
        </span>
      )}
      {hidden.length > 0 && (
        <span className="shrink-0 font-mono text-micro text-muted-foreground tabular-nums" title={hidden.map((role) => role.name).join(", ")}>
          +{hidden.length}
        </span>
      )}
    </span>
  );
};

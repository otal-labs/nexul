import { NoRolesChip, RoleChip } from "@/components/role/RoleChip";
import { cn } from "@/lib/utils";
import type { TeamRole } from "@/models/Team";

const SHOWN = 3;
// A name longer than this can't show whole beside others, so it goes to the count instead.
const WHOLE = 22;

// Each band of the line's width shows one chip fewer than the band above it.
const BANDS = [
  { visible: 3, chip: "", count: "@max-[26rem]:hidden" },
  { visible: 2, chip: "@max-[26rem]:hidden", count: "hidden @max-[26rem]:inline @max-[17rem]:hidden" },
  { visible: 1, chip: "@max-[17rem]:hidden", count: "hidden @max-[17rem]:inline" },
];

// A person's roles on one line, set on the line's baseline: as many whole chips as fit, up to three, then the rest as a mono
// count with every hidden name in its tooltip. Only a lone chip too long for the line truncates.
export const RoleChipRow = ({ roles }: { roles: TeamRole[] }) => {
  const whole = roles.filter((role) => role.name.length <= WHOLE);
  const shown = (whole.length > 0 ? whole : roles).slice(0, SHOWN);
  return (
    <span className="@container flex min-w-0 flex-1 items-baseline gap-1 overflow-hidden">
      {roles.length === 0 && <NoRolesChip />}
      {shown.map((role, index) => (
        <span key={role.id} className={cn("flex min-w-0", index > 0 && "shrink-0", index > 0 && BANDS[3 - index]?.chip)}>
          <RoleChip name={role.name} />
        </span>
      ))}
      {BANDS.map((band) => {
        const visible = shown.slice(0, band.visible);
        const hidden = roles.filter((role) => !visible.includes(role));
        return (
          hidden.length > 0 && (
            <span key={band.visible} className={cn("shrink-0 font-mono text-micro text-muted-foreground tabular-nums", band.count)} title={hidden.map((role) => role.name).join(", ")}>
              +{hidden.length}
            </span>
          )
        );
      })}
    </span>
  );
};

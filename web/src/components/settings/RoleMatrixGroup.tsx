import type { ReactNode } from "react";
import { Check } from "lucide-react";

import { LevelGlyph } from "@/components/access/LevelGlyph";
import { microheaderClass } from "@/components/Microheader";
import { fadeEndCell, fillerCell, leadStrip, nameText, stickyCell } from "@/components/accessgrid/gridStyles";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import { levelOf, type PermissionDomain } from "@/models/PermissionLevel";
import { extraName, rungName, type SectionDomains } from "@/models/RoleSection";

interface RoleMatrixGroupProps {
  title: string;
  domains: SectionDomains;
  // Each column's permissions, in column order.
  held: string[][];
}

const rowHead = cn(stickyCell, "h-9 border-b border-border pr-3 pl-6 text-left font-normal group-hover/row:bg-linear-to-r group-hover/row:from-accent/40 group-hover/row:to-accent/40");
const cellClass = "border-b border-border px-2 group-hover/row:bg-accent/40";

const MatrixRow = ({ label, children }: { label: string; children: ReactNode }) => (
  <tr className="group/row">
    <th scope="row" className={rowHead}>
      <span className="relative block h-5">
        <span className={nameText} title={label}>
          {label}
        </span>
      </span>
    </th>
    {children}
    <td aria-hidden className={cn(fillerCell, "group-hover/row:bg-accent/40")} />
    <td aria-hidden className={fadeEndCell} />
  </tr>
);

const AreaCell = ({ domain, permissions }: { domain: PermissionDomain; permissions: string[] }) => {
  const level = levelOf(domain, permissions);
  const extras = domain.extras.filter((extra) => permissions.includes(extra.value)).map((extra) => extraName(extra.action));
  return (
    <td className={cellClass}>
      <span className={cn("flex items-center gap-1.5 text-xs whitespace-nowrap", level === 0 ? "text-muted-foreground/70" : "text-foreground")}>
        <LevelGlyph level={level} rungs={domain.levels.length} />
        <span>
          {rungName(domain.domain, level)}
          {extras.length > 0 && (
            <span className="ml-1.5 font-mono text-[11px] text-muted-foreground" title={extras.join(", ")}>
              +{extras.length}
            </span>
          )}
        </span>
      </span>
    </td>
  );
};

const AbilityCell = ({ ability, permissions }: { ability: PermissionInfo; permissions: string[] }) => (
  <td className={cellClass}>
    {permissions.includes(ability.value) && <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-label="Yes" />}
    {!permissions.includes(ability.value) && <span className="text-xs text-muted-foreground/70">No</span>}
  </td>
);

// One of the editor's sections as a band, then its yes-or-no rows and its areas, each cell what that column's role holds.
export const RoleMatrixGroup = ({ title, domains, held }: RoleMatrixGroupProps) => (
  <tbody>
    <tr>
      <th scope="colgroup" colSpan={held.length + 3} className={cn(leadStrip, "h-8 p-0 text-left")}>
        <span className={cn(microheaderClass, "sticky left-0 inline-block pl-6")}>{title}</span>
      </th>
    </tr>
    {domains.abilities.map((ability) => (
      <MatrixRow key={ability.value} label={ability.label}>
        {held.map((permissions, column) => (
          <AbilityCell key={column} ability={ability} permissions={permissions} />
        ))}
      </MatrixRow>
    ))}
    {domains.areas.map((domain) => (
      <MatrixRow key={domain.domain} label={domain.name}>
        {held.map((permissions, column) => (
          <AreaCell key={column} domain={domain} permissions={permissions} />
        ))}
      </MatrixRow>
    ))}
  </tbody>
);

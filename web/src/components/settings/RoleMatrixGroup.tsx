import { useState, type ReactNode } from "react";
import { Check } from "lucide-react";

import { LevelGlyph } from "@/components/access/LevelGlyph";
import { microheaderClass } from "@/components/Microheader";
import { OverflowCount } from "@/components/OverflowCount";
import { RoleFoldButton } from "@/components/role/RoleFoldButton";
import { fadeEndCell, leadStrip, matrixAreaWidth, matrixFadeWidth, stickyCell } from "@/components/accessgrid/gridStyles";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import { levelOf, type PermissionDomain } from "@/models/PermissionLevel";
import { extraName, holdsArea, rungName, type SectionDomains } from "@/models/RoleSection";

interface RoleMatrixGroupProps {
  title: string;
  domains: SectionDomains;
  // Each column's permissions, in column order.
  held: string[][];
}

const rowHead = cn(stickyCell, matrixAreaWidth, "h-9 border-b border-border py-1.5 pr-3 pl-6 text-left font-normal group-hover/row:bg-linear-to-r group-hover/row:from-accent/40 group-hover/row:to-accent/40");
const cellClass = "border-b border-border px-2 group-hover/row:bg-accent/40";

const MatrixRow = ({ label, children }: { label: string; children: ReactNode }) => (
  <tr className="group/row">
    <th scope="row" className={rowHead}>
      <span className="line-clamp-3 -translate-y-px leading-tight text-balance lg:line-clamp-2" title={label}>
        {label}
      </span>
    </th>
    {children}
    <td aria-hidden className={cn(fadeEndCell, matrixFadeWidth, "border-b-transparent")} />
  </tr>
);

const AreaCell = ({ domain, permissions }: { domain: PermissionDomain; permissions: string[] }) => {
  const level = levelOf(domain, permissions);
  const extras = domain.extras.filter((extra) => permissions.includes(extra.value)).map((extra) => extraName(extra.action));
  const none = level === 0 && extras.length === 0;
  return (
    <td className={cellClass}>
      {none && <span aria-label="None" className="text-xs text-muted-foreground/70">&mdash;</span>}
      {!none && (
        <span className={cn("flex items-center gap-1.5 text-xs whitespace-nowrap", level === 0 ? "text-muted-foreground/70" : "text-foreground")}>
          <LevelGlyph level={level} rungs={domain.levels.length} />
          <span>
            {rungName(domain.domain, level)}
            {extras.length > 0 && (
              <OverflowCount
                label={`${extras.length} more ${extras.length === 1 ? "permission" : "permissions"}`}
                items={extras}
                className="ml-1.5 text-micro text-muted-foreground"
              />
            )}
          </span>
        </span>
      )}
    </td>
  );
};

const AbilityCell = ({ ability, permissions }: { ability: PermissionInfo; permissions: string[] }) => (
  <td className={cellClass}>
    {permissions.includes(ability.value) && <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-label="Yes" />}
    {!permissions.includes(ability.value) && <span aria-label="No" className="text-xs text-muted-foreground/70">&mdash;</span>}
  </td>
);

// One editor section: a band, then its rows, each cell what that column's role holds; rows no role holds fold under the band.
export const RoleMatrixGroup = ({ title, domains, held }: RoleMatrixGroupProps) => {
  const [showNone, setShowNone] = useState(false);
  const abilities = domains.abilities.filter((ability) => showNone || held.some((permissions) => permissions.includes(ability.value)));
  const areas = domains.areas.filter((domain) => showNone || held.some((permissions) => holdsArea(domain, permissions)));
  const none =
    domains.abilities.filter((ability) => !held.some((permissions) => permissions.includes(ability.value))).length +
    domains.areas.filter((domain) => !held.some((permissions) => holdsArea(domain, permissions))).length;
  return (
    <tbody>
      <tr>
        <th scope="colgroup" colSpan={held.length + 2} className={cn(leadStrip, "h-8 p-0 text-left")}>
          <span className="sticky left-0 inline-flex items-center gap-2 pl-6">
            <span className={microheaderClass}>{title}</span>
            {none > 0 && <RoleFoldButton open={showNone} onToggle={() => setShowNone(!showNone)} label={`${showNone ? "Hide" : "Show"} ${none} held by no role`} className="h-6 font-normal" />}
          </span>
        </th>
      </tr>
      {abilities.map((ability) => (
        <MatrixRow key={ability.value} label={ability.label}>
          {held.map((permissions, column) => (
            <AbilityCell key={column} ability={ability} permissions={permissions} />
          ))}
        </MatrixRow>
      ))}
      {areas.map((domain) => (
        <MatrixRow key={domain.domain} label={domain.name}>
          {held.map((permissions, column) => (
            <AreaCell key={column} domain={domain} permissions={permissions} />
          ))}
        </MatrixRow>
      ))}
    </tbody>
  );
};

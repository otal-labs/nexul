import { useState } from "react";

import { Microheader } from "@/components/Microheader";
import { RoleFoldButton } from "@/components/role/RoleFoldButton";
import { RoleAbilityRow } from "@/components/role/RoleAbilityRow";
import { RoleLevelRow } from "@/components/role/RoleLevelRow";
import { cn } from "@/lib/utils";
import { holdsArea, roleSectionId, type SectionDomains } from "@/models/RoleSection";
import type { PermissionArea } from "@/models/Permission";

interface RoleLevelsGroupProps {
  area: PermissionArea;
  title: string;
  domains: SectionDomains;
  permissions: string[];
}

// One of the role editor's sections, read-only: what the role holds, the areas it has no access to folded under one line,
// and a section it doesn't reach at all as that line alone.
export const RoleLevelsGroup = ({ area, title, domains, permissions }: RoleLevelsGroupProps) => {
  const [showNone, setShowNone] = useState(false);
  const heldAbilities = domains.abilities.filter((ability) => permissions.includes(ability.value));
  const heldAreas = domains.areas.filter((domain) => holdsArea(domain, permissions));
  const noneAbilities = domains.abilities.filter((ability) => !permissions.includes(ability.value));
  const noneAreas = domains.areas.filter((domain) => !holdsArea(domain, permissions));
  const none = noneAbilities.length + noneAreas.length;
  const held = heldAbilities.length + heldAreas.length;
  const rows = held > 0 || showNone;

  return (
    <section id={roleSectionId(area)} className="scroll-mt-4 space-y-1" aria-label={title}>
      <div className="flex min-h-8 items-center gap-2">
        <Microheader>{title}</Microheader>
        {held > 0 && (
          <span className="font-mono text-micro text-muted-foreground tabular-nums">
            {held}/{held + none}
          </span>
        )}
        {held === 0 && <span className="text-xs text-muted-foreground">No access</span>}
        {held === 0 && none > 0 && (
          <RoleFoldButton open={showNone} onToggle={() => setShowNone(!showNone)} label={`${showNone ? "Hide" : "Show"} ${none} areas`} />
        )}
      </div>
      {rows && (
        <dl className={cn("gap-x-8 border-t border-border", held + (showNone ? none : 0) > 1 && "@2xl:columns-2")}>
          {heldAbilities.map((ability) => (
            <RoleAbilityRow key={ability.value} ability={ability} held />
          ))}
          {heldAreas.map((domain) => (
            <RoleLevelRow key={domain.domain} domain={domain} permissions={permissions} />
          ))}
          {showNone && noneAbilities.map((ability) => <RoleAbilityRow key={ability.value} ability={ability} held={false} />)}
          {showNone && noneAreas.map((domain) => <RoleLevelRow key={domain.domain} domain={domain} permissions={permissions} />)}
        </dl>
      )}
      {held > 0 && none > 0 && (
        <RoleFoldButton open={showNone} onToggle={() => setShowNone(!showNone)} label={`${showNone ? "Hide" : "Show"} ${none} with no access`} className="-ml-2" />
      )}
    </section>
  );
};

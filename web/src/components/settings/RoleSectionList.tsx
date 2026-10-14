import { RoleAbilityRow } from "@/components/settings/RoleAbilityRow";
import { RoleAreaRow } from "@/components/settings/RoleAreaRow";
import { RoleBulkRow } from "@/components/settings/RoleBulkRow";
import { RoleLockedActionsRow } from "@/components/settings/RoleLockedActionsRow";
import { PermissionArea } from "@/models/Permission";
import type { RoleSection, SectionDomains } from "@/models/RoleSection";

interface RoleSectionListProps {
  section: RoleSection;
  line: string;
  domains: SectionDomains;
}

// One tab's body: where it applies, then one bordered list led by its bulk row.
export const RoleSectionList = ({ section, line, domains }: RoleSectionListProps) => (
  <div className="space-y-3">
    <p className="text-sm text-muted-foreground">{line}</p>
    {section.area === PermissionArea.Workspace && (
      <ul className="overflow-hidden rounded-md border border-input">
        <RoleLockedActionsRow />
      </ul>
    )}
    <ul className="@container divide-y divide-border overflow-hidden rounded-md border border-input">
      {domains.areas.length > 0 && <RoleBulkRow label={section.bulk} areas={domains.areas} />}
      {domains.abilities.map((ability) => (
        <RoleAbilityRow key={ability.value} ability={ability} />
      ))}
      {domains.areas.map((domain) => (
        <RoleAreaRow key={domain.domain} domain={domain} />
      ))}
    </ul>
  </div>
);

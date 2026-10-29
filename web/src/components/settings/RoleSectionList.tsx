import { RoleAbilityRow } from "@/components/settings/RoleAbilityRow";
import { RoleAreaRow } from "@/components/settings/RoleAreaRow";
import { RoleBulkRow } from "@/components/settings/RoleBulkRow";
import { RoleLockedActionsRow } from "@/components/settings/RoleLockedActionsRow";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { PermissionArea } from "@/models/Permission";
import type { RoleSection, SectionDomains } from "@/models/RoleSection";

interface RoleSectionListProps {
  section: RoleSection;
  line: string;
  domains: SectionDomains;
}

// One tab's body: where it applies, then its rows straight on the dialog, led by the bulk row.
export const RoleSectionList = ({ section, line, domains }: RoleSectionListProps) => (
  <div className="space-y-3 pb-6">
    <SettingsHint>{line}</SettingsHint>
    <ul className="@container -mx-6 divide-y divide-border border-y border-border">
      {section.area === PermissionArea.Workspace && <RoleLockedActionsRow />}
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

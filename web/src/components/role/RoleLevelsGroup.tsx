import { Microheader } from "@/components/Microheader";
import { RoleAbilityRow } from "@/components/role/RoleAbilityRow";
import { RoleLevelRow } from "@/components/role/RoleLevelRow";
import type { SectionDomains } from "@/models/RoleSection";

interface RoleLevelsGroupProps {
  title: string;
  domains: SectionDomains;
  permissions: string[];
}

// One of the role editor's sections, read-only, in two columns once the pane is wide enough.
export const RoleLevelsGroup = ({ title, domains, permissions }: RoleLevelsGroupProps) => (
  <section className="space-y-1" aria-label={title}>
    <Microheader>{title}</Microheader>
    <dl className="grid grid-cols-1 gap-x-8 @2xl:grid-cols-2">
      {domains.abilities.map((ability) => (
        <RoleAbilityRow key={ability.value} ability={ability} held={permissions.includes(ability.value)} />
      ))}
      {domains.areas.map((domain) => (
        <RoleLevelRow key={domain.domain} domain={domain} permissions={permissions} />
      ))}
    </dl>
  </section>
);

import { RoleSectionLink } from "@/components/role/RoleSectionLink";
import type { PermissionInfo } from "@/models/Permission";
import { levelOf } from "@/models/PermissionLevel";
import { holdsArea, ROLE_SECTIONS, sectionDomains } from "@/models/RoleSection";

interface RoleSectionLinksProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

// What the role reaches in each of the editor's sections, in the header, each jumping to its section below the holders.
export const RoleSectionLinks = ({ catalog, permissions }: RoleSectionLinksProps) => {
  const reach = ROLE_SECTIONS.map((section) => {
    const domains = sectionDomains(catalog, section.area);
    const held = domains.abilities.filter((ability) => permissions.includes(ability.value)).length + domains.areas.filter((domain) => holdsArea(domain, permissions)).length;
    const top = Math.max(0, ...domains.areas.map((domain) => levelOf(domain, permissions)));
    return { section, held, top, total: domains.abilities.length + domains.areas.length };
  });
  return (
    <nav aria-label="What it allows" className="-ml-2 flex flex-wrap items-center gap-x-1">
      {reach.map(({ section, held, top, total }) => (
        <RoleSectionLink key={section.area} section={section} held={held} top={top} total={total} />
      ))}
    </nav>
  );
};

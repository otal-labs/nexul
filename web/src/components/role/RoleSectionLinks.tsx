import { RoleSectionLink } from "@/components/role/RoleSectionLink";
import type { PermissionInfo } from "@/models/Permission";
import { roleReach } from "@/models/RoleSection";

interface RoleSectionLinksProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

// What the role reaches in each of the editor's sections, in the header, each jumping to its section below the holders.
export const RoleSectionLinks = ({ catalog, permissions }: RoleSectionLinksProps) => (
  <nav aria-label="What it allows" className="-ml-2 flex flex-wrap items-center gap-x-1">
    {roleReach(catalog, permissions).map(({ section, held, top, total }) => (
      <RoleSectionLink key={section.area} section={section} held={held} top={top} total={total} />
    ))}
  </nav>
);

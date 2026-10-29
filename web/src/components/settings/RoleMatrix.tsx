import { microheaderClass } from "@/components/Microheader";
import { fadeEndCell, fillerCell, snapScroller, stickyCell } from "@/components/accessgrid/gridStyles";
import { ScrollEndFade } from "@/components/ScrollEndFade";
import { RoleMatrixGroup } from "@/components/settings/RoleMatrixGroup";
import { RoleMatrixHead } from "@/components/settings/RoleMatrixHead";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import type { Role } from "@/models/Role";
import { ROLE_SECTIONS, sectionDomains } from "@/models/RoleSection";
import type { TeamPerson } from "@/models/Team";

interface RoleMatrixProps {
  roles: Role[];
  catalog: PermissionInfo[];
  workspaceId: string;
  holdersOf: (roleId: string) => TeamPerson[] | undefined;
}

// Every area down, every role across, read-only, in the editor's sections; each head holds its role's menu.
export const RoleMatrix = ({ roles, catalog, workspaceId, holdersOf }: RoleMatrixProps) => {
  const ordered = [...roles.filter((role) => role.is_owner_role), ...roles.filter((role) => !role.is_owner_role)];
  const everything = catalog.map((entry) => entry.value);
  const held = ordered.map((role) => (role.is_owner_role ? everything : role.permissions));
  return (
    <div className={cn("-mx-6 -mb-6 min-h-0 overflow-auto overscroll-contain [contain:paint] [scrollbar-gutter:stable]", snapScroller)}>
      <table className="w-full border-separate border-spacing-0 text-sm [&_tbody:last-child_tr:last-child>*]:border-b-0">
        <thead>
          <tr>
            <th scope="col" className={cn(stickyCell, microheaderClass, "top-0 z-30 border-b border-border pr-3 pb-[9px] pl-6 text-left align-bottom")}>
              Area
            </th>
            {ordered.map((role) => (
              <RoleMatrixHead key={role.id} role={role} workspaceId={workspaceId} holders={holdersOf(role.id)} />
            ))}
            <th aria-hidden className={cn(fillerCell, "sticky top-0 z-20 bg-card")} />
            <th aria-hidden className={cn(fadeEndCell, "top-0 z-20 bg-card")} />
          </tr>
        </thead>
        {ROLE_SECTIONS.map((section) => (
          <RoleMatrixGroup key={section.area} title={section.title} domains={sectionDomains(catalog, section.area)} held={held} />
        ))}
      </table>
      <ScrollEndFade />
    </div>
  );
};

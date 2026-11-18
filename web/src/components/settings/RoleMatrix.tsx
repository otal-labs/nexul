import { ChevronRight } from "lucide-react";

import { microheaderClass } from "@/components/Microheader";
import { fadeEndCell, matrixFadeWidth, snapScroller, stickyCell } from "@/components/accessgrid/gridStyles";
import { ScrollEndFade } from "@/components/ScrollEndFade";
import { RoleMatrixGroup } from "@/components/settings/RoleMatrixGroup";
import { RoleMatrixHead } from "@/components/settings/RoleMatrixHead";
import { useHiddenColumns } from "@/hooks/useHiddenColumns";
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
  // Off where the matrix grows with its page instead of scrolling inside a card, so no empty gutter runs down its right edge.
  stableGutter?: boolean;
}

// Every area down, every role across, read-only, in the editor's sections; each head holds its role's menu.
export const RoleMatrix = ({ roles, catalog, workspaceId, holdersOf, stableGutter = true }: RoleMatrixProps) => {
  const ordered = [...roles.filter((role) => role.is_owner_role), ...roles.filter((role) => !role.is_owner_role)];
  const everything = catalog.map((entry) => entry.value);
  const held = ordered.map((role) => (role.is_owner_role ? everything : role.permissions));
  // 48px is the end fade's width, which a column under it still counts as hidden behind.
  const { ref, hidden, more } = useHiddenColumns(48);
  return (
    <div ref={ref} className={cn("-mx-6 -mb-6 min-h-0 overflow-auto overscroll-contain [contain:paint]", stableGutter && "[scrollbar-gutter:stable]", snapScroller)}>
      {/* Fixed layout gives every role column one share of the width; the floor keeps each at least 96px, scrolling past it. */}
      <table style={{ minWidth: `calc(15rem + ${ordered.length} * 6rem + 3rem)` }} className="w-full table-fixed border-separate border-spacing-0 text-sm [&_tbody:last-child_tr:last-child>*]:border-b-0">
        <thead>
          <tr>
            <th scope="col" className={cn(stickyCell, microheaderClass, "top-0 z-30 border-b border-border pr-3 pb-[9px] pl-6 text-left align-bottom")}>
              Area
            </th>
            {ordered.map((role) => (
              <RoleMatrixHead key={role.id} role={role} workspaceId={workspaceId} holders={holdersOf(role.id)} />
            ))}
            <th className={cn(fadeEndCell, matrixFadeWidth, "top-0 z-20 border-b-transparent align-bottom")}>
              {hidden > 0 && (
                <button
                  type="button"
                  onClick={more}
                  aria-label={`Show ${hidden} more ${hidden === 1 ? "role" : "roles"}`}
                  className="relative z-10 mb-1.5 inline-flex h-6 items-center gap-0.5 rounded-md bg-card px-1 font-mono text-micro text-muted-foreground tabular-nums hover:bg-accent hover:text-foreground"
                >
                  +{hidden}
                  <ChevronRight aria-hidden className="size-3" />
                </button>
              )}
            </th>
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

import type { CSSProperties, ReactNode } from "react";
import { ChevronRight } from "lucide-react";

import { microheaderClass } from "@/components/Microheader";
import { fadeEndCell, matrixAreaWidth, matrixFadeWidth, snapScroller, stickyCell } from "@/components/accessgrid/gridStyles";
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
  // Leaves the Owner out where roles are compared: it holds everything, so its column tells nothing.
  hideOwner?: boolean;
  // A control beside "Area" in the head, such as which roles are shown.
  areaHead?: ReactNode;
}

// Every area down, every role across, read-only, in the editor's sections; each head holds its role's menu.
export const RoleMatrix = ({ roles, catalog, workspaceId, holdersOf, stableGutter = true, hideOwner = false, areaHead }: RoleMatrixProps) => {
  const ordered = [...(hideOwner ? [] : roles.filter((role) => role.is_owner_role)), ...roles.filter((role) => !role.is_owner_role)];
  const everything = catalog.map((entry) => entry.value);
  const held = ordered.map((role) => (role.is_owner_role ? everything : role.permissions));
  // 48px is the end fade's width, which a column under it still counts as hidden behind.
  const { ref, hidden, more } = useHiddenColumns(48, ordered.length);
  return (
    <div ref={ref} className={cn("-mx-6 -mb-6 min-h-0 overflow-auto overscroll-contain [contain:paint]", stableGutter && "[scrollbar-gutter:stable]", snapScroller, "scroll-ps-44")}>
      {/* Fixed layout gives every role column one share of the width; the floor keeps each at least 96px beside the area column, scrolling past it. */}
      <table style={{ "--role-columns": ordered.length } as CSSProperties} className="w-full min-w-[calc(11rem+var(--role-columns)*6rem+3rem)] table-fixed lg:min-w-[calc(15rem+var(--role-columns)*6rem+3rem)] border-separate border-spacing-0 text-sm [&_tbody:last-child_tr:last-child>*]:border-b-0">
        <thead>
          <tr>
            <th scope="col" className={cn(stickyCell, matrixAreaWidth, "top-0 z-30 border-b border-border pr-3 pb-1.5 pl-6 text-left align-bottom font-normal")}>
              <span className="flex h-6 items-center justify-between gap-2">
                <span className={microheaderClass}>Area</span>
                {areaHead}
              </span>
            </th>
            {ordered.map((role) => (
              <RoleMatrixHead key={role.id} role={role} workspaceId={workspaceId} holders={holdersOf(role.id)} />
            ))}
            {/* The heads fade under this slot like the cells do; its control sits above the names, clear of the counts and menus under it. */}
            <th className={cn(fadeEndCell, matrixFadeWidth, "top-0 z-20 border-b-transparent pt-2.5 align-top")}>
              {hidden > 0 && (
                <button
                  type="button"
                  onClick={more}
                  aria-label={`Show ${hidden} more ${hidden === 1 ? "role" : "roles"}`}
                  className="relative z-10 mr-3.5 inline-flex h-7 items-center gap-0.5 rounded-md border border-border bg-card px-1.5 font-mono text-xs text-foreground tabular-nums hover:bg-accent"
                >
                  +{hidden}
                  <ChevronRight aria-hidden className="size-3.5" />
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

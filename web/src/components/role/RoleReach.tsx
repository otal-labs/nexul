import { LevelGlyph } from "@/components/access/LevelGlyph";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import { levelName } from "@/models/PermissionLevel";
import { ROLE_SECTIONS, roleReach } from "@/models/RoleSection";

const reachLabel = (title: string, held: number, total: number, top: number): string => {
  if (held === 0) return `${title}: no access`;
  if (top === 0) return `${title}: ${held} of ${total}`;
  return `${title}: ${held} of ${total}, up to ${levelName(top)}`;
};

// The three section columns a picker's reach lines up under, so a section reads down the list.
export const reachColumns = "grid grid-cols-3 gap-x-3";

interface RoleReachProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

// A role's reach where roles are picked: the header's count and top level per section, in columns named once by RoleReachHead.
export const RoleReach = ({ catalog, permissions }: RoleReachProps) => (
  <span className={cn(reachColumns, "text-xs text-muted-foreground")}>
    {roleReach(catalog, permissions).map(({ section, held, total, top }) => (
      <span key={section.area} className="inline-flex min-w-0 items-center gap-1.5 whitespace-nowrap">
        <span className="sr-only">{reachLabel(section.title, held, total, top)}</span>
        {held === 0 && <span aria-hidden>No access</span>}
        {held > 0 && (
          <>
            <span aria-hidden className="font-mono text-foreground tabular-nums">{`${held}/${total}`}</span>
            {top > 0 && <LevelGlyph level={top} />}
            {top > 0 && <span aria-hidden className="text-foreground">{levelName(top)}</span>}
          </>
        )}
      </span>
    ))}
  </span>
);

// The section names over a picker's reach columns, once at the top of the list.
export const RoleReachHead = ({ className }: { className?: string }) => (
  <div aria-hidden className={cn(reachColumns, microheaderClass, className)}>
    {ROLE_SECTIONS.map((section) => (
      <span key={section.area}>{section.title}</span>
    ))}
  </div>
);

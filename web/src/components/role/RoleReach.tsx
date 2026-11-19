import { LevelGlyph } from "@/components/access/LevelGlyph";
import type { PermissionInfo } from "@/models/Permission";
import { levelName } from "@/models/PermissionLevel";
import { roleReach } from "@/models/RoleSection";

interface RoleReachProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

// A role's reach where roles are picked, the header's summary without its links: each section, its count and top level or No access.
export const RoleReach = ({ catalog, permissions }: RoleReachProps) => {
  const reach = roleReach(catalog, permissions);
  return (
    <span className="flex flex-wrap items-center gap-x-5 text-xs text-muted-foreground">
      {reach.map(({ section, held, total, top }) => (
        <span key={section.area} className="inline-flex items-center gap-1.5 whitespace-nowrap">
          {section.title}
          {held === 0 && <span>No access</span>}
          {held > 0 && <span className="font-mono text-foreground tabular-nums">{`${held}/${total}`}</span>}
          {top > 0 && (
            <span className="inline-flex items-center gap-1.5">
              up to
              <LevelGlyph level={top} />
              <span className="text-foreground">{levelName(top)}</span>
            </span>
          )}
        </span>
      ))}
    </span>
  );
};

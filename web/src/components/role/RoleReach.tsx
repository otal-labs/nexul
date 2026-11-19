import { LevelGlyph } from "@/components/access/LevelGlyph";
import type { PermissionInfo } from "@/models/Permission";
import { levelName } from "@/models/PermissionLevel";
import { roleReach } from "@/models/RoleSection";

interface RoleReachProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

// A role's reach on one line where roles are picked, the header's summary without its links: each section it reaches, its count and top level.
export const RoleReach = ({ catalog, permissions }: RoleReachProps) => {
  const reached = roleReach(catalog, permissions).filter(({ held }) => held > 0);
  return (
    <span className="flex flex-wrap items-center gap-x-5 text-xs text-muted-foreground">
      {reached.length === 0 && <span>No access</span>}
      {reached.map(({ section, held, total, top }) => (
        <span key={section.area} className="inline-flex items-center gap-1.5 whitespace-nowrap">
          {section.title}
          <span className="font-mono text-foreground tabular-nums">{`${held}/${total}`}</span>
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

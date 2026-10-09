import { LevelGlyph } from "@/components/access/LevelGlyph";
import type { PermissionInfo } from "@/models/Permission";
import { domainsOf, levelName, levelTally } from "@/models/PermissionLevel";

interface RoleAccessSummaryProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

const MISSING_SHOWN = 3;

// A role at a glance: how many areas sit at each level, highest first, and the first few it can't open at all.
export const RoleAccessSummary = ({ catalog, permissions }: RoleAccessSummaryProps) => {
  const { counts, missing } = levelTally(domainsOf(catalog), permissions);
  const levels = counts.map((count, level) => ({ count, level })).filter(({ count }) => count > 0).reverse();
  const more = missing.length - MISSING_SHOWN;
  return (
    <div className="space-y-1.5">
      <ul aria-label="Areas per level" className="flex flex-wrap items-center gap-x-4 gap-y-1">
        {levels.map(({ count, level }) => (
          <li key={level} className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <LevelGlyph level={level} />
            <span className="text-foreground/90">{levelName(level)}</span>
            <span className="font-mono tabular-nums">{count}</span>
          </li>
        ))}
      </ul>
      {missing.length > 0 && (
        <p className="truncate text-xs text-muted-foreground" title={missing.join(", ")}>
          <span className="text-foreground/80">No access</span> · {missing.slice(0, MISSING_SHOWN).join(", ")}
          {more > 0 && <span className="font-mono tabular-nums"> +{more}</span>}
        </p>
      )}
    </div>
  );
};

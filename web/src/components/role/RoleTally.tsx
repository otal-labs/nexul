import { LevelGlyph } from "@/components/access/LevelGlyph";
import { levelName, levelTally, type PermissionDomain } from "@/models/PermissionLevel";

interface RoleTallyProps {
  domains: PermissionDomain[];
  permissions: string[];
}

const LEVELS_SHOWN = 2;

// What a role is for, read from what it holds: the areas at its two highest levels, as the role rows tally them.
export const RoleTally = ({ domains, permissions }: RoleTallyProps) => {
  const { counts } = levelTally(domains, permissions);
  const levels = counts
    .map((count, level) => ({ count, level }))
    .filter(({ count, level }) => level > 0 && count > 0)
    .reverse()
    .slice(0, LEVELS_SHOWN);
  return (
    <span className="flex shrink-0 items-center gap-3 text-xs text-muted-foreground">
      {levels.map(({ count, level }) => (
        <span key={level} className="inline-flex items-center gap-1.5">
          <LevelGlyph level={level} />
          <span className="text-foreground/80">{levelName(level)}</span>
          <span className="font-mono tabular-nums">{count}</span>
        </span>
      ))}
    </span>
  );
};

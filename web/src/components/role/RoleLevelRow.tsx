import { LevelGlyph } from "@/components/access/LevelGlyph";
import { cn } from "@/lib/utils";
import { levelOf, type PermissionDomain } from "@/models/PermissionLevel";
import { extraName, rungName } from "@/models/RoleSection";

interface RoleLevelRowProps {
  domain: PermissionDomain;
  permissions: string[];
}

// One area as the role matrix reads it: the level as its glyph and word, muted at None, and the extra verbs held.
export const RoleLevelRow = ({ domain, permissions }: RoleLevelRowProps) => {
  const level = levelOf(domain, permissions);
  const extras = domain.extras.filter((extra) => permissions.includes(extra.value)).map((extra) => extraName(extra.action));
  return (
    <div className="flex min-h-9 items-center justify-between gap-3 border-b border-border py-1.5">
      <dt className={cn("min-w-0 truncate text-sm", level === 0 && "text-muted-foreground")} title={domain.name}>
        {domain.name}
      </dt>
      <dd className={cn("flex shrink-0 items-center gap-1.5 text-xs", level === 0 ? "text-muted-foreground" : "text-foreground")}>
        <LevelGlyph level={level} rungs={domain.levels.length} />
        {rungName(domain.domain, level)}
        {extras.length > 0 && <span className="text-muted-foreground">+ {extras.join(", ")}</span>}
      </dd>
    </div>
  );
};

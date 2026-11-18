import { LevelGlyph } from "@/components/access/LevelGlyph";
import { cn } from "@/lib/utils";
import { levelOf, type PermissionDomain } from "@/models/PermissionLevel";
import { extraName, rungName } from "@/models/RoleSection";

interface RoleLevelRowProps {
  domain: PermissionDomain;
  permissions: string[];
}

// One area: its name, the extra verbs held on a line under it, and its level as the matrix draws it, glyph then word, in a fixed slot on the name's baseline.
export const RoleLevelRow = ({ domain, permissions }: RoleLevelRowProps) => {
  const level = levelOf(domain, permissions);
  const extras = domain.extras.filter((extra) => permissions.includes(extra.value)).map((extra) => extraName(extra.action));
  return (
    <div className="flex min-h-10 break-inside-avoid items-baseline justify-between gap-3 border-b border-border py-2.5">
      <dt className="min-w-0">
        <span className={cn("block truncate text-sm", level === 0 && "text-muted-foreground")} title={domain.name}>
          {domain.name}
        </span>
        {extras.length > 0 && <span className="block truncate text-xs text-muted-foreground">Also {extras.join(", ")}</span>}
      </dt>
      <dd className={cn("w-20 shrink-0 text-xs whitespace-nowrap", level === 0 ? "text-muted-foreground" : "text-foreground")}>
        <LevelGlyph level={level} rungs={domain.levels.length} className="mr-1.5 -translate-y-px align-middle" />
        {rungName(domain.domain, level)}
      </dd>
    </div>
  );
};

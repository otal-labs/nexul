import { LevelGlyph } from "@/components/access/LevelGlyph";
import { Microheader } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import { PermissionArea, type PermissionInfo } from "@/models/Permission";
import { domainsOf, levelName, levelOf, type PermissionDomain } from "@/models/PermissionLevel";

interface RoleAccessDetailProps {
  catalog: PermissionInfo[];
  permissions: string[];
}

interface DomainLineProps {
  domain: PermissionDomain;
  permissions: string[];
}

const DomainLine = ({ domain, permissions }: DomainLineProps) => {
  const level = levelOf(domain, permissions);
  const extras = domain.extras.filter((extra) => permissions.includes(extra.value)).map((extra) => extra.action);
  return (
    <li className={cn("flex items-center gap-2 py-1 text-xs", level === 0 && "text-muted-foreground")}>
      <span className="min-w-0 flex-1 truncate" title={domain.name}>
        {domain.name}
      </span>
      {extras.length > 0 && <span className="truncate font-mono text-[11px] text-muted-foreground">+{extras.join(" +")}</span>}
      <LevelGlyph level={level} rungs={domain.levels.length} />
      <span className="w-12 shrink-0 text-muted-foreground">{levelName(level)}</span>
    </li>
  );
};

// Every domain of a role, read-only, in the editor's two groups; the summary above is the short version.
export const RoleAccessDetail = ({ catalog, permissions }: RoleAccessDetailProps) => {
  const groups = [
    { label: "Workspace", domains: domainsOf(catalog.filter((entry) => entry.area !== PermissionArea.Project)) },
    { label: "Every project", domains: domainsOf(catalog.filter((entry) => entry.area === PermissionArea.Project)) },
  ];
  return (
    <div className="@container grid gap-5 pt-1">
      {groups.map((group) => (
        <section key={group.label} className="space-y-1.5">
          <Microheader>{group.label}</Microheader>
          <ul className="grid gap-x-8 @xl:grid-cols-2">
            {group.domains.map((domain) => (
              <DomainLine key={domain.domain} domain={domain} permissions={permissions} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
};

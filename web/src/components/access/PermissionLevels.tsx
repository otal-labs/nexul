import { PermissionDomainRow } from "@/components/access/PermissionDomainRow";
import { PermissionLevelControl } from "@/components/access/PermissionLevelControl";
import type { PermissionInfo } from "@/models/Permission";
import { domainsOf, uniformLevelOf, withLevelEverywhere } from "@/models/PermissionLevel";

interface PermissionLevelsProps {
  entries: PermissionInfo[];
  value: string[];
  onChange: (value: string[]) => void;
  // The top row that sets every domain at once ("Every domain", "Every area").
  everyLabel?: string;
}

// One access level per domain plus a row that sets them all; extra verbs (run, clone, thread) sit beside the level.
export const PermissionLevels = ({ entries, value, onChange, everyLabel = "Every domain" }: PermissionLevelsProps) => {
  const domains = domainsOf(entries);
  const top = Math.max(0, ...domains.map((domain) => domain.levels.length));

  return (
    <ul className="@container divide-y divide-border overflow-hidden rounded-md border border-input">
      <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 bg-muted/40 px-2 py-2 @md:px-3 @md:grid-cols-[minmax(0,1fr)_auto_auto]">
        <span className="truncate text-sm font-medium">{everyLabel}</span>
        <span />
        <PermissionLevelControl
          label={`${everyLabel} access`}
          levelCount={top}
          level={uniformLevelOf(domains, value)}
          onLevel={(level) => onChange(withLevelEverywhere(value, domains, level))}
        />
      </li>
      {domains.map((domain) => (
        <PermissionDomainRow key={domain.domain} domain={domain} value={value} onChange={onChange} />
      ))}
    </ul>
  );
};

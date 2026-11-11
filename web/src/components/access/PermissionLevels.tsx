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

// One access level per domain plus a row that sets them all.
export const PermissionLevels = ({ entries, value, onChange, everyLabel = "Every domain" }: PermissionLevelsProps) => {
  const domains = domainsOf(entries);
  const top = Math.max(0, ...domains.map((domain) => domain.levels.length));

  // The level strips query an @container the caller provides: one gaining a next sibling (a form error) collapses to 0px in Chromium.
  return (
    <ul className="divide-y divide-border overflow-hidden rounded-md border border-input">
      <li className="flex items-center gap-3 bg-muted/40 px-3 py-2">
        <span className="min-w-0 flex-1 truncate text-sm font-medium">{everyLabel}</span>
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

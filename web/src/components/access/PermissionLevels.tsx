import { LevelDropdown } from "@/components/access/LevelDropdown";
import { LevelRow } from "@/components/access/LevelRow";
import { PermissionDomainRow } from "@/components/access/PermissionDomainRow";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import { domainsOf, uniformLevelOf, withLevelEverywhere } from "@/models/PermissionLevel";

interface PermissionLevelsProps {
  entries: PermissionInfo[];
  value: string[];
  onChange: (value: string[]) => void;
  // The top row that sets every domain at once ("Every domain", "Every area").
  everyLabel?: string;
  disabled?: boolean;
  className?: string;
}

// One hairline row per domain, led by a row that sets them all; it reads Custom once the rows differ.
export const PermissionLevels = ({ entries, value, onChange, everyLabel = "Every domain", disabled = false, className }: PermissionLevelsProps) => {
  const domains = domainsOf(entries);
  const top = Math.max(0, ...domains.map((domain) => domain.levels.length));
  const uniform = uniformLevelOf(domains, value);
  return (
    <ul className={cn("divide-y divide-border", className)}>
      <LevelRow name={everyLabel} strong>
        <LevelDropdown
          label={everyLabel}
          levelCount={top}
          level={uniform}
          display={uniform === undefined ? "Custom" : undefined}
          disabled={disabled}
          onLevel={(level) => onChange(withLevelEverywhere(value, domains, level))}
        />
      </LevelRow>
      {domains.map((domain) => (
        <PermissionDomainRow key={domain.domain} domain={domain} value={value} onChange={onChange} disabled={disabled} />
      ))}
    </ul>
  );
};

import { LevelDropdown } from "@/components/access/LevelDropdown";
import { LevelRow } from "@/components/access/LevelRow";
import { DropdownMenuCheckboxItem, DropdownMenuLabel, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { levelLabel, levelOf, withExtra, withLevel, type PermissionDomain } from "@/models/PermissionLevel";

interface PermissionDomainRowProps {
  domain: PermissionDomain;
  value: string[];
  onChange: (value: string[]) => void;
  disabled?: boolean;
}

const capitalize = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1);

// The domain's verbs read in the value ("Write + Clone") and are ticked under "Also allow" in the same menu.
export const PermissionDomainRow = ({ domain, value, onChange, disabled = false }: PermissionDomainRowProps) => {
  const level = levelOf(domain, value);
  const hasExtras = domain.extras.length > 0;
  return (
    <LevelRow name={domain.name}>
      <LevelDropdown
        label={`${domain.name} access`}
        levelCount={domain.levels.length}
        level={level}
        display={levelLabel(domain, value)}
        disabled={disabled}
        onLevel={(next) => onChange(withLevel(value, domain, next))}
      >
        {hasExtras && <DropdownMenuSeparator />}
        {hasExtras && (
          <DropdownMenuLabel className="font-mono text-[11px] font-normal tracking-wide text-muted-foreground uppercase">
            Also allow
          </DropdownMenuLabel>
        )}
        {domain.extras.map((extra) => (
          <DropdownMenuCheckboxItem
            key={extra.value}
            checked={value.includes(extra.value)}
            onSelect={(event) => event.preventDefault()}
            onCheckedChange={(on) => onChange(withExtra(value, extra, on === true))}
          >
            <span className="flex flex-col">
              <span>{capitalize(extra.action)}</span>
              <span className="text-xs text-muted-foreground">{extra.label}</span>
            </span>
          </DropdownMenuCheckboxItem>
        ))}
      </LevelDropdown>
    </LevelRow>
  );
};

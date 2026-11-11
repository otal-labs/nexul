import { PermissionLevelControl } from "@/components/access/PermissionLevelControl";
import { Checkbox } from "@/components/ui/checkbox";
import { usePermissionLock } from "@/hooks/usePermissionLock";
import { cn } from "@/lib/utils";
import { levelOf, withExtra, withLevel, type PermissionDomain } from "@/models/PermissionLevel";
import { extraName, readName } from "@/models/RoleSection";

interface PermissionDomainRowProps {
  domain: PermissionDomain;
  value: string[];
  onChange: (value: string[]) => void;
  className?: string;
}

// One area: its name and level strip, and its extra verbs on a line of their own so they never squeeze the name.
export const PermissionDomainRow = ({ domain, value, onChange, className }: PermissionDomainRowProps) => {
  const locked = usePermissionLock();
  const level = levelOf(domain, value);
  const firstRung = readName(domain.domain);

  return (
    <li className={cn("px-3 py-2", className)}>
      <div className="flex min-h-7 items-center gap-3">
        <span className={cn("min-w-0 flex-1 truncate text-sm", level === 0 && "text-muted-foreground")} title={domain.name}>
          {domain.name}
        </span>
        <PermissionLevelControl
          label={`${domain.name} access`}
          levelCount={domain.levels.length}
          level={level}
          readName={firstRung}
          disabled={locked}
          onLevel={(next) => onChange(withLevel(value, domain, next))}
        />
      </div>
      {domain.extras.length > 0 && (
        <div className="mt-1.5 ml-1 flex flex-wrap items-center gap-x-4 gap-y-1.5 border-l border-border pb-0.5 pl-4">
          {domain.extras.map((extra) => (
            <label
              key={extra.value}
              title={level === 0 ? `Needs ${firstRung} first` : extra.label}
              className={cn("flex items-center gap-2 text-xs text-muted-foreground", level === 0 || locked ? "cursor-not-allowed" : "cursor-pointer")}
            >
              <Checkbox
                aria-label={extra.label}
                disabled={level === 0 || locked}
                checked={value.includes(extra.value)}
                onCheckedChange={(on) => onChange(withExtra(value, extra, on === true))}
              />
              {extraName(extra.action)}
            </label>
          ))}
        </div>
      )}
    </li>
  );
};

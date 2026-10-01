import { PermissionLevelControl } from "@/components/access/PermissionLevelControl";
import { Toggle } from "@/components/ui/toggle";
import { usePermissionLock } from "@/hooks/usePermissionLock";
import { levelOf, withExtra, withLevel, type PermissionDomain } from "@/models/PermissionLevel";

interface PermissionDomainRowProps {
  domain: PermissionDomain;
  value: string[];
  onChange: (value: string[]) => void;
}

export const PermissionDomainRow = ({ domain, value, onChange }: PermissionDomainRowProps) => {
  const disabled = usePermissionLock();
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 px-2 py-2 @md:px-3 @md:grid-cols-[minmax(0,1fr)_auto_auto]">
      <span className="truncate text-sm">{domain.name}</span>
      <div className="flex items-center gap-1.5">
        {domain.extras.map((extra) => (
          <Toggle
            key={extra.value}
            variant="outline"
            size="sm"
            aria-label={extra.label}
            title={extra.label}
            disabled={disabled}
            pressed={value.includes(extra.value)}
            onPressedChange={(on) => onChange(withExtra(value, extra, on))}
            className="h-7 px-2 text-xs capitalize text-muted-foreground data-[state=on]:bg-accent data-[state=on]:text-foreground"
          >
            {extra.action}
          </Toggle>
        ))}
      </div>
      <PermissionLevelControl
        label={`${domain.name} access`}
        levelCount={domain.levels.length}
        level={levelOf(domain, value)}
        disabled={disabled}
        onLevel={(level) => onChange(withLevel(value, domain, level))}
      />
    </li>
  );
};

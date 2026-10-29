import { Checkbox } from "@/components/ui/checkbox";
import { PermissionLevelControl } from "@/components/access/PermissionLevelControl";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import { levelOf, withExtra, withLevel, type PermissionDomain } from "@/models/PermissionLevel";
import { extraName, readName } from "@/models/RoleSection";
import { cn } from "@/lib/utils";

interface RoleAreaRowProps {
  domain: PermissionDomain;
}

// One area: its name and level strip, and its extra verbs on a line of their own so they aren't missed.
export const RoleAreaRow = ({ domain }: RoleAreaRowProps) => {
  const { actions, setActions } = useRoleDraft();
  const level = levelOf(domain, actions);
  const firstRung = readName(domain.domain);

  return (
    <li className="px-6 py-2">
      <div className="flex min-h-7 items-center gap-3">
        <span className={cn("min-w-0 flex-1 truncate text-sm", level === 0 && "text-muted-foreground")} title={domain.name}>
          {domain.name}
        </span>
        <PermissionLevelControl
          label={`${domain.name} access`}
          levelCount={domain.levels.length}
          level={level}
          readName={firstRung}
          onLevel={(next) => setActions(withLevel(actions, domain, next))}
        />
      </div>
      {domain.extras.length > 0 && (
        <div className="mt-1.5 ml-1 flex flex-wrap items-center gap-x-4 gap-y-1.5 border-l border-border pb-0.5 pl-4">
          {domain.extras.map((extra) => (
            <label
              key={extra.value}
              title={level === 0 ? `Needs ${firstRung} first` : extra.label}
              className={cn(
                "flex items-center gap-2 text-xs text-muted-foreground",
                level === 0 ? "cursor-not-allowed opacity-70" : "cursor-pointer",
              )}
            >
              <Checkbox
                aria-label={extra.label}
                disabled={level === 0}
                checked={actions.includes(extra.value)}
                onCheckedChange={(on) => setActions(withExtra(actions, extra, on === true))}
              />
              {extraName(extra.action)}
            </label>
          ))}
        </div>
      )}
    </li>
  );
};

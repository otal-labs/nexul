import { LevelDropdown } from "@/components/access/LevelDropdown";
import { LevelRow } from "@/components/access/LevelRow";
import { PermissionDomainRow } from "@/components/access/PermissionDomainRow";
import { DropdownMenuItem, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { useProjectAreas } from "@/hooks/PermissionHooks";
import { accessSummary, projectLevelLabel, uniformLevelOf, withLevelEverywhere } from "@/models/PermissionLevel";

interface ProjectAccessRowProps {
  name: string;
  value: string[];
  onChange: (value: string[]) => void;
  // Whether this project's area rows are open under it; one project at a time.
  customizing: boolean;
  onCustomize: (open: boolean) => void;
  disabled?: boolean;
}

// One dropdown for the whole project; its areas open inline under it, each with its own level.
export const ProjectAccessRow = ({ name, value, onChange, customizing, onCustomize, disabled = false }: ProjectAccessRowProps) => {
  const areas = useProjectAreas();
  const top = Math.max(0, ...areas.map((area) => area.levels.length));
  const label = projectLevelLabel(areas, value);
  const held = value.length > 0;

  return (
    <li>
      <ul>
        <LevelRow name={name} meta={label === "Custom" ? accessSummary(areas, value) : undefined}>
          <LevelDropdown
            label={`${name} access`}
            levelCount={top}
            level={uniformLevelOf(areas, value)}
            display={label}
            disabled={disabled}
            onLevel={(level) => onChange(withLevelEverywhere(value, areas, level))}
          >
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => onCustomize(!customizing)}>{customizing ? "Hide areas" : "Customize areas…"}</DropdownMenuItem>
            {held && <DropdownMenuSeparator />}
            {held && (
              <DropdownMenuItem variant="destructive" onSelect={() => onChange([])}>
                Remove access
              </DropdownMenuItem>
            )}
          </LevelDropdown>
        </LevelRow>
      </ul>
      {customizing && (
        <div className="animate-in fade-in-0 slide-in-from-top-1 border-t border-border pl-5 duration-150 ease-out">
          <ul className="divide-y divide-border">
            {areas.map((area) => (
              <PermissionDomainRow key={area.domain} domain={area} value={value} onChange={onChange} disabled={disabled} />
            ))}
          </ul>
          <button
            type="button"
            className="mb-2 rounded-sm text-sm text-muted-foreground underline-offset-4 outline-none hover:text-foreground hover:underline focus-visible:ring-[3px] focus-visible:ring-ring/30"
            onClick={() => onCustomize(false)}
          >
            Hide areas
          </button>
        </div>
      )}
    </li>
  );
};

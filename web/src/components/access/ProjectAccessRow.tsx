import { useState } from "react";
import { ChevronDownIcon } from "lucide-react";

import { PermissionDomainRow } from "@/components/access/PermissionDomainRow";
import { PermissionLevelControl } from "@/components/access/PermissionLevelControl";
import { useProjectAreas } from "@/hooks/PermissionHooks";
import { usePermissionLock } from "@/hooks/usePermissionLock";
import { cn } from "@/lib/utils";
import { accessSummary, uniformLevelOf, withLevelEverywhere } from "@/models/PermissionLevel";

interface ProjectAccessRowProps {
  name: string;
  value: string[];
  onChange: (value: string[]) => void;
}

// One level for the whole project, like a domain row; its areas open underneath, each with its own level.
export const ProjectAccessRow = ({ name, value, onChange }: ProjectAccessRowProps) => {
  const areas = useProjectAreas();
  const disabled = usePermissionLock();
  const [open, setOpen] = useState(false);
  const top = Math.max(0, ...areas.map((area) => area.levels.length));
  const uniform = uniformLevelOf(areas, value);

  return (
    <li>
      <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 px-2 py-2 @md:px-3 @md:grid-cols-[minmax(0,1fr)_auto_auto]">
        <div className="min-w-0">
          <p className="truncate text-sm">{name}</p>
          {uniform === undefined && <p className="truncate font-mono text-xs text-muted-foreground">{accessSummary(areas, value)}</p>}
        </div>
        <button
          type="button"
          aria-label={`Areas of ${name}`}
          aria-expanded={open}
          onClick={() => setOpen(!open)}
          className="flex h-7 items-center gap-1 rounded-md px-2 text-xs text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/30"
        >
          Areas
          <ChevronDownIcon className={cn("size-3.5 transition-transform duration-150 ease-standard", open && "rotate-180")} aria-hidden />
        </button>
        <PermissionLevelControl
          label={`${name} access`}
          levelCount={top}
          level={uniform}
          disabled={disabled}
          onLevel={(level) => onChange(withLevelEverywhere(value, areas, level))}
        />
      </div>
      {open && (
        <ul className="animate-in fade-in-0 slide-in-from-top-1 divide-y divide-border border-t border-border pl-4 duration-150 ease-out">
          {areas.map((area) => (
            <PermissionDomainRow key={area.domain} domain={area} value={value} onChange={onChange} />
          ))}
        </ul>
      )}
    </li>
  );
};

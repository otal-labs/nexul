import { CheckIcon, PlusIcon } from "lucide-react";

import { PopoverContent } from "@/components/ui/popover";
import type { Project } from "@/models/Project";

const menuItemClass =
  "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:bg-accent/60 focus-visible:text-foreground";

interface ProjectSwitcherMenuProps {
  projects: Project[];
  currentId: string;
  onSelect: (project: Project) => void;
  onCreate: () => void;
}

export const ProjectSwitcherMenu = ({ projects, currentId, onSelect, onCreate }: ProjectSwitcherMenuProps) => (
  <PopoverContent side="bottom" align="start" sideOffset={6} className="w-56 p-1.5">
    <div className="flex max-h-80 flex-col gap-0.5 overflow-y-auto">
      {projects.map((project) => (
        <button key={project.id} type="button" onClick={() => onSelect(project)} className={menuItemClass}>
          <span className="w-9 shrink-0 font-mono text-[11px] font-semibold">{project.prefix}</span>
          <span className="flex-1 truncate">{project.name}</span>
          {project.id === currentId && <CheckIcon className="size-3.5 shrink-0 text-primary" aria-hidden />}
        </button>
      ))}
    </div>
    <div className="my-1 border-t border-border" />
    <button type="button" onClick={onCreate} className={menuItemClass}>
      <PlusIcon className="size-4 shrink-0" aria-hidden />
      <span>New project</span>
    </button>
  </PopoverContent>
);

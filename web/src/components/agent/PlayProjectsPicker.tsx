import { ChevronDownIcon } from "lucide-react";

import { menuItemClass } from "@/components/MenuItem";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchProjects } from "@/hooks/ProjectHooks";

interface PlayProjectsPickerProps {
  excluded: string[];
  onChange: (excluded: string[]) => void;
  disabled: boolean;
}

const summary = (shown: number, total: number) => {
  if (total === 0) return "No projects yet";
  if (shown === total) return `All ${total} projects`;
  if (shown === 0) return "No projects";
  return `${shown} of ${total} projects`;
};

// The projects the play's button shows in; the play stores the ones left out.
export const PlayProjectsPicker = ({ excluded, onChange, disabled }: PlayProjectsPickerProps) => {
  const { data: projects } = useFetchProjects();
  const all = projects ?? [];
  const shown = all.filter((project) => !excluded.includes(project.id));
  const toggle = (id: string) => onChange(excluded.includes(id) ? excluded.filter((value) => value !== id) : [...excluded, id]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Projects"
          disabled={disabled}
          className="flex h-8 w-full max-w-xs items-center justify-between gap-1.5 rounded-md border border-input px-2.5 text-left text-sm shadow-xs transition-[border-color] duration-150 ease-standard hover:border-foreground/25 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/20"
        >
          <span className="truncate">{summary(shown.length, all.length)}</span>
          <ChevronDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="max-h-72 w-64 overflow-y-auto p-1">
        {all.map((project) => (
          <label key={project.id} className={`${menuItemClass} cursor-pointer`}>
            <Checkbox checked={!excluded.includes(project.id)} onCheckedChange={() => toggle(project.id)} />
            <span className="truncate">{project.name}</span>
          </label>
        ))}
      </PopoverContent>
    </Popover>
  );
};
